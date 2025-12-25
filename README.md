# try-mysql-partitioning

A simple Document Management System demonstrating MySQL table partitioning with a Go web application.

## Overview

This project consists of three main components:

1. **Infrastructure (compose.yaml)**: Docker Compose configuration with Go app and MySQL8 containers
2. **Application (app/)**: Go implementation providing a simple CRUD web API for documents and attributes
3. **Database (db/)**: MySQL8 initialization SQL with partitioned tables

## Features

- **Document Management**: Create, Read, Update, Delete documents
- **Attribute Management**: Manage key-value attributes for documents
- **MySQL Partitioning**:
  - Documents table: RANGE partitioning by year (created_at)
  - Attributes table: HASH partitioning by document_id (4 partitions)
- **No 3rd party libraries**: Pure Go standard library (except MySQL driver)

## Project Structure

```
.
├── compose.yaml          # Docker Compose configuration
├── Dockerfile           # Go app container definition
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── .gitignore          # Git ignore rules
├── app/
│   └── main.go        # Go application with HTTP handlers
└── db/
    └── init.sql       # Database initialization and partitioning setup
```

## Getting Started

### Prerequisites

- Docker
- Docker Compose

### Running the Application

1. Clone the repository:
   ```bash
   git clone https://github.com/sters/try-mysql-partitioning.git
   cd try-mysql-partitioning
   ```

2. Start the services:
   ```bash
   docker compose up --build
   ```

3. Access the application:
   - Web UI: http://localhost:8080
   - API: http://localhost:8080/documents

### API Endpoints

#### Documents
- `GET /documents` - List all documents
- `GET /documents/{id}` - Get a specific document
- `POST /documents` - Create a new document
- `PUT /documents/{id}` - Update a document
- `DELETE /documents/{id}` - Delete a document

#### Attributes
- `GET /attributes` - List all attributes
- `GET /attributes?document_id={id}` - List attributes for a document
- `GET /attributes/{id}` - Get a specific attribute
- `POST /attributes` - Create a new attribute
- `PUT /attributes/{id}` - Update an attribute
- `DELETE /attributes/{id}` - Delete an attribute

### Example API Calls

Create a document:
```bash
curl -X POST http://localhost:8080/documents \
  -H "Content-Type: application/json" \
  -d '{"title":"My Document","content":"Document content","created_at":"2024-12-25T10:00:00Z"}'
```

List all documents:
```bash
curl http://localhost:8080/documents
```

Create an attribute:
```bash
curl -X POST http://localhost:8080/attributes \
  -H "Content-Type: application/json" \
  -d '{"document_id":1,"attr_key":"author","attr_value":"John Doe"}'
```

## Database Partitioning Details

### Documents Table (RANGE Partitioning)
Partitioned by year using the `created_at` field:
- p2020: Documents from 2020
- p2021: Documents from 2021
- p2022: Documents from 2022
- p2023: Documents from 2023
- p2024: Documents from 2024
- p2025: Documents from 2025
- pfuture: Documents from 2026 onwards

### Attributes Table (HASH Partitioning)
Partitioned into 4 partitions using HASH on `document_id` field for even distribution.

## Stopping the Application

```bash
docker compose down
```

To remove volumes:
```bash
docker compose down -v
```

## Development

To modify the application:
1. Edit the Go code in `app/main.go`
2. Edit database schema in `db/init.sql`
3. Rebuild and restart: `docker compose up --build`