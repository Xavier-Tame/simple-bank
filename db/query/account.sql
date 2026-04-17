-- name: CreateAccount :one
INSERT INTO
  accounts (owner, balance, currency)
VALUES
  ($1, $2, $3)
RETURNING
  *;

-- name: GetAccount :one
SELECT
  *
FROM
  accounts
WHERE
  id = $1
  AND closed_at IS NULL
LIMIT
  1;

-- name: GetDeletedAccount :one
SELECT
  *
FROM
  accounts
WHERE
  id = $1
  AND closed_at IS NOT NULL
LIMIT
  1;

-- name: GetAccountForUpdate :one
SELECT
  *
FROM
  accounts
WHERE
  id = $1
  AND closed_at IS NULL
LIMIT
  1
FOR NO KEY UPDATE;

-- name: ListAccounts :many
SELECT
  *
FROM
  accounts
WHERE
  owner = $1
  AND closed_at IS NULL
ORDER BY
  id
LIMIT
  $2
OFFSET
  $3;

-- name: UpdateAccount :one
UPDATE accounts
SET
  balance = $2
WHERE
  id = $1
  AND closed_at IS NULL
RETURNING
  *;

-- name: AddAccountBalance :one
UPDATE accounts
SET
  balance = balance + sqlc.arg (amount)
WHERE
  id = sqlc.arg (id)
  AND closed_at IS NULL
RETURNING
  *;

-- name: DeleteAccount :exec
UPDATE accounts
SET
  closed_at = now()
WHERE
  id = $1
  AND closed_at IS NULL;