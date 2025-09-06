-- name: AddUser :exec
INSERT INTO users (username, password) VALUES (?, ?);

-- name: Authenticate :one
SELECT * FROM users WHERE username = ? AND password = ?;
