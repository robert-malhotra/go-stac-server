// collection_handler.go
package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/planetlabs/go-stac"
	"github.com/rs/zerolog/log"
)

// ModifyCollection handles both creation (POST) and update (PUT) of a collection.
// Endpoint: POST /collections   and   PUT /collections
func ModifyCollection(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse the request body into a stac.Collection.
	var coll stac.Collection
	if err := c.BodyParser(&coll); err != nil {
		log.Error().Err(err).Msg("failed to parse collection JSON")
		return c.Status(fiber.StatusBadRequest).JSON(Message{
			Code:        ParameterError,
			Description: "invalid JSON for collection",
		})
	}

	// Make sure the collection has an ID.
	if coll.Id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(Message{
			Code:        ParameterError,
			Description: "collection id is required",
		})
	}

	// Call the appropriate service method.
	var result *stac.Collection
	var err error
	if c.Method() == fiber.MethodPut {
		log.Info().Msg("updating collection")
		result, err = stacService.UpdateCollection(ctx, &coll)
	} else {
		result, err = stacService.CreateCollection(ctx, &coll)
	}
	if err != nil {
		log.Error().Err(err).Msg("failed to modify collection")
		return c.Status(fiber.StatusBadRequest).JSON(Message{
			Code:        "ModifyCollectionFailed",
			Description: err.Error(),
		})
	}

	// Enrich the returned collection with extra links.
	enriched, err := enrichCollection(c, result)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Message{
			Code:        "LinkEnrichmentFailed",
			Description: err.Error(),
		})
	}
	return c.JSON(enriched)
}

// DeleteCollection deletes a collection.
// Endpoint: DELETE /collections/:collectionId
func DeleteCollection(c *fiber.Ctx) error {
	ctx := c.Context()
	collectionID := c.Params("collectionId")

	if err := stacService.DeleteCollection(ctx, collectionID); err != nil {
		log.Error().Err(err).Str("id", collectionID).Msg("failed to delete collection")
		return c.Status(fiber.StatusNotFound).JSON(Message{
			Code:        NotFoundError,
			Description: "collection not found",
		})
	}

	return c.JSON(Message{
		Code:        "CollectionDeleted",
		Description: "the collection was successfully deleted",
	})
}

// Collection retrieves a single collection by its ID.
// Endpoint: GET /collections/:collectionId
func Collection(c *fiber.Ctx) error {
	ctx := c.Context()
	collectionID := c.Params("collectionId")

	coll, err := stacService.GetCollection(ctx, collectionID)
	if err != nil {
		log.Error().Err(err).Msg("collection not found")
		return c.Status(fiber.StatusNotFound).JSON(Message{
			Code:        NotFoundError,
			Description: fmt.Sprintf("collection '%s' not found", collectionID),
		})
	}

	enriched, err := enrichCollection(c, coll)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Message{
			Code:        "LinkEnrichmentFailed",
			Description: err.Error(),
		})
	}
	return c.JSON(enriched)
}

// Collections returns a list of all collections managed by this STAC server.
// Endpoint: GET /collections
func Collections(c *fiber.Ctx) error {
	ctx := c.Context()

	collList, err := stacService.ListCollections(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list collections")
		return c.Status(fiber.StatusInternalServerError).JSON(Message{
			Code:        "CollectionsListError",
			Description: err.Error(),
		})
	}

	// Optionally, enrich each collection with additional links.
	enrichedList := make([]*stac.Collection, 0, len(collList))
	for _, coll := range collList {
		enriched, err := enrichCollection(c, coll)
		if err != nil {
			// If link enrichment fails, log the error and use the collection as is.
			log.Error().Err(err).Str("collectionId", coll.Id).Msg("failed to enrich collection links")
			enrichedList = append(enrichedList, coll)
		} else {
			enrichedList = append(enrichedList, enriched)
		}
	}

	// Build overall response links.
	baseURL := getBaseURL(c)
	overallLinks := []stac.Link{
		{Rel: "self", Href: fmt.Sprintf("%s/collections", baseURL), Type: "application/json"},
		{Rel: "root", Href: fmt.Sprintf("%s/", baseURL), Type: "application/json"},
		{Rel: "parent", Href: fmt.Sprintf("%s/", baseURL), Type: "application/json"},
	}

	return c.JSON(struct {
		Collections []*stac.Collection `json:"collections"`
		Links       []stac.Link        `json:"links"`
	}{
		Collections: enrichedList,
		Links:       overallLinks,
	})
}

// enrichCollection is a helper function that adds standard links to a collection.
func enrichCollection(c *fiber.Ctx, coll *stac.Collection) (*stac.Collection, error) {
	baseURL := getBaseURL(c)
	// Add links such as self, root, parent, and items.
	coll.Links = AddLink(coll.Links, baseURL, "self", fmt.Sprintf("/collections/%s", coll.Id), "application/json")
	coll.Links = AddLink(coll.Links, baseURL, "root", "/", "application/json")
	coll.Links = AddLink(coll.Links, baseURL, "parent", "/", "application/json")
	coll.Links = AddLink(coll.Links, baseURL, "items", fmt.Sprintf("/collections/%s/items", coll.Id), "application/geo+json")
	// Ensure the type is set.
	return coll, nil
}
