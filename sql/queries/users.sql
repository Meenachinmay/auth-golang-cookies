-- name: CreateUser :one
INSERT INTO users (
                   id,
                   name,
                   userName,
                   email,
                   password,
                   created_at,
                   updated_at
) VALUES (
          $1, $2, $3, $4, $5, $6, $7
         ) RETURNING *;

-- name: FindUserByEmail :one
SELECT *
FROM users
WHERE email = $1
LIMIT 1;

-- name: FindUserById :one
SELECT *
FROM users
WHERE id = $1
LIMIT 1;