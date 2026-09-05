package notes

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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

	response := APIResponse[Note]{
		Status:  "success",
		Message: "Note created successfully",
		Data:    createdNote,
	}

	c.JSON(http.StatusCreated, response)

}

func (h *Handler) ListNotes(c *gin.Context) {
	lastID := c.Query("nextCursor")

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'limit' must be a positive integer"})
		return
	}

	notes, err := h.repo.List(c.Request.Context(), lastID, limit+1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list notes"})
		return
	}

	hasMore := len(notes) > int(limit)
	if hasMore {
		notes = notes[:limit]
	}

	nextCursor := ""
	if hasMore && len(notes) > 0 {
		nextCursor = notes[len(notes)-1].ID.Hex()
	}

	response := APIResponse[[]Note]{
		Status:  "success",
		Message: "Notes retrieved successfully",
		Pagination: &PaginationMeta{
			Limit:      limit,
			NextCursor: nextCursor,
			HasMore:    hasMore,
			PageLength: int64(len(notes)),
		},
		Data: notes,
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetNoteByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Note ID is required"})
		return
	}

	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note ID"})
		return
	}

	note, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Note not found for that ID"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get note"})
		return
	}

	response := APIResponse[Note]{
		Status:  "success",
		Message: "Note retrieved successfully",
		Data:    note,
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateNoteByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Note ID is required"})
		return
	}

	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note ID"})
		return
	}

	var req UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedNote, err := h.repo.UpdateByID(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Note not found for that ID"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update note"})
		return
	}

	response := APIResponse[Note]{
		Status:  "success",
		Message: "Note updated successfully",
		Data:    updatedNote,
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) DeleteNoteByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Note ID is required"})
		return
	}

	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note ID"})
		return
	}

	deletedNote, err := h.repo.DeleteByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Note not found for that ID"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete note"})
		return
	}

	if !deletedNote {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found for that ID"})
		return
	}
	response := APIResponse[any]{
		Status:  "success",
		Message: "Note deleted successfully",
	}

	c.JSON(http.StatusOK, response)
}
