-- +goose Up
CREATE TABLE IF NOT EXISTS user (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    username         VARCHAR(255) NOT NULL UNIQUE,
    password         VARCHAR(255),
    active           BOOLEAN NOT NULL DEFAULT 1,
    timezone         VARCHAR(150),
    twofa_secret     VARCHAR(64),
    twofa_status     BOOLEAN NOT NULL DEFAULT 0,
    twofa_last_token VARCHAR(6)
);

-- +goose Down
DROP TABLE IF EXISTS user;
