
-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
  token, created_at, updated_at, user_id, expires_at, revoked_at
) VALUES (
  $1, NOW(), NOW(), $2, $3, NULL
)RETURNING *;


-- name: GetRefreshToken :one
SELECT
  token,
  created_at,
  updated_at,
  user_id,
  expires_at,
  revoked_at
FROM refresh_tokens
WHERE token = $1;

-- name: GetValidRefreshToken :one
SELECT
  token,
  created_at,
  updated_at,
  user_id,
  expires_at,
  revoked_at
FROM refresh_tokens
WHERE token = $1
  AND revoked_at IS NULL
  AND expires_at > NOW();


-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW(),
    updated_at = NOW()
WHERE token = $1
  AND revoked_at IS NULL;



-- name: ListRefreshTokensForUser :many
SELECT
  token, created_at, updated_at, expires_at, revoked_at
FROM refresh_tokens
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at <= NOW();

-- name: RevokeAllRefreshTokensForUser :exec
UPDATE refresh_tokens
SET revoked_at = NOW(),
    updated_at = NOW()
WHERE user_id = $1
  AND revoked_at IS NULL;

