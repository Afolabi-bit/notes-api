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
	Pinned  bool   `json:"pinned"`
}
type UpdateNoteRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Pinned  bool   `json:"pinned"`
}

type NoteFilter struct {
	Pinned *bool  `json:"pinned,omitempty"`
	Search string `json:"search,omitempty"`
}

type PaginationMeta struct {
	Limit      int64  `json:"limit"`
	NextCursor string `json:"nextCursor"`
	HasMore    bool   `json:"hasMore"`
	PageLength int64  `json:"pageLength"`
}

//	type PaginatedResponse struct {
//		Status     string         `json:"status"`
//		Pagination PaginationMeta `json:"pagination"`
//		Data       []Note         `json:"data"`
//	}

type APIResponse[T any] struct {
	Status     string          `json:"status"`
	Message    string          `json:"message,omitempty"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
	Data       T               `json:"data,omitempty"`
}
