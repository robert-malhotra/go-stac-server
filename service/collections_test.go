package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/planetlabs/go-stac" // adjust import if necessary
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func init() {
	// Configure the DSN from an environment variable or use a default.
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "postgresql://username:password@localhost:5439/postgis"
	}
	viper.Set("database.dsn", dsn)
}

func TestServiceCRUDIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create an instance of the decoupled CollectionsService.
	svc := NewSTACService()

	// Generate a unique collection id to avoid conflicts.
	collectionID := fmt.Sprintf("test-collection-%d", time.Now().UnixNano())

	// Create a new stac.Collection.
	collection := &stac.Collection{
		Id:    collectionID,
		Title: "Integration Test Collection",
	}

	// --- CREATE ---
	created, err := svc.CreateCollection(ctx, collection)
	require.NoError(t, err, "failed to create collection")
	require.NotNil(t, created, "created collection should not be nil")
	require.Equal(t, collectionID, created.Id, "created collection id should match")

	// --- GET ---
	retrieved, err := svc.GetCollection(ctx, collectionID)
	require.NoError(t, err, "failed to retrieve collection")
	require.NotNil(t, retrieved, "retrieved collection should not be nil")
	require.Equal(t, collectionID, retrieved.Id, "retrieved collection id should match created id")

	// --- UPDATE ---
	// Update the Title property.
	newTitle := "Updated Integration Test Collection"
	collection.Title = newTitle

	updated, err := svc.UpdateCollection(ctx, collection)
	require.NoError(t, err, "failed to update collection")
	require.NotNil(t, updated, "updated collection should not be nil")
	require.Equal(t, newTitle, updated.Title, "updated collection title should match")

	// --- LIST ---
	// Retrieve a list of collections and verify that our collection is present.
	collections, err := svc.ListCollections(ctx)
	require.NoError(t, err, "failed to list collections")
	found := false
	for _, coll := range collections {
		if coll.Id == collectionID {
			found = true
			break
		}
	}
	require.True(t, found, "created collection should appear in the list")

	// --- DELETE ---
	err = svc.DeleteCollection(ctx, collectionID)
	require.NoError(t, err, "failed to delete collection")

	// Verify deletion by attempting to get the deleted collection.
	_, err = svc.GetCollection(ctx, collectionID)
	require.Error(t, err, "expected error when retrieving a deleted collection")
}
