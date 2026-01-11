// Copyright 2025 Rahul De
// SPDX-License-Identifier: MIT

// climate allows the server to influence the CLI behaviour by using OpenAPI's extensions.
// It encourages spec-first practices thereby keeping both users and maintenance manageable.
// It does just enough to handle the spec and nothing more.

package climate

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// Currently supported OpenAPI types
type OpenAPIType string

const (
	String  OpenAPIType = "string"
	Number  OpenAPIType = "number"
	Integer OpenAPIType = "integer"
	Boolean OpenAPIType = "boolean"
)

// Metadata for all parameters
type ParamMeta struct {
	Name string
	Type OpenAPIType
}

// Data passed into each handler
type HandlerData struct {
	Method           string      // the HTTP method
	Path             string      // the path with the path params filled in
	PathParams       []ParamMeta // List of path params
	QueryParams      []ParamMeta // List of query params
	HeaderParams     []ParamMeta // List of header params
	CookieParams     []ParamMeta // List of cookie params
	RequestBodyParam *ParamMeta  // The optional request body
}

type extensions struct {
	hidden  bool
	aliases []string
	group   string
	ignored bool
	name    string
}

func parseExtensions(exts *orderedmap.Map[string, *yaml.Node]) (*extensions, error) {
	ex := extensions{}
	aliases := []string{}

	for ext, val := range exts.FromOldest() {
		var opts any
		if err := val.Decode(&opts); err != nil {
			return nil, err
		}

		switch ext {
		case "x-cli-hidden":
			ex.hidden = opts.(bool)
		case "x-cli-aliases":
			for _, alias := range opts.([]any) {
				aliases = append(aliases, alias.(string))
			}
			ex.aliases = aliases
		case "x-cli-group":
			ex.group = opts.(string)
		case "x-cli-ignored":
			ex.ignored = opts.(bool)
		case "x-cli-name":
			ex.name = opts.(string)
		}
	}

	return &ex, nil
}

// Loads and verifies an OpenAPI spec frpm an array of bytes
func LoadV3(data []byte) (*libopenapi.DocumentModel[v3.Document], error) {
	document, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, err
	}

	model, err := document.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("Cannot create v3 model: %e", err)
	}

	return model, nil
}

// Loads and verifies an OpenAPI spec from a file path
func LoadFileV3(path string) (*libopenapi.DocumentModel[v3.Document], error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return LoadV3(data)
}

func getParamType(param *v3.Parameter, op *v3.Operation) OpenAPIType {
	schema := param.Schema.Schema()
	if schema != nil {
		return OpenAPIType(schema.Type[0])
	}

	slog.Warn("No type set for param, defaulting to string", "param", param.Name, "id", op.OperationId)

	return String
}

func makeRequestBody(op *v3.Operation, handlerData *HandlerData) (name string, desc string, required bool, err error) {
	if body := op.RequestBody; body != nil {
		// TODO: hammock on ways to handle the req bodies. Maybe take in a stdin?
		bExts, err := parseExtensions(body.Extensions)
		if err != nil {
			return "", "", false, err
		}

		paramName := "climate-data"
		if altName := bExts.name; altName != "" {
			paramName = altName
		} else {
			slog.Warn(
				fmt.Sprintf("No name set of requestBody, defaulting to %s", paramName),
				"id",
				op.OperationId,
			)
		}

		// TODO: Handle all the different MIME types and schemas from body.Content
		// maybe assert the shape if mime is json and schema is an object
		// Treats all request body content as a string as of now
		handlerData.RequestBodyParam = &ParamMeta{Name: paramName, Type: String}
		required := body.Required

		return paramName, body.Description, required != nil && *required, nil
	}

	return "", "", false, nil
}
