package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode"
)

// ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓ JSON STRUCTS
type errorResponse struct {
	Error string `json:"body"`
}

type validResponse struct {
	Body string `json:"body"`
}

type RequestBody struct {
	Body string `json:"body"`
}

// ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓ HELPER FUNCTIONS
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
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
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
	_ = cfg.fileserverHits.Swap(0)
}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) { //is there a special reason, why http.Handlers aren't allowed to return errors?
	profanities := []string{"kerfuffle", "sharbert", "fornax"}
	decoder := json.NewDecoder(r.Body)
	params := RequestBody{}

	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, 400, "Something went wrong")
		log.Printf("Error decoding parameters: %s", err)
		return
	}
	if len(params.Body) > 141 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	for _, prof := range profanities {
		params.Body = profanityFilter(params.Body, prof, "****")
	}
	respondWithJSON(w, 200, validResponse{Body: params.Body})
}
