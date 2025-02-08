// Package service provides a pure ItemsService that encapsulates
// CRUD operations for STAC items, wrapping the database and using
// Planet Labs’ Go‑STAC types.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-geospatial/go-stac-server/database"
	"github.com/go-geospatial/go-stac-server/jsonutil"
	"github.com/goccy/go-json"
	"github.com/jackc/pgx/v5"
	"github.com/planetlabs/go-stac"
)

// CreateItem inserts a new item into the database.
func (s *STACService) CreateItem(ctx context.Context, item *stac.Item) (*stac.Item, error) {
	if item == nil {
		return nil, errors.New("item is nil")
	}
	if item.Id == "" {
		return nil, errors.New("item id is required")
	}
	if item.Collection == "" {
		return nil, errors.New("item collection is required")
	}

	itemJSON, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item to JSON: %w", err)
	}

	pool := database.GetInstance(ctx)
	// Assumes a stored procedure create_item(jsonb) exists.
	if _, err = pool.Exec(ctx, "SELECT create_item($1::jsonb)", itemJSON); err != nil {
		return nil, fmt.Errorf("failed to create item with id %s: %w", item.Id, err)
	}

	return s.GetItem(ctx, item.Collection, item.Id)
}

// UpdateItem updates an existing item in the database.
func (s *STACService) UpdateItem(ctx context.Context, item *stac.Item) (*stac.Item, error) {
	if item == nil {
		return nil, errors.New("item is nil")
	}
	if item.Id == "" {
		return nil, errors.New("item id is required")
	}
	if item.Collection == "" {
		return nil, errors.New("item collection is required")
	}

	itemJSON, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item to JSON: %w", err)
	}

	pool := database.GetInstance(ctx)
	// Assumes a stored procedure update_item(jsonb) exists.
	if _, err = pool.Exec(ctx, "SELECT update_item($1::jsonb)", itemJSON); err != nil {
		return nil, fmt.Errorf("failed to update item with id %s: %w", item.Id, err)
	}

	return s.GetItem(ctx, item.Collection, item.Id)
}

// PatchItem applies a partial update to an item.
func (s *STACService) PatchItem(ctx context.Context, collectionID, itemID string, patch map[string]interface{}) (*stac.Item, error) {
	if collectionID == "" || itemID == "" {
		return nil, errors.New("collectionID and itemID are required")
	}

	pool := database.GetInstance(ctx)
	var rawItem string
	err := pool.QueryRow(ctx, "SELECT get_item($1::text, $2::text)", itemID, collectionID).Scan(&rawItem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("item '%s' not found in collection '%s'", itemID, collectionID)
		}
		return nil, fmt.Errorf("failed to retrieve item: %w", err)
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch: %w", err)
	}
	//TODO do we need to do this?
	mergedBytes, err := jsonutil.Merge(patchBytes, []byte(rawItem))
	if err != nil {
		return nil, fmt.Errorf("failed to merge patch with item: %w", err)
	}

	if _, err = pool.Exec(ctx, "SELECT update_item($1::jsonb)", mergedBytes); err != nil {
		return nil, fmt.Errorf("failed to update item with id %s: %w", itemID, err)
	}

	return s.GetItem(ctx, collectionID, itemID)
}

// DeleteItem deletes an item by its ID from the specified collection.
func (s *STACService) DeleteItem(ctx context.Context, collectionID, itemID string) error {
	if collectionID == "" || itemID == "" {
		return errors.New("collectionID and itemID are required")
	}

	pool := database.GetInstance(ctx)
	if _, err := pool.Exec(ctx, "SELECT delete_item($1::text, $2::text)", itemID, collectionID); err != nil {
		return fmt.Errorf("failed to delete item with id %s: %w", itemID, err)
	}
	return nil
}

// GetItem retrieves an item by its ID from the specified collection.
func (s *STACService) GetItem(ctx context.Context, collectionID, itemID string) (*stac.Item, error) {
	if collectionID == "" || itemID == "" {
		return nil, errors.New("collectionID and itemID are required")
	}

	pool := database.GetInstance(ctx)
	var rawItem string
	err := pool.QueryRow(ctx, "SELECT get_item($1::text, $2::text)", itemID, collectionID).Scan(&rawItem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("item '%s' not found in collection '%s'", itemID, collectionID)
		}
		return nil, fmt.Errorf("failed to retrieve item: %w", err)
	}

	var item stac.Item
	if err := json.Unmarshal([]byte(rawItem), &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal item JSON for id %s: %w", itemID, err)
	}

	return &item, nil
}

// ListItems returns a FeatureCollection of items in the specified collection.
// The search criteria is provided as a stac.CQL object.
func (s *STACService) ListItems(ctx context.Context, collectionID string, cql CQL) (*SearchResponse, error) {
	if collectionID == "" {
		return nil, errors.New("collectionID is required")
	}
	if cql.Limit == 0 {
		cql.Limit = 100 //TODO: make configurable
	}

	// Ensure that the CQL is restricted to the provided collection.
	cql.Collections = []string{collectionID}

	// stac.Search is assumed to perform the query and return a FeatureCollection.
	featureCollection, err := Search(cql)
	if err != nil {
		return nil, fmt.Errorf("stac search returned an error: %w", err)
	}

	return featureCollection, nil
}

// // GetItemsByIDs retrieves items matching the supplied list of IDs from the specified collection.
func (s *STACService) GetItemsByIDs(ctx context.Context, collectionID string, ids []string) (*SearchResponse, error) {
	if collectionID == "" {
		return nil, errors.New("collectionID is required")
	}
	if len(ids) == 0 {
		return nil, errors.New("no item IDs provided")
	}

	cql := CQL{
		Collections: []string{collectionID},
		Ids:         ids,
		Limit:       100, //TODO: make configurable?
	}

	items, err := Search(cql)
	if err != nil {
		return nil, fmt.Errorf("stac search returned an error: %w", err)
	}

	return items, nil
}
