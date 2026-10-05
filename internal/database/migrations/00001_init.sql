-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id serial primary key,
    username varchar(255) not null unique,
    password varchar(255) not null,
    created_at timestamp default current_timestamp
);

CREATE TABLE IF NOT EXISTS books (
    id serial primary key,
    title varchar(255) not null,
    author varchar(255) not null,
    published_date date not null,
    is_available boolean default true,
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp
);

CREATE TABLE IF NOT EXISTS favorite_books (
    id serial primary key,
    book_id int references books(id) on delete cascade,
    user_id int references users(id) on delete cascade,
    created_at timestamp default current_timestamp
);

-- +goose Down
-- Up may reuse existing tables, so their ownership cannot be determined safely.
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION '00001_init is irreversible: schema rollback requires manual intervention';
END;
$$;
-- +goose StatementEnd
