-- +goose Up
CREATE TABLE refresh_tokens (
    token TEXT primary key,
    created_at timestamp not null,
    updated_at timestamp not null,
    user_id UUID not null references users(id) on DELETE CASCADE,
    expires_at timestamp not null,
    revoked_at timestamp
);

-- +goose Down
DROP TABLE refresh_tokens;


