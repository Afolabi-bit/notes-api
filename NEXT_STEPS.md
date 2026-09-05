# Next Steps & Practice Roadmap: Go + Gin + MongoDB API

Congratulations on finishing the base CRUD API! This roadmap provides targeted exercises designed to take this project from a beginner tutorial to a production-ready Go backend service.

---

## Roadmap Overview

```
Phase 1: Feature Expansion ──► Phase 2: Architecture & Testing ──► Phase 3: Auth & Production
  • Pagination & Filtering       • Interface Abstraction           • JWT Authentication
  • Partial Updates (PATCH)      • Unit Testing (httptest)         • Multi-Tenancy
  • Query Optimization           • Service Layer Separation        • Graceful Shutdown
```

---

## Phase 1: Feature Expansion (API & Mongo Driver v2 Skills)

### 1. Pagination, Sorting & Filtering (`GET /notes`)
Currently, `r.coll.Find(opCtx, filter)` returns all documents into memory. In production, collections with thousands of documents will cause high latency and memory spikes.

- **Goal**: Support query parameters: `GET /notes?page=1&limit=10&pinned=true&sort=desc`
- **Concepts**:
  - Reading query parameters with Gin: `c.DefaultQuery("page", "1")`, `c.Query("pinned")`.
  - Type parsing and validation (`strconv.Atoi`).
  - Mongo Driver v2 builder options:
    ```go
    findOpts := options.Find().
        SetSkip(int64((page - 1) * limit)).
        SetLimit(int64(limit)).
        SetSort(bson.D{{Key: "createdAt", Value: -1}})
    ```
  - Dynamic BSON query construction:
    ```go
    filter := bson.M{}
    if pinnedStr != "" {
        filter["pinned"] = (pinnedStr == "true")
    }
    ```
- **Bonus**: Return pagination metadata in the response:
  ```json
  {
    "data": [...],
    "pagination": {
      "currentPage": 1,
      "limit": 10,
      "totalRecords": 42,
      "totalPages": 5
    }
  }
  ```

---

### 2. Partial Updates (`PATCH /notes/:id`)
Currently, `PUT` replaces all fields (`title`, `content`, `pinned`). If a client only wants to pin a note, sending only `{"pinned": true}` with `PUT` can zero out `title` and `content`.

- **Goal**: Implement `PATCH /notes/:id` to update only provided fields.
- **Concepts**:
  - Using pointer fields in Go structs to distinguish between **omitted** fields (`nil`) and **zero values** (`""` or `false`):
    ```go
    type PatchNoteRequest struct {
        Title   *string `json:"title"`
        Content *string `json:"content"`
        Pinned  *bool   `json:"pinned"`
    }
    ```
  - Building dynamic `$set` maps:
    ```go
    setFields := bson.M{"updatedAt": time.Now().UTC()}
    if req.Title != nil {
        setFields["title"] = *req.Title
    }
    if req.Content != nil {
        setFields["content"] = *req.Content
    }
    if req.Pinned != nil {
        setFields["pinned"] = *req.Pinned
    }
    update := bson.M{"$set": setFields}
    ```

---

### 3. Query Optimization: `DeleteOne` vs `FindOneAndDelete`
- **Observation**: In `DeleteByID`, `FindOneAndDelete` fetches and deserializes the entire document from disk before deleting it.
- **Improvement**: If you only need to know whether the document existed and was deleted, switch to `r.coll.DeleteOne`:
  ```go
  res, err := r.coll.DeleteOne(opCtx, bson.M{"_id": id})
  if err != nil {
      return false, err
  }
  return res.DeletedCount > 0, nil
  ```
  This is significantly faster and consumes less bandwidth and memory.

---

## Phase 2: Architecture & Testing (Writing Idiomatic Go)

### 4. Repository Interfaces & Unit Testing
Currently, the `Handler` struct tightly couples to the concrete `*Repo` pointer.

