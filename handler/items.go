// item_handler.go
package handler

import (
	"fmt"

	"github.com/planetlabs/go-stac"

	"github.com/go-geospatial/go-stac-server/service"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// stacService is a global (or injected) instance of our service layer.
// (You could also inject this into the handler via a struct or other means.)
var stacService = service.NewSTACService()

// DeleteItem deletes an item from a collection.
// Endpoint: DELETE /collections/:collectionId/items/:itemId
func DeleteItem(c *fiber.Ctx) error {
	// Use c.Context() so that cancellation/deadlines are respected.
	ctx := c.Context()
	collectionID := c.Params("collectionId")
	itemID := c.Params("itemId")

	// Delegate the deletion to the service layer.
	if err := stacService.DeleteItem(ctx, collectionID, itemID); err != nil {
		log.Error().Err(err).Msg("failed to delete item")
		return c.Status(fiber.StatusNotFound).JSON(Message{
			Code:        "DeleteItemFailed",
			Description: fmt.Sprintf("failed to delete item %q in collection %q", itemID, collectionID),
		})
	}

	return c.JSON(Message{
		Code:        "ItemDeleted",
		Description: "the item has been deleted",
	})
}

// UpdateItem updates an existing item.
// Endpoint: PUT /collections/:collectionId/items/:itemId
func UpdateItem(c *fiber.Ctx) error {
	ctx := c.Context()
	collectionID := c.Params("collectionId")
	itemID := c.Params("itemId")

	// Parse the request body into a stac.Item.
	var item stac.Item
	if err := c.BodyParser(&item); err != nil {
		log.Error().Err(err).Msg("failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(Message{
			Code:        "PutItemFailed",
			Description: "failed to parse request body as JSON",
		})
	}

	// Validate that the item’s IDs match the URL parameters.
	if item.Id != itemID || item.Collection != collectionID {
		return c.Status(fiber.StatusBadRequest).JSON(Message{
			Code:        "ModifyItemFailed",
			Description: "item id or collection id in request body does not match URL",
		})
	}

	// Call the service to update the item.
	updatedItem, err := stacService.UpdateItem(ctx, &item)
	if err != nil {
		log.Error().Err(err).Msg("failed to update item")
		return c.Status(fiber.StatusNotFound).JSON(Message{
			Code:        "PutItemFailed",
			Description: fmt.Sprintf("collection %q does not contain an item with id %q", collectionID, itemID),
		})
	}

	return c.JSON(updatedItem)
}

// PatchItem applies a partial update to an item.
// Endpoint: PATCH /collections/:collectionId/items/:itemId
func PatchItem(c *fiber.Ctx) error {
	ctx := c.Context()
	collectionID := c.Params("collectionId")
	itemID := c.Params("itemId")

	// Parse the request body into a generic patch map.
	var patch map[string]interface{}
	if err := c.BodyParser(&patch); err != nil {
		log.Error().Err(err).Msg("failed to parse patch body")
		return c.Status(fiber.StatusBadRequest).JSON(Message{
			Code:        "PatchItemFailed",
			Description: "failed to parse request body as JSON",
		})
	}

	// Call the service to perform the patch.
	patchedItem, err := stacService.PatchItem(ctx, collectionID, itemID, patch)
	if err != nil {
		log.Error().Err(err).Msg("failed to patch item")
		return c.Status(fiber.StatusInternalServerError).JSON(Message{
			Code:        "PatchItemFailed",
			Description: "failed to patch item",
		})
	}

	return c.JSON(patchedItem)
}

// CreateItems creates one or more items.
// Endpoint: POST /collections/:collectionId/items
//
// This handler distinguishes between a single “Feature” and a “FeatureCollection”
// and then calls the appropriate service method.
func CreateItems(c *fiber.Ctx) error {
	ctx := c.Context()
	collectionID := c.Params("collectionId")

	// First, decode just the "type" field to decide which payload we have.
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := c.BodyParser(&typeCheck); err != nil {
		log.Error().Err(err).Msg("failed to parse JSON body for type")
		return c.Status(fiber.StatusBadRequest).JSON(Message{
			Code:        "CreateItemsFailed",
			Description: "failed to parse JSON body",
		})
	}

	switch typeCheck.Type {
	case "Feature":
		// A single feature; decode to stac.Item.
		var item stac.Item
		if err := c.BodyParser(&item); err != nil {
			log.Error().Err(err).Msg("failed to parse item")
			return c.Status(fiber.StatusBadRequest).JSON(Message{
				Code:        "CreateItemFailed",
				Description: "failed to parse item JSON",
			})
		}
		// Validate that the item belongs to the collection specified by the URL.
		if item.Collection != collectionID {
			return c.Status(fiber.StatusBadRequest).JSON(Message{
				Code:        "CreateItemFailed",
				Description: "collection id in request body does not match URL",
			})
		}
		createdItem, err := stacService.CreateItem(ctx, &item)
		if err != nil {
			log.Error().Err(err).Msg("failed to create item")
			return c.Status(fiber.StatusConflict).JSON(Message{
				Code:        "CreateItemFailed",
				Description: "failed to create item",
			})
		}
		return c.JSON(createdItem)

	case "FeatureCollection":
		// A feature collection; decode to stac.ItemsList.
		var itemsList stac.ItemsList
		if err := c.BodyParser(&itemsList); err != nil {
			log.Error().Err(err).Msg("failed to parse items list")
			return c.Status(fiber.StatusBadRequest).JSON(Message{
				Code:        "CreateItemsFailed",
				Description: "failed to parse items JSON",
			})
		}
		// Optionally, you can validate that each item in the list has the proper collection.
		createdItems, err := stacService.CreateItems(ctx, itemsList)
		if err != nil {
			log.Error().Err(err).Msg("failed to create items")
			return c.Status(fiber.StatusConflict).JSON(Message{
				Code:        "CreateItemsFailed",
				Description: "failed to create items",
			})
		}
		return c.JSON(createdItems)

	default:
		return c.Status(fiber.StatusBadRequest).JSON(Message{
			Code:        "CreateItemsFailed",
			Description: "invalid geojson type; must be 'Feature' or 'FeatureCollection'",
		})
	}
}

// GetItem returns the details of a specific item.
// Endpoint: GET /collections/:collectionId/items/:itemId
func GetItem(c *fiber.Ctx) error {
	ctx := c.Context()
	collectionID := c.Params("collectionId")
	itemID := c.Params("itemId")

	item, err := stacService.GetItem(ctx, collectionID, itemID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get item")
		return c.Status(fiber.StatusNotFound).JSON(Message{
			Code:        "ItemNotFound",
			Description: "item not found",
		})
	}
	return c.JSON(item)
}

// ListItems returns a list of items from a collection using query parameters.
// Endpoint: GET /collections/:collectionId/items
func ListItems(c *fiber.Ctx) error {
	ctx := c.Context()
	collectionID := c.Params("collectionId")

	// Convert query parameters (e.g. limit, token, etc.) into a CQL struct.
	cql, err := getCQLFromQuery(c)
	if err != nil {
		// buildCQLFromQuery can write an error response.
		return nil
	}

	// Force the CQL search to only this collection.
	cql.Collections = []string{collectionID}

	// Delegate the search to the service.
	result, err := stacService.ListItems(ctx, collectionID, cql)
	if err != nil {
		log.Error().Err(err).Msg("failed to list items")
		return c.Status(fiber.StatusInternalServerError).JSON(Message{
			Code:        "ServerError",
			Description: "failed to list items",
		})
	}
	return c.JSON(result)
}
