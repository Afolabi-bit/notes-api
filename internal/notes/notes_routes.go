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
	notesGroup.GET("", handler.ListNotes)
	notesGroup.GET("/:id", handler.GetNoteByID)
	notesGroup.PATCH("/:id", handler.UpdateNoteByID)
	notesGroup.DELETE("/:id", handler.DeleteNoteByID)
}
