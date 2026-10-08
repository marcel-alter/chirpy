-- +goose Up
CREATE TABLE chirps (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    body TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users
    ON DELETE CASCADE 
    ON UPDATE CASCADE,
    FOREIGN KEY(user_id) REFERENCES users (id)
);

-- +goose Down
DROP TABLE chirps;