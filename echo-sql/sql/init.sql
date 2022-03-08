CREATE TABLE IF NOT EXISTS url_map (
    id varchar(8) NOT NULL,
    redirect_url varchar(150) NOT NULL,
    created_at timestamp WITHOUT time zone NOT NULL,
    updated_at timestamp WITHOUT time zone NOT NULL,
    PRIMARY KEY(id)
);