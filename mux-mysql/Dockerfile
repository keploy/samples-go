FROM golang:1.21.4-alpine AS build
RUN mkdir /app
COPY . /app
WORKDIR /app

ENV ConnectionString="root:my-secret-pw@tcp(mysql:3306)/mysql"

RUN go build -o main .
RUN chmod +x /app/main

#### Stage 2
FROM scratch
COPY --from=build /app/main /main
ENV ConnectionString="root:my-secret-pw@tcp(mysql:3306)/mysql"
ENTRYPOINT [ "/main" ]