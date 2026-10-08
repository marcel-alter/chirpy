package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/marcel-alter/chirpy.git/internal/database"
)

// ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓ JSON STRUCTS
type errorResponse struct {
	Error string `json:"body"`
}

type validResponse struct {
	Body      string `json:"body"`
	CleanBody string `json:"cleaned_body"`
}

type RequestBody struct {
	Body    string    `json:"body"`
	Message string    `json:"message"`
	Email   string    `json:"email"`
	UserID  uuid.UUID `json:"user_id"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

// ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓ HELPER FUNCTIONS
func decodeJSON(r *http.Request) (RequestBody, error) {
	decoder := json.NewDecoder(r.Body)
	req := RequestBody{}
	if err := decoder.Decode(&req); err != nil {
		return RequestBody{}, fmt.Errorf("Error decoding RequestJsonBody: %s", err)
	}
	return req, nil
}
func respondWithError(w http.ResponseWriter, code int, msg string) {
	respBody := errorResponse{
		Error: msg,
	}
	data, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) { //Explain payload interface{} in detail!
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}
func isLetterAt(text string, index int) bool {
	if index < 0 || index >= len(text) {
		return false
	}
	r, _, _ := strings.NewReader(text[index:]).ReadRune()
	return unicode.IsLetter(r)
}

func profanityFilter(text, search, replacement string) string {
	if search == "" {
		return text
	}

	lowerText := strings.ToLower(text)
	lowerSearch := strings.ToLower(search)

	var result strings.Builder
	result.Grow(len(text))

	lastIndex := 0

	for {
		matchIndex := strings.Index(lowerText[lastIndex:], lowerSearch)
		if matchIndex == -1 {
			break
		}

		absoluteMatchIndex := lastIndex + matchIndex
		endIndex := absoluteMatchIndex + len(search)

		hasLetterBefore := isLetterAt(text, absoluteMatchIndex-1)
		hasLetterAfter := isLetterAt(text, endIndex)

		if hasLetterBefore || hasLetterAfter {
			result.WriteString(text[lastIndex:endIndex])
			lastIndex = endIndex
			continue
		}

		result.WriteString(text[lastIndex:absoluteMatchIndex])
		result.WriteString(replacement)
		lastIndex = endIndex
	}

	result.WriteString(text[lastIndex:])
	return result.String()
}

// ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓ HANDLERS
func handlerStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	htmlText := fmt.Sprintf(
		`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`,
		cfg.fileserverHits.Load())
	w.Write([]byte(htmlText))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Reset is only allowed in dev environment."))
		return
	}

	cfg.fileserverHits.Store(0)
	err := cfg.dbQueries.RemoveAllUsers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to reset the database: " + err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0 and database reset to initial state."))
}

func (cfg *apiConfig) handlerAddChirp(w http.ResponseWriter, r *http.Request) { //is there a special reason, why http.Handlers aren't allowed to return errors?
	profanities := []string{"kerfuffle", "sharbert", "fornax"}
	req, err := decodeJSON(r)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		log.Printf("Error using decodeJson in handlerAddChirp! Error: %s", err)
		return
	}
	if len(req.Body) > 141 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	for _, prof := range profanities {
		req.Body = profanityFilter(req.Body, prof, "****")
	}
	arg := database.CreateChirpParams{
		ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		Body:      req.Body,
		UserID:    pgtype.UUID{Bytes: req.UserID, Valid: true},
	}
	chirp, err := cfg.dbQueries.CreateChirp(r.Context(), arg)
	if err != nil {
		log.Printf("Something went wrong in handlerAddChirp! Failed to save User to database! Error: %v", err)
	}
	respondWithJSON(w, 201, Chirp{chirp.ID.Bytes, chirp.CreatedAt.Time, chirp.UpdatedAt.Time, chirp.Body, chirp.UserID.Bytes})
}

func (cfg *apiConfig) handlerPostUser(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON(r)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		log.Printf("Error using decodeJson in handlerPostUser! Error: %s", err)
		return
	}
	args := database.CreateUserParams{
		ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		Email:     req.Email,
	}
	queryUser, err := cfg.dbQueries.CreateUser(r.Context(), args)
	if err != nil {
		log.Printf("Something went wrong in handlerPostUser trying to create user! Error: %v", err)
	}
	user := User{
		ID:        queryUser.ID.Bytes,
		CreatedAt: queryUser.CreatedAt.Time,
		UpdatedAt: queryUser.CreatedAt.Time,
		Email:     queryUser.Email,
	}
	respondWithJSON(w, 201, user)
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	SQLChirps, err := cfg.dbQueries.ReturnAllChirps(r.Context())
	if err != nil {
		log.Printf("Something went wrong in handlerGetAllChirps, trying ReturnAllChirps! Error: %v", err)
	}
	var chirps []Chirp
	for _, chirp := range SQLChirps {
		value := Chirp{
			ID:        chirp.ID.Bytes,
			CreatedAt: chirp.CreatedAt.Time,
			UpdatedAt: chirp.CreatedAt.Time,
			Body:      chirp.Body,
			UserID:    chirp.UserID.Bytes,
		}
		chirps = append(chirps, value)
	}
	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) handlerGetOneChirp(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("Query values: %v\n", r.PathValue("chirpID"))
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		log.Printf("Error using uuid.Parse in handlerGetOneChrip! Error: %s", err)
		return
	}
	SQLChirp, err := cfg.dbQueries.ReturnOneChirp(r.Context(), pgtype.UUID{Bytes: chirpID, Valid: true})
	if err != nil {
		log.Printf("Something went wrong in handlerGetOneChirps, trying ReturnOneChirps! Error: %v", err)
		respondWithError(w, 404, err.Error())
		return
	}
	chirp := Chirp{
		ID:        SQLChirp.ID.Bytes,
		CreatedAt: SQLChirp.CreatedAt.Time,
		UpdatedAt: SQLChirp.CreatedAt.Time,
		Body:      SQLChirp.Body,
		UserID:    SQLChirp.UserID.Bytes,
	}
	respondWithJSON(w, 200, chirp)
}
