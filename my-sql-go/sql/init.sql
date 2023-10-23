CREATE TABLE IF NOT EXISTS url_map (
    id char(8) NOT NULL,
    redirect_url varchar(150) NOT NULL UNIQUE,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL,
    PRIMARY KEY(id)
);
