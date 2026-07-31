# Chirpy

This is a project about creating an HTTP server from scratch. Implementing authentication, a backend database, authorization, and webhooks.

## Requirements

To setup and use this project:
Go version >= 1.25
PostgreSQL

## Setup

First clone the project and download dependencies

```
go clone github.com/hussmaster/Chirpy
cd Chirpy
go mod download
```

### Postgres

After installing postgres you will need to create the database
Using the user 'postgres'
```
sudo -u postgres psql
CREATE DATABASE chirpy;
\c chirpy
```
Install goose for database migration setups
```
go install github.com/pressly/goose/v3/cmd/goose@latest
```
Navigate to the sql/schema folder then run a goose migration
```
goose postgres "postgres://username:password@localhost(or IP if different server):5432/chirpy" up
```
After downloading dependencies, setting up the database and setting the environment variables, you can either run from the project folder
```
go run .
```
or build and execute a binary
```
go build -o chirpy
./chirpy
```

API information can be found at [!docs](/docs/DOCS.md)

