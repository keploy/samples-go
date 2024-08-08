FROM --platform=linux/amd64 debian:stable-slim

RUN apt-get update && apt-get install -y ca-certificates

ADD http-pokeapi /usr/bin/http-pokeapi

EXPOSE 8080

CMD ["http-pokeapi"]
