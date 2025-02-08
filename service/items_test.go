// items_service_test.go
package service_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-geospatial/go-stac-server/service"
	"github.com/planetlabs/go-stac" // using Planet Labs' Go-STAC types
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

// TestItemsServiceCRUDIntegration performs an end-to-end test of the STACService CRUD methods for items,
// including the new ListItems and GetItemsByIDs functions.
func TestItemsServiceCRUDIntegration(t *testing.T) {
	// Use a context with a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Instantiate the unified STACService.
	svc := service.NewSTACService()

	// Generate unique collection and item IDs.
	testCollection := stac.Collection{
		Id:    fmt.Sprintf("test-collection-%d", time.Now().UnixNano()),
		Title: "Integration Test Collection",
	}
	itemID := fmt.Sprintf("test-item-%d", time.Now().UnixNano())

	// --- SETUP: Create a dummy collection ---
	// Many stored procedures expect that the collection exists.
	// Here we insert a minimal dummy collection into the database.
	_, err := svc.CreateCollection(context.Background(), &testCollection)
	require.NoError(t, err, "failed to create dummy collection")

	// --- CREATE ---
	// Create a new item. Note that we now include a valid geometry as a GeoJSON object.
	item := &stac.Item{
		Id:         itemID,
		Collection: testCollection.Id,
		Geometry: map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{0.0, 0.0},
		},
		Properties: map[string]interface{}{
			"title":    "Test Item",
			"datetime": time.Now().Format(time.RFC3339),
		},
	}

	created, err := svc.CreateItem(ctx, item)
	require.NoError(t, err, "failed to create item")
	require.NotNil(t, created, "created item should not be nil")
	require.Equal(t, itemID, created.Id, "created item id should match")
	require.Equal(t, testCollection.Id, created.Collection, "created item collection should match")
	require.Equal(t, "Test Item", created.Properties["title"], "initial item title should match")

	// --- GET ---
	retrieved, err := svc.GetItem(ctx, testCollection.Id, itemID)
	require.NoError(t, err, "failed to retrieve item")
	require.NotNil(t, retrieved, "retrieved item should not be nil")
	require.Equal(t, itemID, retrieved.Id, "retrieved item id should match")
	require.Equal(t, testCollection.Id, retrieved.Collection, "retrieved item collection should match")

	// --- UPDATE ---
	// Update the item's title.
	created.Properties["title"] = "Updated Test Item"
	updated, err := svc.UpdateItem(ctx, created)
	require.NoError(t, err, "failed to update item")
	require.NotNil(t, updated, "updated item should not be nil")
	require.Equal(t, "Updated Test Item", updated.Properties["title"], "updated item title should match")

	// --- PATCH ---
	// Prepare a patch to update the title to a new value.
	patch := map[string]interface{}{
		"properties": map[string]interface{}{
			"title": "Patched Test Item",
		},
	}
	patched, err := svc.PatchItem(ctx, testCollection.Id, itemID, patch)
	require.NoError(t, err, "failed to patch item")
	require.NotNil(t, patched, "patched item should not be nil")
	require.Equal(t, "Patched Test Item", patched.Properties["title"], "patched item title should match")

	// --- NEW: LIST ITEMS ---
	// Test the ListItems function.
	// We assume that service.CQL is defined and that the SearchResponse type includes a Features field,
	// where each feature has an Id field.
	listResponse, err := svc.ListItems(ctx, testCollection.Id, service.CQL{})
	require.NoError(t, err, "failed to list items")
	require.NotNil(t, listResponse, "list items response should not be nil")
	found := false
	for _, feature := range listResponse.Features {
		if feature.Id == itemID {
			found = true
			break
		}
	}
	require.True(t, found, "created item should be found in list items")

	// --- NEW: GET ITEMS BY IDS ---
	// Test the GetItemsByIDs function.
	itemsByIDsResponse, err := svc.GetItemsByIDs(ctx, testCollection.Id, []string{itemID})
	require.NoError(t, err, "failed to get items by IDs")
	require.NotNil(t, itemsByIDsResponse, "get items by IDs response should not be nil")
	found = false
	for _, feature := range itemsByIDsResponse.Features {
		if feature.Id == itemID {
			found = true
			break
		}
	}
	require.True(t, found, "created item should be found in getItemsByIDs response")

	// --- DELETE ---
	err = svc.DeleteItem(ctx, testCollection.Id, itemID)
	require.NoError(t, err, "failed to delete item")

	// Verify deletion by attempting to retrieve the deleted item.
	_, err = svc.GetItem(ctx, testCollection.Id, itemID)
	require.Error(t, err, "expected error when retrieving deleted item")
}
