-- +goose Up
CREATE TABLE chirps (
    id UUID primary key,
    created_at timestamp not null,
    updated_at timestamp not null,
    body TEXT  not null,
    user_id UUID not null references users(id) on DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps;