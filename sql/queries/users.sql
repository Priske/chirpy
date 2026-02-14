-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email,hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: ResetUsers :exec
DELETE FROM users;


-- name: GetUserByEmail :one
select * from  users where email = $1;


-- name: GetUserById :one
select * from users where id = $1;

-- name: UpdateUserInfo :one
UPDATE users
SET email = COALESCE($2, email),
    hashed_password = COALESCE($3, hashed_password)
where id = $1
RETURNING *;


-- name: UpgradeUserByIdToChirpyRed :one
UPDATE users
set is_chirpy_red = true
where id = $1
RETURNING *;
