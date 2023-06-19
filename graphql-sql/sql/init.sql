CREATE TABLE IF NOT EXISTS authors
(
    id serial PRIMARY KEY,
    name varchar(100) NOT NULL,
    email varchar(150) NOT NULL,
    created_at date
);

CREATE TABLE IF NOT EXISTS posts
(
    id serial PRIMARY KEY,
    title varchar(100) NOT NULL,
    content text NOT NULL,
    author_id int,
    created_at date
);