package handler

import (
	"fmt"

	"github.com/planetlabs/go-stac"

	json "github.com/goccy/go-json"
)

type Message struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

var JSONParsingError = "JSONParsingError"
var NotFoundError = "NotFoundError"
var DatabaseError = "DatabaseError"
var ParameterError = "ParameterError"
var ServerError = "ServerError"

type GeoJSON struct {
	Type        string           `json:"type"`
	Coordinates *json.RawMessage `json:"coordinates"`
}

// AddLink creates a new link reference in the Links array of a Feature
// rel is the name of the link relationship
// baseUrl baseUrl of this STAC server
// endpoint is the last portion of the URL i.e. <base url>/api/stac/v1/<endpoint>
func AddLink(links []*stac.Link, baseURL string, rel string, endpoint string, mimeType string) []*stac.Link {
	href := fmt.Sprintf("%s/api/stac/v1%s", baseURL, endpoint)
	links = append(links, &stac.Link{
		Rel:  rel,
		Type: mimeType,
		Href: href,
	})

	return links
}

// AddLinkPost creates a new link reference in the Links array of a Feature
// rel is the name of the link relationship
// baseUrl baseUrl of this STAC server
// endpoint is the last portion of the URL i.e. <base url>/api/stac/v1/<endpoint>
func AddLinkPost(links []*stac.Link, baseURL string, rel string, endpoint string, mimeType string) []*stac.Link {
	href := fmt.Sprintf("%s/api/stac/v1%s", baseURL, endpoint)
	links = append(links, &stac.Link{
		Rel:  rel,
		Type: mimeType,
		Href: href,
	})

	return links
}