- **Goal**: Decouple the handler using a Go interface to enable fast unit testing without a live MongoDB database.
- **Concepts**:
  - Define the interface where it is consumed (or in the `notes` package):
    ```go
    type NoteRepository interface {
        Create(ctx context.Context, note Note) (Note, error)
        List(ctx context.Context) ([]Note, error)
        GetByID(ctx context.Context, id bson.ObjectID) (Note, error)
        UpdateByID(ctx context.Context, id bson.ObjectID, req UpdateNoteRequest) (Note, error)
        DeleteByID(ctx context.Context, id bson.ObjectID) (bool, error)
    }

    type Handler struct {
        repo NoteRepository
    }
    ```
  - Create a mock repository for tests:
    ```go
    type MockRepo struct {
        MockGetByID func(ctx context.Context, id bson.ObjectID) (Note, error)
    }
    func (m *MockRepo) GetByID(ctx context.Context, id bson.ObjectID) (Note, error) {
        return m.MockGetByID(ctx, id)
    }
    // implement remaining methods...
    ```
  - Write HTTP tests using `net/http/httptest`:
    ```go
    func TestGetNoteByID_NotFound(t *testing.T) {
        gin.SetMode(gin.TestMode)
        mock := &MockRepo{
            MockGetByID: func(ctx context.Context, id bson.ObjectID) (Note, error) {
                return Note{}, mongo.ErrNoDocuments
            },
        }
        h := NewHandler(mock)
        r := gin.Default()
        r.GET("/notes/:id", h.GetNoteByID)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest("GET", "/notes/64b5f8e5b6a7c8d9e0f1a2b3", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusNotFound, w.Code)
    }
    ```

---

### 5. Add a Service Layer (Clean Architecture)
Currently, handlers directly call database methods and perform business logic (like ID generation and timestamps).

- **Goal**: Split concerns into three distinct layers:
  1. **Transport Layer (Handler)**: HTTP status codes, request decoding, query parsing, calling service.
  2. **Business Layer (Service)**: Validations (e.g., maximum pinned notes allowed, word count limit), data enrichment, calling repository.
  3. **Data Layer (Repository)**: Raw queries, BSON decoding, database transactions.

---

## Phase 3: Real-World Features (Multi-Tenancy & Production Readiness)

### 6. User Authentication & Multi-Tenancy
Turn this single-user API into a real SaaS backend where users have private notes.

- **Tasks**:
  1. **User Model & Repo**: Create a `users` collection with `_id`, `email`, `passwordHash`, and `createdAt`.
  2. **Auth Endpoints**:
     - `POST /auth/register` (hash password with `golang.org/x/crypto/bcrypt`).
     - `POST /auth/login` (compare hash, generate signed JWT using `github.com/golang-jwt/jwt/v5`).
  3. **Auth Middleware**:
     - Inspect `Authorization: Bearer <token>` header.
     - Verify signature and store `userID` in `c.Set("userID", claims.UserID)`.
  4. **Scoped Note Operations**:
     - Add `UserID bson.ObjectID` to `Note`.
     - In every repository query, enforce ownership:
       ```go
       filter := bson.M{"_id": noteID, "userId": userID}
       ```
       This prevents User A from reading, updating, or deleting User B's notes.

---

### 7. Graceful Server Shutdown
If the container or server terminates while a request or database write is active, requests can fail abruptly.

- **Goal**: Catch `SIGINT` (Ctrl+C) and `SIGTERM`, stop accepting new connections, finish existing requests, and close the MongoDB pool gracefully.
- **Pattern (`main.go`)**:
  ```go
  srv := &http.Server{
      Addr:    ":" + port,
      Handler: router,
  }

  go func() {
      if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
          log.Fatalf("listen: %s\n", err)
      }
  }()

  quit := make(chan os.Signal, 1)
  signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
  <-quit
  log.Println("Shutting down server...")

  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()

  if err := srv.Shutdown(ctx); err != nil {
      log.Fatal("Server forced to shutdown:", err)
  }

  // Disconnect MongoDB client
  if err := client.Disconnect(ctx); err != nil {
      log.Fatal("Mongo disconnect failed:", err)
  }
  log.Println("Server exited cleanly")
  ```

---

## Quick Reference: Recommended Order of Execution

| Step | Topic | Difficulty | Key Takeaway |
|:---|:---|:---:|:---|
| **1** | Pagination & Query Filters | Easy | Master Gin query params & Mongo v2 `options.Find()` |
| **2** | `PATCH` Partial Updates | Easy-Medium | Pointers in structs for optional JSON fields |
| **3** | Replace with `DeleteOne` | Easy | Understanding performance differences in Mongo operations |
| **4** | Repository Interfaces & Unit Tests | Medium | Decoupled code, testable handlers with `httptest` |
| **5** | Service Layer | Medium | Clean separation of business logic vs HTTP |
| **6** | JWT Auth & Scoped Queries | Advanced | Real-world multi-tenant authorization |
| **7** | Graceful Shutdown | Medium | Reliable production application lifecycle |
