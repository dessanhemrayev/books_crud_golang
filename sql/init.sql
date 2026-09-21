Drop table if exists books;
Drop table if exists favorite_books;
Drop table if exists users;

create table users (
    id serial primary key,
    username varchar(255) not null unique,
    password varchar(255) not null,
    created_at timestamp default current_timestamp
);

create table books (
    id serial primary key,
    title varchar(255) not null,
    author varchar(255) not null,
    published_date date not null,
    is_available boolean default true,
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp
);
create table favorite_books (
    id serial primary key,
    book_id int references books(id) on delete cascade,
    user_id int references users(id) on delete cascade,
    created_at timestamp default current_timestamp
);

INSERT INTO users (username, password) VALUES
('user1', 'password1'),
('user2', 'password2'),
('user3', 'password3');

INSERT INTO books (title, author, published_date, is_available) VALUES
('The Great Gatsby', 'F. Scott Fitzgerald', '1925-04-10', true),
('To Kill a Mockingbird', 'Harper Lee', '1960-07-11', true),
('1984', 'George Orwell', '1949-06-08', true),
('Pride and Prejudice', 'Jane Austen', '1813-01-28', true),
('The Catcher in the Rye', 'J.D. Salinger', '1951-07-16', true);