# 📝 Notes REST API

A robust, production-grade RESTful CRUD API for managing notes, built with **Go (Golang)**, the **Gin Web Framework**, and the **MongoDB Go Driver v2**.

This project implements a clean, layered architecture separating domain entities, data access (repository), transport handlers, and configuration, along with modern features like cursor-based pagination, regex search, partial updates (PATCH), and live reloading.

---

## 🌟 Features

- **Layered Architecture**: Clean separation of concerns (`cmd`, `config`, `db`, `server`, `repository`, `handlers`, `models`).
- **Full CRUD Operations**: Create, Read (single & list), Partial Update (`PATCH`), and Delete notes.
- **Partial Updates (`PATCH`)**: Uses Go pointer fields (`*string`, `*bool`) to selectively update only the fields sent in the request body without overriding unprovided fields.
- **Cursor-Based Pagination**: Efficient, scalable pagination via MongoDB ObjectID cursor (`nextCursor`) and configurable `limit`.
- **Query Filtering & Search**:
  - Filter notes by pinned status (`pinned=true` / `pinned=false`).
  - Case-insensitive regex title search (`search=<query>` or `q=<query>`).
- **Standardized API Envelope**: Unified JSON response contract (`APIResponse[T]`) across all endpoints.
- **Resilient MongoDB Driver v2**: Safe database lifecycle management, context timeouts (`context.WithTimeout`), ping validation, and graceful teardown.
- **Live Reload**: Pre-configured with [Air](https://github.com/air-verse/air) for instant recompilation on file changes.

---

## 🏗️ Project Structure

```text
.
├── cmd/
│   └── api/
│       └── main.go              # Application entrypoint (bootstrap, DB init, server start)
├── internal/
│   ├── config/
│   │   └── config.go            # Environment variable loader and validator
│   ├── db/
│   │   └── mongo.go             # MongoDB connection, ping verification, and teardown
│   ├── notes/
│   │   ├── note_model.go        # Domain models (Note), DTOs (Requests), and Generic API responses
│   │   ├── notes_handler.go     # Gin HTTP handlers (request parsing, response formatting)
│   │   ├── notes_repo.go        # Data access layer (MongoDB CRUD queries, filters, cursors)
│   │   └── notes_routes.go      # Route definitions and handler mappings
│   └── server/
│       └── router.go            # Gin engine initialization, global routes, and middleware
├── .air.toml                    # Air live-reload configuration
├── .env.example                 # Environment variable template
├── go.mod                       # Go module dependencies
├── go.sum                       # Dependency checksums
└── README.md                    # Project documentation
```

---

## 🛠️ Tech Stack & Dependencies

- **Language**: [Go](https://go.dev/) (v1.22+)
- **Web Framework**: [Gin Web Framework](https://github.com/gin-gonic/gin) (`v1.12.0`)
- **Database Driver**: [MongoDB Go Driver v2](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2) (`v2.5.0`)
- **Environment Management**: [godotenv](https://github.com/joho/godotenv) (`v1.5.1`)
- **Validation**: [validator/v10](https://github.com/go-playground/validator) (via Gin)
- **Development Tooling**: [Air](https://github.com/air-verse/air) (Live reload)

---

## 📋 Prerequisites

Before running the application, ensure you have the following installed:

1. **Go**: Version 1.22 or higher ([Download Go](https://go.dev/dl/))
2. **MongoDB**: Local MongoDB instance (`mongodb://localhost:27017`) or a [MongoDB Atlas](https://www.mongodb.com/atlas) cluster connection string.
3. *(Optional)* **Air**: For live reload during development:
   ```bash
   go install github.com/air-verse/air@latest
   ```

---

## 🚀 Getting Started

### 1. Clone the Repository

```bash
git clone <repository-url>
cd 01crud
```

### 2. Configure Environment Variables

Create a `.env` file in the root directory by copying `.env.example`:

```bash
# Windows PowerShell
Copy-Item .env.example .env

# macOS / Linux
cp .env.example .env
```

Edit `.env` with your configuration:

```env
# HTTP Port for the Gin web server
PORT=8080

# MongoDB Connection String (Atlas or Local)
MONGO_URI=mongodb://localhost:27017

# Target Database Name
MONGO_DB_NAME=notes_db
```

### 3. Install Dependencies

Download and tidy Go modules:

```bash
go mod download
go mod tidy
```

### 4. Run the Server

#### Option A: Normal Execution
```bash
go run ./cmd/api
```

#### Option B: Live Reload with Air (Recommended for Development)
```bash
air
```

The server will start listening on the configured port (default: `http://localhost:8080`).

---

## 📡 API Reference

Base URL: `http://localhost:8080`

All successful responses follow the standardized generic JSON envelope:

```json
{
  "status": "success",
  "message": "Human-readable description",
  "data": { ... },
  "pagination": { ... } // Present on paginated endpoints
}
```

---

### 1. Health Check

Verifies that the server is alive and operational.

- **Method**: `GET`
- **Path**: `/health`
- **Response** (`200 OK`):
  ```json
  {
    "ok": true,
    "status": "healthy server"
  }
  ```

---

### 2. Create Note

Creates a new note. `title` and `content` are required. `pinned` defaults to `false` if omitted.

- **Method**: `POST`
- **Path**: `/notes`
- **Headers**: `Content-Type: application/json`
- **Request Body**:
  ```json
  {
    "title": "Grocery List",
    "content": "Milk, Eggs, Bread, Coffee",
    "pinned": true
  }
  ```
- **Response** (`201 Created`):
  ```json
  {
    "status": "success",
    "message": "Note created successfully",
    "data": {
      "id": "67ca356b4f728c31ab901a11",
      "title": "Grocery List",
      "content": "Milk, Eggs, Bread, Coffee",
      "pinned": true,
      "createdAt": "2026-09-05T20:30:00Z",
      "updatedAt": "2026-09-05T20:30:00Z"
    }
  }
  ```

---

### 3. List Notes (with Cursor Pagination & Filters)

Returns a paginated list of notes sorted chronologically by `_id`.

- **Method**: `GET`
- **Path**: `/notes`
- **Query Parameters**:

| Parameter | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `limit` | `int` | `10` | Number of items per page (positive integer). |
| `nextCursor` | `string` | `""` | The ObjectID hex string from the previous page's `nextCursor`. |
| `pinned` | `bool` | *(all)* | Filter by pinned status (`true` or `false`). |
| `search` or `q` | `string` | `""` | Case-insensitive regex match against note `title`. |

- **Response** (`200 OK`):
  ```json
  {
    "status": "success",
    "message": "Notes retrieved successfully",
    "pagination": {
      "limit": 2,
      "nextCursor": "67ca356b4f728c31ab901a12",
      "hasMore": true,
      "pageLength": 2
    },
    "data": [
      {
        "id": "67ca356b4f728c31ab901a11",
        "title": "Grocery List",
        "content": "Milk, Eggs, Bread",
        "pinned": true,
        "createdAt": "2026-09-05T20:30:00Z",
        "updatedAt": "2026-09-05T20:30:00Z"
      },
      {
        "id": "67ca356b4f728c31ab901a12",
        "title": "Meeting Notes",
        "content": "Discuss architecture",
        "pinned": false,
        "createdAt": "2026-09-05T20:31:00Z",
        "updatedAt": "2026-09-05T20:31:00Z"
      }
    ]
  }
  ```

---

### 4. Get Note By ID

Retrieves a single note by its 24-character hexadecimal MongoDB ObjectID.

- **Method**: `GET`
- **Path**: `/notes/:id`
- **URL Parameters**:
  - `id`: 24-character hex MongoDB ObjectID (e.g. `67ca356b4f728c31ab901a11`)
- **Response** (`200 OK`):
  ```json
  {
    "status": "success",
    "message": "Note retrieved successfully",
    "data": {
      "id": "67ca356b4f728c31ab901a11",
      "title": "Grocery List",
      "content": "Milk, Eggs, Bread",
      "pinned": true,
      "createdAt": "2026-09-05T20:30:00Z",
      "updatedAt": "2026-09-05T20:30:00Z"
    }
  }
  ```
- **Error Responses**:
  - `400 Bad Request`: If `:id` is not a valid 24-hex MongoDB ObjectID.
  - `404 Not Found`: If no note matches the provided ID.

---

### 5. Update Note (Partial Update / PATCH)

Performs a partial update on an existing note. You only need to pass the fields you want to change; omitted fields remain unchanged in MongoDB.

- **Method**: `PATCH`
- **Path**: `/notes/:id`
- **Headers**: `Content-Type: application/json`
- **Request Body** *(all fields optional)*:
  ```json
  {
    "pinned": false
  }
  ```
- **Response** (`200 OK`):
  ```json
  {
    "status": "success",
    "message": "Note updated successfully",
    "data": {
      "id": "67ca356b4f728c31ab901a11",
      "title": "Grocery List",
      "content": "Milk, Eggs, Bread",
      "pinned": false,
      "createdAt": "2026-09-05T20:30:00Z",
      "updatedAt": "2026-09-05T20:45:00Z"
    }
  }
  ```

---

### 6. Delete Note

Deletes a note by its ObjectID.

- **Method**: `DELETE`
- **Path**: `/notes/:id`
- **Response** (`200 OK`):
  ```json
  {
    "status": "success",
    "message": "Note deleted successfully"
  }
  ```
- **Error Responses**:
  - `400 Bad Request`: If `:id` is invalid hex format.
  - `404 Not Found`: If note does not exist.

---

## 🧪 Quick Test with cURL

Copy and run these commands in your terminal:

```bash
# 1. Health check
curl -X GET http://localhost:8080/health

# 2. Create a Note
curl -X POST http://localhost:8080/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"Golang Study","content":"Review Go routines and channels","pinned":true}'

# 3. List Notes (Limit 5)
curl -X GET "http://localhost:8080/notes?limit=5"

# 4. Search Notes by Title (Case-insensitive)
curl -X GET "http://localhost:8080/notes?search=study"

# 5. Filter Only Pinned Notes
curl -X GET "http://localhost:8080/notes?pinned=true"

# 6. Get Next Page Using Cursor
curl -X GET "http://localhost:8080/notes?limit=2&nextCursor=<ID_FROM_PREVIOUS_PAGE>"

# 7. Partial Update (PATCH title only)
curl -X PATCH http://localhost:8080/notes/<NOTE_ID> \
  -H "Content-Type: application/json" \
  -d '{"title":"Advanced Golang Concurrency"}'

# 8. Delete a Note
curl -X DELETE http://localhost:8080/notes/<NOTE_ID>
```

---

## ⚙️ Environment Variables Reference

| Variable | Type | Required | Default | Description |
| :--- | :--- | :---: | :--- | :--- |
| `PORT` | String / Int | Yes | `8080` | Port where Gin binds the HTTP server. |
| `MONGO_URI` | String | Yes | - | MongoDB connection URI (`mongodb://...` or `mongodb+srv://...`). |
| `MONGO_DB_NAME` | String | Yes | `notes_db` | Name of the database to use inside MongoDB. |

---

## 🛡️ Architecture & Design Decisions

1. **DTOs vs Domain Entities**:
   - `Note`: Mirrors the MongoDB document (`_id`, `title`, `content`, `pinned`, `createdAt`, `updatedAt`).
   - `CreateNoteRequest`: Enforces mandatory fields (`title`, `content`) via `binding:"required"`.
   - `UpdateNoteRequest`: Uses Go pointers (`*string`, `*bool`) to distinguish between an omitted field (`nil`) and an explicit empty or false field (`""` or `false`).
2. **Cursor Pagination vs Offset Pagination**:
   - Instead of costly `skip()` queries that degrade with large datasets ($O(N)$), this API queries `_id: { $gt: objID }` with an index on `_id` for constant-time ($O(1)$) pagination.
   - It fetches `limit + 1` records to determine whether `hasMore` is `true` without needing an extra `countDocuments` query.
3. **Context Timeouts & Connection Safety**:
   - Every repository operation runs under a 5-second timeout (`context.WithTimeout`).
   - The database connection initializes with a 10-second timeout and actively verifies server reachability with `client.Ping`.
   - The `main.go` entrypoint defers `db.Disconnect(client)` to ensure connection pools close gracefully when the application stops.

---

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.
