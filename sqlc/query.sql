-- name: AddUser :exec
INSERT INTO users (username, password) VALUES (?, ?);

-- name: Authenticate :one
SELECT * FROM users WHERE username = ? AND password = ?;

-- name: AddPost :one
INSERT INTO posts (user_id, content) 
SELECT u.id, ? 
FROM users u 
WHERE u.username = ?
RETURNING id;
