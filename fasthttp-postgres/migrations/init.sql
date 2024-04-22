CREATE TABLE IF NOT EXISTS authors(
    id serial UNIQUE,
    first_name varchar,
    last_name varchar
);
INSERT INTO authors(first_name, last_name)  VALUES 
    ('Charles', 'Dickens'),
    ('Alexandre', 'Dumas'),
    ('Jane', 'Austin'),
    ('Franz', 'Kafka'),
    ('Mark', 'Twain'),
    ('Leo', 'Tolsoy');

CREATE TABLE IF NOT EXISTS books(
    id serial UNIQUE,
    title varchar,
    year integer,
    author_id integer,
    CONSTRAINT fk_author
        FOREIGN KEY(author_id)
            REFERENCES authors(id)
);

INSERT INTO books (title, year, author_id) VALUES 
    ('Oliver Twist', 1837, 1),
    ('David Copperfield', 1849, 1),
    ('Great Expectations',1860, 1),
    ('The Three Musketeers', 1844,2),
    ('The count of Monte Cristo', 1844, 2),
    ('Pride and Prejudice',1813, 3),
    ('Sense and Sensibility',1811, 3),
    ('The castle', 1926, 4),
    ('The trial',1925, 4),
    ('The metamorphosis', 1915, 4),
    ('The adventures of Tom Sawyer', 1876, 5),
    ('The adventures of Huckleberry Finn', 1884, 5),
    ('War and Peace', 1869, 6),
    ('Anna Karenina', 1878,6);
