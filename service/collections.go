package service

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/planetlabs/go-stac"
	"github.com/spf13/viper"

	"github.com/goccy/go-json"
	"github.com/jackc/pgx/v5"

	"github.com/go-geospatial/go-stac-server/database"
)

// init configures the database DSN from an environment variable or a default.
func init() {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "postgresql://username:password@localhost:5439/postgis"
	}
	viper.Set("database.dsn", dsn)
}

// STACService provides CRUD operations for both STAC collections and items.
type STACService struct{}

// NewSTACService creates and returns a new STACService instance.
func NewSTACService() *STACService {
	return &STACService{}
}

// =======================================================
//                   COLLECTIONS METHODS
// =======================================================

// CreateCollection inserts a new collection into the database.
func (s *STACService) CreateCollection(ctx context.Context, collection *stac.Collection) (*stac.Collection, error) {
	if collection == nil {
		return nil, errors.New("collection is nil")
	}
	if collection.Id == "" {
		return nil, errors.New("collection id is required")
	}
	collectionJSON, err := json.Marshal(collection)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection: %w", err)
	}

	pool := database.GetInstance(ctx)
	// Assumes a stored procedure create_collection(jsonb) exists.
	if _, err = pool.Exec(ctx, "SELECT create_collection($1::jsonb)", collectionJSON); err != nil {
		return nil, fmt.Errorf("failed to create collection with id %s: %w", collection.Id, err)
	}

	// Return the created collection.
	return s.GetCollection(ctx, collection.Id)
}

// GetCollection retrieves a collection by its ID.
func (s *STACService) GetCollection(ctx context.Context, collectionID string) (*stac.Collection, error) {
	if collectionID == "" {
		return nil, errors.New("collection id is required")
	}

	pool := database.GetInstance(ctx)
	var rawCollection string
	err := pool.QueryRow(ctx, "SELECT get_collection($1::text)", collectionID).Scan(&rawCollection)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("collection '%s' not found", collectionID)
		}
		return nil, fmt.Errorf("failed to retrieve collection: %w", err)
	}

	var collection stac.Collection
	if err = json.Unmarshal([]byte(rawCollection), &collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection: %w", err)
	}
	return &collection, nil
}

// UpdateCollection updates an existing collection.
func (s *STACService) UpdateCollection(ctx context.Context, collection *stac.Collection) (*stac.Collection, error) {
	if collection == nil {
		return nil, errors.New("collection is nil")
	}
	if collection.Id == "" {
		return nil, errors.New("collection id is required")
	}
	collectionJSON, err := json.Marshal(collection)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection: %w", err)
	}

	pool := database.GetInstance(ctx)
	// Assumes a stored procedure update_collection(jsonb) exists.
	if _, err = pool.Exec(ctx, "SELECT update_collection($1::jsonb)", collectionJSON); err != nil {
		return nil, fmt.Errorf("failed to update collection with id %s: %w", collection.Id, err)
	}

	return s.GetCollection(ctx, collection.Id)
}

// ListCollections returns all collections from the database.
func (s *STACService) ListCollections(ctx context.Context) ([]*stac.Collection, error) {
	pool := database.GetInstance(ctx)
	rows, err := pool.Query(ctx, "SELECT id, content FROM pgstac.collections ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("error querying collections: %w", err)
	}
	defer rows.Close()

	collections := make([]*stac.Collection, 0)
	for rows.Next() {
		var collectionID string
		var rawCollection string
		if err := rows.Scan(&collectionID, &rawCollection); err != nil {
			return nil, fmt.Errorf("error scanning collection row: %w", err)
		}

		var collection stac.Collection
		if err := json.Unmarshal([]byte(rawCollection), &collection); err != nil {
			return nil, fmt.Errorf("failed to unmarshal collection JSON for id %s: %w", collectionID, err)
		}

		collections = append(collections, &collection)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collections rows: %w", err)
	}

	return collections, nil
}

// DeleteCollection removes a collection by its ID.
func (s *STACService) DeleteCollection(ctx context.Context, collectionID string) error {
	if collectionID == "" {
		return errors.New("collection id is required")
	}

	pool := database.GetInstance(ctx)
	if _, err := pool.Exec(ctx, "SELECT delete_collection($1::text)", collectionID); err != nil {
		return fmt.Errorf("failed to delete collection %s: %w", collectionID, err)
	}
	return nil
}
