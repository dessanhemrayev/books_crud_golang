-- +goose Up
INSERT INTO users (username, password) VALUES
('user1', 'password1'),
('user2', 'password2'),
('user3', 'password3')
ON CONFLICT (username) DO NOTHING;

INSERT INTO books (title, author, published_date, is_available)
SELECT * FROM (VALUES
    ('The Great Gatsby', 'F. Scott Fitzgerald', '1925-04-10'::date, true),
    ('To Kill a Mockingbird', 'Harper Lee', '1960-07-11'::date, true),
    ('1984', 'George Orwell', '1949-06-08'::date, true),
    ('Pride and Prejudice', 'Jane Austen', '1813-01-28'::date, true),
    ('The Catcher in the Rye', 'J.D. Salinger', '1951-07-16'::date, true)
) AS seed(title, author, published_date, is_available)
WHERE NOT EXISTS (
    SELECT 1 FROM books b
    WHERE b.title = seed.title AND b.author = seed.author
);

-- +goose Down
-- Up does not record inserted IDs; matching rows may predate the migration,
-- have been edited, or be referenced by favorite_books.
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION '00002_seed is irreversible: seed rollback requires manual intervention';
END;
$$;
-- +goose StatementEnd
