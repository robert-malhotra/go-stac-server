// Copyright 2021-2023
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"fmt"
	"strings"

	"github.com/go-geospatial/go-stac-server/common"
	"github.com/go-geospatial/go-stac-server/service"
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/planetlabs/go-stac"
	"github.com/rs/zerolog/log"
)

// Item returns details of a specific item
// GET /search
// POST /search
func Search(c *fiber.Ctx) error {
	baseURL := getBaseURL(c)
	token := c.Query("token", "")

	var cql service.CQL
	var err error
	switch c.Method() {
	case "GET":
		cql, err = getCQLFromQuery(c)
		if err != nil {
			// note http response and logging handled by getCQLFromQuery
			return nil
		}
	case "POST":
		cql, err = getCQLFromBody(c)
		if err != nil {
			// note http response and logging handled by getCQLFromBody
			return nil
		}
	default:
		c.Status(fiber.StatusBadRequest)
		_ = c.JSON(Message{
			Code:        ParameterError,
			Description: "unsupported method",
		})
	}

	if token != "" {
		cql.Token = token
	}

	// do the search
	fc, err := service.Search(cql)
	if err != nil {
		log.Error().Err(err).Msg("stac search returned an error")
		c.Status(fiber.StatusBadRequest)
		return c.JSON(Message{
			Code:        ParameterError,
			Description: err.Error(),
		})
	}

	// enrich links
	for _, item := range fc.Features {
		var links []*stac.Link

		for idx, link := range links {
			if link.Rel == "collection" {
				link.Href = fmt.Sprintf("%s/api/stac/v1/collections/%s", baseURL, item.Collection)
			}
			links[idx] = link
		}

		links = AddLink(links, baseURL, "parent", fmt.Sprintf("/collections/%s", item.Collection), "application/json")
		links = AddLink(links, baseURL, "root", "/", "application/json")
		links = AddLink(links, baseURL, "self", fmt.Sprintf("/collections/%s/items/%s", item.Collection, item.Id), "application/geo+json")

		item.Links = links
	}

	// overall links
	overallLinks := make([]*stac.Link, 0, 5)
	overallLinks = AddLink(overallLinks, baseURL, "parent", "/", "application/json")
	overallLinks = AddLink(overallLinks, baseURL, "root", "/", "application/json")

	switch c.Method() {
	case "GET":
		queryParts := buildQueryArray(c)
		token := c.Query("token", "")
		var queryPartsFull []string
		if token != "" {
			queryPartsFull = queryParts
			queryPartsFull = append(queryPartsFull, fmt.Sprintf("token=%s", token))
		}
		query := strings.Join(queryPartsFull, "&")
		overallLinks = AddLink(overallLinks, baseURL, "self", fmt.Sprintf("/search?%s", query), "application/geo+json")

		if fc.Next != "" {
			queryPartsFull = queryParts
			queryPartsFull = append(queryPartsFull, fmt.Sprintf("token=%s", fc.Next))
			query := strings.Join(queryPartsFull, "&")
			overallLinks = AddLink(overallLinks, baseURL, "next", fmt.Sprintf("/search?%s", query), "application/geo+json")
		}
		if fc.Prev != "" {
			queryPartsFull = queryParts
			queryPartsFull = append(queryPartsFull, fmt.Sprintf("token=%s", fc.Prev))
			query := strings.Join(queryPartsFull, "&")
			overallLinks = AddLink(overallLinks, baseURL, "previous", fmt.Sprintf("/search?%s", query), "application/geo+json")
		}
	case "POST":
		overallLinks = AddLinkPost(overallLinks, baseURL, "self", "/search", "application/geo+json")

		if fc.Next != "" {
			cql.Token = fc.Next
			overallLinks = AddLinkPost(overallLinks, baseURL, "next", "/search", "application/geo+json")
		}
		if fc.Prev != "" {
			cql.Token = fc.Prev
			overallLinks = AddLinkPost(overallLinks, baseURL, "previous", "/search", "application/geo+json")
		}
	default:
		c.Status(fiber.StatusBadRequest)
		_ = c.JSON(Message{
			Code:        ParameterError,
			Description: "unsupported method",
		})
	}

	return common.GeoJSON(c, struct {
		Type     string           `json:"type"`
		Context  *json.RawMessage `json:"context"`
		Features []stac.Item      `json:"features"`
		Links    []*stac.Link     `json:"links"`
	}{
		Type:     "FeatureCollection",
		Context:  fc.Context,
		Features: fc.Features,
		Links:    overallLinks,
	})
}
