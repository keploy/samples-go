-- Create the database
CREATE DATABASE moviedb;

-- Connect to the database
USE moviedb;

-- Create the movies table
CREATE TABLE movies(
    id INT NOT NULL AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    year INT NOT NULL,
    rating INT NOT NULL,
    PRIMARY KEY (id)
    );

-- seed table with data
INSERT INTO movies (title, year, rating) VALUES ('The Godfather', 1972, 9);
INSERT INTO movies (title, year, rating) VALUES ('The Shawshank Redemption', 1994, 9);
INSERT INTO movies (title, year, rating) VALUES ('The Dark Knight', 2008, 7);
INSERT INTO movies (title, year, rating) VALUES ('The Godfather: Part II', 1974, 9);
INSERT INTO movies (title, year, rating) VALUES ('The Lord of the Rings: The Return of the King', 2003, 6);
INSERT INTO movies (title, year, rating) VALUES ('Pulp Fiction', 1994, 8);
INSERT INTO movies (title, year, rating) VALUES ('Schindler''s List', 1993, 8);
INSERT INTO movies (title, year, rating) VALUES ('The Good, the Bad and the Ugly', 1966, 8);
INSERT INTO movies (title, year, rating) VALUES ('The Lord of the Rings: The Fellowship of the Ring', 2001, 6);
INSERT INTO movies (title, year, rating) VALUES ('The Lord of the Rings: The Two Towers', 2002, 6);
INSERT INTO movies (title, year, rating) VALUES ('Fight Club', 1999, 8);
INSERT INTO movies (title, year, rating) VALUES ('The Matrix', 1999, 8);
INSERT INTO movies (title, year, rating) VALUES ('Forrest Gump', 1994, 8);
INSERT INTO movies (title, year, rating) VALUES ('Inception', 2010, 8);
INSERT INTO movies (title, year, rating) VALUES ('The Dark Knight Rises', 2012, 7);