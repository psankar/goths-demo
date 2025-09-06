-- name: AddUser :exec
INSERT INTO users (username, password) VALUES (?, ?);