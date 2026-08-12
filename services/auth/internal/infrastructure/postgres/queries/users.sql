-- name: GetUserByExternalID :one
SELECT id, external_id, email, password_hash, status, version, created_at, updated_at, deleted_at
FROM users
WHERE external_id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT id, external_id, email, password_hash, status, version, created_at, updated_at, deleted_at
FROM users
WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL;

-- name: UserExistsByEmail :one
SELECT EXISTS(
    SELECT 1 FROM users WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL
) AS exists;

-- name: CreateUser :one
INSERT INTO users (
    external_id, email, password_hash, status, version, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, external_id, email, password_hash, status, version, created_at, updated_at, deleted_at;

-- name: UpdateUser :one
UPDATE users
SET
    email = $2,
    password_hash = $3,
    status = $4,
    version = $5,
    updated_at = $6
WHERE external_id = $1
  AND version = $7
  AND deleted_at IS NULL
RETURNING id, external_id, email, password_hash, status, version, created_at, updated_at, deleted_at;

-- name: SoftDeleteUser :execrows
UPDATE users
SET deleted_at = $2, updated_at = $2, version = version + 1
WHERE external_id = $1 AND deleted_at IS NULL;

-- name: Ping :one
SELECT 1::integer AS result;
