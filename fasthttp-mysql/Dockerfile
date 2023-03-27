FROM mysql

# Copy the database schema
COPY ./models/init.sql /docker-entrypoint-initdb.d/

ENV MYSQL_ROOT_PASSWORD=root

# Expose the port
EXPOSE 3306

