package notes

// Repo -data acces layer

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

func (r *Repo) List(ctx context.Context, nextCursor string, limit int64, filter NoteFilter) ([]Note, error) {

	query := bson.M{}

	if nextCursor != "" {
		objID, err := bson.ObjectIDFromHex(nextCursor)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse nextCursor: %w", err)
		}

		query["_id"] = bson.M{"$gt": objID}
	}

	if filter.Pinned != nil {
		query["pinned"] = *filter.Pinned
	}

	if filter.Search != "" {
		query["title"] = bson.M{"$regex": filter.Search, "$options": "i"}
	}

	findOptions := options.Find()
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{Key: "_id", Value: 1}})

	opCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	cursor, err := r.coll.Find(opCtx, query, findOptions)
	if err != nil {
		return nil, fmt.Errorf("Failed to find notes: %w", err)
	}

	// cursor must be closed to avoid memory leaks
	defer cursor.Close(opCtx)

	var notes []Note

	if err := cursor.All(opCtx, &notes); err != nil {
		return nil, fmt.Errorf("Failed to decode notes: %w", err)
	}

	if notes == nil {
		notes = []Note{}
	}

	return notes, nil
}

func (r *Repo) GetByID(ctx context.Context, id bson.ObjectID) (Note, error) {

	opCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	filter := bson.M{"_id": id}
	var note Note

	err := r.coll.FindOne(opCtx, filter, options.FindOne()).Decode(&note)

	if err != nil {
		return Note{}, fmt.Errorf("Failed to find note: %w", err)
	}

	return note, nil
}

func (r *Repo) UpdateByID(ctx context.Context, id bson.ObjectID, req UpdateNoteRequest) (Note, error) {

	opCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	filter := bson.M{"_id": id}

	fields := bson.M{
		"updatedAt": time.Now().UTC(),
	}
	if req.Title != nil {
		fields["title"] = *req.Title
	}

	if req.Content != nil {
		fields["content"] = *req.Content
	}

	if req.Pinned != nil {
		fields["pinned"] = *req.Pinned
	}

	updateFields := bson.M{
		"$set": fields,
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedNote Note
	err := r.coll.FindOneAndUpdate(opCtx, filter, updateFields, opts).Decode(&updatedNote)
	if err != nil {
		return Note{}, fmt.Errorf("Failed to update note: %w", err)
	}

	return updatedNote, nil
}

func (r *Repo) DeleteByID(ctx context.Context, id bson.ObjectID) (bool, error) {

	opCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	filter := bson.M{"_id": id}
	var deletedNote Note

	err := r.coll.FindOneAndDelete(opCtx, filter, options.FindOneAndDelete()).Decode(&deletedNote)
	if err != nil {
		return false, fmt.Errorf("Failed to delete note: %w", err)
	}

	return true, nil
}
