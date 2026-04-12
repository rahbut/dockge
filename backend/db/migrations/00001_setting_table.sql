-- +goose Up
CREATE TABLE IF NOT EXISTS setting (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    key   VARCHAR(200) NOT NULL UNIQUE,
    value TEXT,
    type  VARCHAR(20)
);

-- +goose Down
DROP TABLE IF EXISTS setting;
