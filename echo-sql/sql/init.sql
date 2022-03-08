CREATE TABLE IF NOT EXISTS url_map (
    id varchar(8) NOT NULL,
    redirect_url varchar(150) NOT NULL,
    PRIMARY KEY(id)
);