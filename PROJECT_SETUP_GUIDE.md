# The Definitive Guide to Building a Go REST API with Gin, MongoDB Atlas, and Air

---

## Table of Contents
1. [Architecture & Philosophy](#1-architecture--philosophy)
2. [Project Directory Layout](#2-project-directory-layout)
3. [Configuration & Environment Management (`.env`)](#3-configuration--environment-management-env)
4. [Live-Reload Tooling Configuration (`.air.toml`)](#4-live-reload-tooling-configuration-airtoml)
5. [Dependency Management (`go.mod`)](#5-dependency-management-gomod)
6. [Deep Dive: Configuration Module (`internal/config/config.go`)](#6-deep-dive-configuration-module-internalconfigconfiggo)
7. [Deep Dive: Database Driver & Connection Pool (`internal/db/mongo.go`)](#7-deep-dive-database-driver--connection-pool-internaldbmongogo)
8. [Deep Dive: Domain Models & Data Transfer Objects (`internal/notes/note_model.go`)](#8-deep-dive-domain-models--data-transfer-objects-internalnotesnotemodelgo)
9. [Deep Dive: Data Access Layer & Repository (`internal/notes/notes_repo.go`)](#9-deep-dive-data-access-layer--repository-internalnotesnotesrepogo)
10. [Deep Dive: HTTP Handlers & Controllers (`internal/notes/notes_handler.go`)](#10-deep-dive-http-handlers--controllers-internalnotesnoteshandlergo)
11. [Deep Dive: Route Registration & Domain Grouping (`internal/notes/notes_routes.go`)](#11-deep-dive-route-registration--domain-grouping-internalnotesnotesroutesgo)
12. [Deep Dive: Web Router & HTTP Engine (`internal/server/router.go`)](#12-deep-dive-web-router--http-engine-internalserverroutergo)
13. [Deep Dive: Application Entry Point (`cmd/api/main.go`)](#13-deep-dive-application-entry-point-cmdapimaingo)
14. [Networking, TLS, & MongoDB Atlas Lifecycle](#14-networking-tls--mongodb-atlas-lifecycle)
15. [End-to-End API Testing & Verification Guide](#15-end-to-end-api-testing--verification-guide)

---

## 1. Architecture & Philosophy

In Go, simplicity, testability, and modularity take precedence over heavy, opinionated frameworks. This project implements a **Layered Domain Architecture** combined with the **Standard Go Project Layout** (`/cmd` and `/internal`).

```
┌─────────────────────────────────────────────────────────────┐
│                       HTTP Client                           │
└──────────────────────────────┬──────────────────────────────┘
                               │ JSON / HTTP Request
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                    Gin Router & Engine                      │
│                 (internal/server/router.go)                 │
└──────────────────────────────┬──────────────────────────────┘
                               │ Route Dispatch (/notes)
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Handler Layer                       │
│               (internal/notes/notes_handler.go)             │
│   • Request Validation (c.ShouldBindJSON)                   │
│   • DTO to Domain Model Mapping                             │
│   • ID Generation (BSON ObjectID) & UTC Timestamps          │
└──────────────────────────────┬──────────────────────────────┘
                               │ Context & Domain Entity
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                 Repository / Data Access Layer              │
│                (internal/notes/notes_repo.go)               │
│   • Query Execution (InsertOne)                             │
│   • Operation Timeout Bounds (5s Context)                   │
└──────────────────────────────┬──────────────────────────────┘
                               │ Wire Protocol (v2 Driver)
                               ▼
┌─────────────────────────────────────────────────────────────┐
│               MongoDB Atlas Cloud Database                  │
└─────────────────────────────────────────────────────────────┘
```

### Core Design Principles:
1. **Layered Separation of Concerns**:
   - **Routes**: Map paths and HTTP verbs to handler methods.
   - **Handlers (Controllers)**: Parse HTTP requests, validate input bodies, orchestrate business workflows, and format HTTP responses.
   - **Repositories (Data Access)**: Isolate raw MongoDB driver operations and collection queries behind clean Go methods.
   - **Models**: Pure data definitions carrying BSON (persistence) and JSON (transport) struct tags.
2. **Explicit Dependency Injection**:
   Database handles (`*mongo.Database`) and repository instances (`*Repo`) are passed down explicitly through constructors (`NewRouter`, `NewRepo`, `NewHandler`) rather than stored in global state.
3. **Fail-Fast Initialization**:
   If any required environment variable (`MONGO_URI`, `MONGO_DB_NAME`, `PORT`) is missing or invalid, or if the MongoDB Atlas cluster cannot be pinged, the application halts immediately during boot.
4. **Context-Aware I/O & Timeouts**:
   Every database query and network operation is governed by Go's `context.Context` to prevent lingering zombie queries and resource exhaustion.

---

## 2. Project Directory Layout

```text
01crud/
├── cmd/
│   └── api/
│       └── main.go                 # Application entry point: loads config, connects DB, starts server
├── internal/                       # Internal packages protected by Go compiler boundary
│   ├── config/
│   │   └── config.go               # Environment loader & required variable validation
│   ├── db/
│   │   └── mongo.go                # MongoDB Driver v2 connection & lifecycle management
│   ├── notes/                      # Notes domain module
│   │   ├── note_model.go           # Domain structs (Note) & DTOs (CreateNoteRequest)
│   │   ├── notes_repo.go           # MongoDB collection operations (InsertOne, etc.)
│   │   ├── notes_handler.go        # HTTP handlers (validation, response formatting)
│   │   └── notes_routes.go         # Route grouping (/notes) and handler binding
│   └── server/
│       └── router.go               # Gin HTTP engine setup, middleware, and health check
├── tmp/                            # Temporary directory for Air build artifacts
│   └── api.exe                     # Compiled application binary generated by Air
├── .air.toml                       # Configuration file for Air live-reloading tool
├── .env                            # Secrets & environment variables (excluded from version control)
├── .gitignore                      # Git ignore patterns
├── go.mod                          # Module definition and dependency version tracking
├── go.sum                          # Cryptographic checksums of third-party dependencies
└── PROJECT_SETUP_GUIDE.md          # Comprehensive architecture and implementation guide
```

---

## 3. Configuration & Environment Management (`.env`)

The `.env` file holds local secrets, connection strings, and runtime variables.

### The Code:
```env
MONGODB_USERNAME="maverickoluwatomisin_db_user"
MONGODB_PASSWORD="<REDACTED_PASSWORD>"
MONGO_URI="mongodb+srv://maverickoluwatomisin_db_user:<REDACTED_PASSWORD>@cluster0.feg2tgo.mongodb.net/?retryWrites=true&w=majority"
MONGO_DB_NAME=notes_db
PORT=8080
```

### Detailed Explanation:
- `MONGODB_USERNAME` & `MONGODB_PASSWORD`: Database user credentials configured in MongoDB Atlas under Database Access.
- `MONGO_URI`: The MongoDB Connection String using the DNS Seedlist Protocol:
  - `mongodb+srv://`: Instructs the driver to query DNS SRV records to discover active replica set hostnames automatically without hardcoding IP addresses.
  - `?retryWrites=true`: Instructs the driver to automatically retry write operations once if a transient network glitch occurs.
  - `w=majority`: Write Concern requiring that data modifications are committed to a majority of replica set nodes before acknowledging success.
- `MONGO_DB_NAME`: The target logical database (`notes_db`) housing the collections.
- `PORT`: The local TCP port where the HTTP server will bind and listen (`8080`).

---

## 4. Live-Reload Tooling Configuration (`.air.toml`)

[Air](https://github.com/air-verse/air) provides automated live-reloading during Go development, automatically recompiling and restarting the server whenever code changes are saved.

### The Code:
```toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/api.exe ./cmd/api"
bin = "tmp/api.exe"
delay = 300
exclude_dir = ["tmp", "vendor"]
exclude_regex = ["_test.go"]

[log]
time = true
```

### Detailed Line-by-Line Explanation:
- `root = "."`: Sets the project root as the watched directory.
- `tmp_dir = "tmp"`: Designates the directory where temporary build binaries and Air state files are placed.
- `[build]`:
  - `cmd = "go build -o ./tmp/api.exe ./cmd/api"`: Compiles the `cmd/api` entry point into a Windows binary `./tmp/api.exe`.
  - `bin = "tmp/api.exe"`: Tells Air which executable binary to run after compilation.
  - `delay = 300`: 300ms debounce interval preventing repeated recompilations when saving multiple files quickly.
  - `exclude_dir = ["tmp", "vendor"]`: Excludes output and third-party directories from file monitoring.
  - `exclude_regex = ["_test.go"]`: Prevents unnecessary server restarts when editing unit tests.
- `[log] time = true`: Prepends timestamps to all build and terminal output logs.

---

## 5. Dependency Management (`go.mod`)

The project uses Go modules for explicit versioning:

```go
module notes-api

go 1.26.3

require (
	github.com/gin-gonic/gin v1.12.0
	github.com/joho/godotenv v1.5.1
	go.mongodb.org/mongo-driver/v2 v2.5.0
)
```

### Key Dependencies:
- `github.com/gin-gonic/gin`: Ultra-fast HTTP web framework utilizing a Radix tree router, lightweight memory allocation, and composable middleware.
- `github.com/joho/godotenv`: Dotenv library that parses `.env` files and exports them into the running process's environment.
- `go.mongodb.org/mongo-driver/v2`: Official MongoDB Go Driver (v2), featuring modernized context management, generic BSON unmarshaling, and superior throughput over v1.

---

## 6. Deep Dive: Configuration Module (`internal/config/config.go`)

The configuration package provides strict environment validation and loads settings into a typed struct.

### The Code:
```go
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI   string
	MongoDB    string
	ServerPort string
}

func Load() (Config, error) {
	// godotenv.Load() reads the env file and loads the variables into the process environment
	// os.Getenv("ENV_NAME") reads the env variable

	if err := godotenv.Load(); err != nil {
		return Config{}, fmt.Errorf("error loading .env: %w", err)
	}

	mongoURI, err := extractEnv("MONGO_URI")
	if err != nil {
		return Config{}, err
	}

	mongoDB, err := extractEnv("MONGO_DB_NAME")
	if err != nil {
		return Config{}, err
	}

	serverPort, err := extractEnv("PORT")
	if err != nil {
		return Config{}, err
	}

	return Config{
		MongoURI:   mongoURI,
		MongoDB:    mongoDB,
		ServerPort: serverPort,
	}, nil
}

func extractEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	return val, nil
}
```

### Key Technical Details:
- **`Config` Struct**: Encapsulates configuration in memory. Application components never query `os.Getenv` directly.
- **Fail-Fast `extractEnv`**: If an environment variable is empty or unset, it immediately returns a structured error: `"environment variable <KEY> is not set"`.
- **Zero Values on Failure**: If any key is missing, `Config{}, err` is returned so that the caller can abort execution cleanly.

---

## 7. Deep Dive: Database Driver & Connection Pool (`internal/db/mongo.go`)

Handles connecting, health verification via pinging, and graceful disconnection for the MongoDB Atlas database.

### The Code:
```go
package db

import (
	"context"
	"fmt"
	"notes-api/internal/config"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Connect(cfg config.Config) (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.MongoURI)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.MongoDB)

	return client, db, nil
}

func Disconnect(client *mongo.Client) error {
	if client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return client.Disconnect(ctx)
}
```

### Key Technical Details:
- **10-Second Context Deadline**: `Connect()` uses `context.WithTimeout(..., 10*time.Second)` to guarantee the process does not hang indefinitely during cluster discovery or TLS handshake negotiation.
- **Active Health Ping**: `client.Ping(ctx, nil)` forces an immediate roundtrip network check to ensure that the Atlas IP whitelist allows the connection and credentials are valid.
- **Explicit Database Extraction**: `db := client.Database(cfg.MongoDB)` returns the specific `*mongo.Database` handle to pass to domain repositories.
- **Graceful Shutdown Support**: `Disconnect()` cleanly drains and terminates active socket connections within 5 seconds.

---

## 8. Deep Dive: Domain Models & Data Transfer Objects (`internal/notes/note_model.go`)

The domain model layer defines both the database persistence entity and the HTTP request validation contracts.

### The Code:
```go
package notes

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Note struct {
	ID        bson.ObjectID `json:"id" bson:"_id"`
	Title     string        `json:"title" bson:"title"`
	Content   string        `json:"content" bson:"content"`
	Pinned    bool          `json:"pinned" bson:"pinned"`
	CreatedAt time.Time     `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt" bson:"updatedAt"`
}

type CreateNoteRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Pinned  bool   `json:"pinned" binding:"required"`
}
```

### Key Technical Details:
- **`Note` (Domain Entity)**:
  - `ID`: Represented as `bson.ObjectID`, mapped to MongoDB's primary key `_id` and exported as `"id"` in JSON.
  - Dual Struct Tags (`json` and `bson`): Provides strict camelCase JSON serialization for client clients while adhering to BSON document field standards.
  - `CreatedAt` & `UpdatedAt`: Standardized `time.Time` fields for auditing and chronological sorting.
- **`CreateNoteRequest` (Data Transfer Object)**:
  - Decouples client payloads from internal database structures (clients cannot tamper with `_id`, `createdAt`, or `updatedAt`).
  - Gin Validation Tags: `binding:"required"` enforces that `title`, `content`, and `pinned` must be provided in the incoming JSON body; otherwise, Gin automatically flags a validation error.

---

## 9. Deep Dive: Data Access Layer & Repository (`internal/notes/notes_repo.go`)

The repository layer abstracts all database collection queries from the web transport layer.

### The Code:
```go
package notes

// Repo -data access layer

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Repo struct {
	coll *mongo.Collection
}

func NewRepo(db *mongo.Database) *Repo {
	return &Repo{coll: db.Collection("notes")}
}

// CRUD operations

// CreateNote creates a new note
func (r *Repo) Create(ctx context.Context, note Note) (Note, error) {

	opCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := r.coll.InsertOne(opCtx, note)
	if err != nil {
		return Note{}, fmt.Errorf("Failed to save note to database: %w", err)
	}
	return note, nil
}
```

### Key Technical Details:
- **Encapsulated Collection**: `Repo` holds an unexported `coll *mongo.Collection` pointer initialized to the `"notes"` collection.
- **Bounded Execution Time**: `opCtx, cancel := context.WithTimeout(ctx, time.Second*5)` derives a scoped context that guarantees write operations fail after 5 seconds if MongoDB becomes unresponsive.
- **Error Wrapping**: Returns an error wrapped with `%w` for upstream debugging while preserving error chains.

---

## 10. Deep Dive: HTTP Handlers & Controllers (`internal/notes/notes_handler.go`)

The handler layer handles HTTP serialization, schema validation, domain entity generation, and HTTP response formatting.

### The Code:
```go
package notes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler {
	return &Handler{repo: repo}
}

// CreateNote creates a new note
func (h *Handler) CreateNote(c *gin.Context) {

	var req CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().UTC()
	note := Note{
		ID:        bson.NewObjectID(),
		Title:     req.Title,
		Content:   req.Content,
		Pinned:    req.Pinned,
		CreatedAt: now,
		UpdatedAt: now,
	}

	createdNote, err := h.repo.Create(c.Request.Context(), note)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create note"})
		return
	}

	c.JSON(http.StatusCreated, createdNote)
}
```

### Key Technical Details:
1. **Request Validation (`c.ShouldBindJSON`)**:
   Reads the HTTP request body and deserializes it into `CreateNoteRequest`. If required fields are missing or JSON is malformed, it halts and responds with HTTP `400 Bad Request`.
2. **Domain Hydration**:
   - `bson.NewObjectID()`: Generates a unique 12-byte BSON ObjectId on the server side.
   - `time.Now().UTC()`: Ensures all database timestamps are persisted in coordinated universal time (UTC) without server timezone drift.
3. **Context Propagation (`c.Request.Context()`)**:
   Passes the incoming HTTP request context to `h.repo.Create()`. If the HTTP client disconnects or cancels the request mid-flight, the database driver is notified to cancel the downstream query.
4. **Status Code Semantics**:
   Returns `201 Created` with the full note JSON on success, `400 Bad Request` on invalid payloads, and `500 Internal Server Error` on database failure.

---

## 11. Deep Dive: Route Registration & Domain Grouping (`internal/notes/notes_routes.go`)

Encapsulates endpoint routing and wires the repository and handler together for the notes domain.

### The Code:
```go
package notes

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func RegisterRoutes(r *gin.Engine, db *mongo.Database) {
	repo := NewRepo(db)
	handler := NewHandler(repo)

	notesGroup := r.Group("/notes")

	notesGroup.POST("", handler.CreateNote)
}
```

### Key Technical Details:
- **Domain Dependency Injection**: Instantiates `NewRepo(db)` and injects it into `NewHandler(repo)`.
- **Scoped Route Group (`r.Group("/notes")`)**: Isolates all notes-related endpoints under the `/notes` prefix, making it easy to attach domain-specific middleware (such as auth or rate limiting) in the future.
- **RESTful Mapping**: Maps `POST /notes` to `handler.CreateNote`.

---

## 12. Deep Dive: Web Router & HTTP Engine (`internal/server/router.go`)

The core server router initializes the Gin engine, attaches global middleware, configures the system health check, and delegates domain routes.

### The Code:
```go
package server

import (
	"net/http"

	"notes-api/internal/notes"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func NewRouter(database *mongo.Database) *gin.Engine {

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"status": "healthy server",
		})
	})

	notes.RegisterRoutes(r, database)

	return r
}
```

### Key Technical Details:
- **`database *mongo.Database` Parameter**: Receives the live database handle from `main.go` and forwards it to domain route registrars.
- **`gin.Default()`**: Pre-wires standard Gin middlewares:
  1. **Logger**: Formats request logs (`method`, `path`, `status`, `latency`) to `stdout`.
  2. **Recovery**: Catches unhandled panics anywhere in the request cycle, logs the stack trace, and returns HTTP `500 Internal Server Error`.
- **System Health Probe**: `GET /health` provides a lightweight liveness endpoint returning `{"ok": true, "status": "healthy server"}`.
- **Domain Delegation**: Calls `notes.RegisterRoutes(r, database)` to wire up domain resources.

---

## 13. Deep Dive: Application Entry Point (`cmd/api/main.go`)

Orchestrates application startup: parses configuration, connects to the database, schedules graceful cleanup, initializes routes, and starts the HTTP server.

### The Code:
```go
package main

import (
	"fmt"
	"log"
	"notes-api/internal/config"
	"notes-api/internal/db"
	"notes-api/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	client, database, err := db.Connect(cfg)

	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}

	defer func() {
		if err := db.Disconnect(client); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()

	router := server.NewRouter(database)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)

	log.Printf("starting server on %s", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
```

### Lifecycle Execution Sequence:
1. **Config Loading**: `config.Load()` checks environment variables. If any are missing, process exits with code 1.
2. **Database Initialization**: `db.Connect(cfg)` establishes connection and pings the MongoDB Atlas cluster. Both the `*mongo.Client` and `*mongo.Database` handles are returned.
3. **Deferred Cleanup**: `defer db.Disconnect(client)` is registered on the stack, ensuring socket cleanup when `main()` exits.
4. **Router Construction**: `server.NewRouter(database)` configures endpoints and injects the database handle.
5. **Server Listen**: `router.Run(addr)` binds to the configured TCP port (`:8080`) and begins serving incoming HTTP requests.

---

## 14. Networking, TLS, & MongoDB Atlas Lifecycle

Connecting Go to MongoDB Atlas requires understanding the network and security pipeline:

```
┌─────────────────┐       TCP (27017)       ┌───────────────────────────────┐
│ Go Application  │ ──────────────────────> │  MongoDB Atlas Cloud Proxy    │
└─────────────────┘                         └───────────────────────────────┘
                                                           │
                                              TLS Handshake / IP Inspection
                                                           │
                                            ┌──────────────┴──────────────┐
                                            ▼                             ▼
                                   [IP Whitelisted & Active]    [IP Not Whitelisted / Paused]
                                            │                             │
                                            ▼                             ▼
                                    TLS Handshake OK              TLS Alert 80: Internal Error
                                            │                             │
                                            ▼                             ▼
                                   ReplicaSet Discovered             Ping Timeout (Exit Code 1)
```

### Critical Atlas Diagnostics & Solutions:

#### 1. Why `remote error: tls: internal error` Occurs
Atlas proxy endpoints accept initial TCP connections on port 27017. During the TLS `ClientHello` exchange, Atlas inspects the client's public IP address. If the client IP is **not whitelisted** in Atlas Network Access, Atlas terminates the connection with a TLS Alert `internal_error`.
- **Solution**: In MongoDB Atlas, go to **Network Access** → **Add IP Address** → add current IP or `0.0.0.0/0` (for development).

#### 2. Paused M0 Free Tier Clusters
Free-tier (M0) clusters are automatically paused by MongoDB Atlas after prolonged periods of inactivity. When a cluster is paused, all network rules are temporarily marked **Inactive**.
- **Solution**: Go to the **Clusters** view in Atlas and click **Resume**. Wait ~1–2 minutes for the cluster to resume before starting the API.

---

## 15. End-to-End API Testing & Verification Guide

### 1. Start the Application
Run Air from the project root:
```bash
air
```

### 2. Verify Health Endpoint
```bash
curl -X GET http://localhost:8080/health
```

**Expected Response (`200 OK`)**:
```json
{
  "ok": true,
  "status": "healthy server"
}
```

---

### 3. Create a Note (Valid Payload)
```bash
curl -X POST http://localhost:8080/notes \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Study Go Concurrency",
    "content": "Deep dive into goroutines, channels, and sync.WaitGroup",
    "pinned": true
  }'
```

**Expected Response (`201 Created`)**:
```json
{
  "id": "66b1e7c9f5d1a23b8e4c10a1",
  "title": "Study Go Concurrency",
  "content": "Deep dive into goroutines, channels, and sync.WaitGroup",
  "pinned": true,
  "createdAt": "2026-09-02T19:42:00.123456789Z",
  "updatedAt": "2026-09-02T19:42:00.123456789Z"
}
```

---

### 4. Create a Note (Validation Error - Missing Field)
Send a request missing the required `title` field:
```bash
curl -X POST http://localhost:8080/notes \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Missing title",
    "pinned": false
  }'
```

**Expected Response (`400 Bad Request`)**:
```json
{
  "error": "Key: 'CreateNoteRequest.Title' Error:Field validation for 'Title' failed on the 'required' tag"
}
```
