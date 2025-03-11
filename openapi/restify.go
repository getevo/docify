package openapi

import (
	"fmt"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/restify"
	"gorm.io/gorm/schema"
	"reflect"
	"sort"
	"strings"
)

var stdResponse = []Response{
	{
		StatusCode:  "400",
		Description: "Data validation error",
	},
	{
		StatusCode:  "403",
		Description: "Authentication required",
	},
	{
		StatusCode:  "5XX",
		Description: "Server error",
	},
}

func (o *OpenAPI) ParseRestify() {

	var resources []*restify.Resource

	for idx, _ := range restify.Resources {
		resources = append(resources, restify.Resources[idx])
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Table < resources[j].Table
	})

	for _, resource := range resources {
		var chunks = strings.Split(resource.Name, ".")
		var description = "App:" + chunks[0] + " Entity:" + chunks[1]
		o.Tags = append(o.Tags, Tag{
			Name:        resource.Name,
			Description: description,
		})
		var body, err = GetRequestBody(resource)
		if err != nil {
			log.Fatal(err)
		}
		var paths = map[string]*PathItem{}
		for _, action := range resource.Actions {
			var pathItem *PathItem
			var ok bool
			if pathItem, ok = paths[action.AbsoluteURI]; !ok {
				pathItem = &PathItem{
					Path: action.AbsoluteURI,
				}
				paths[action.AbsoluteURI] = pathItem
			}

			var responseProperties []SchemaProperty

			for _, field := range resource.Schema.Fields {
				var jsonField = strings.Split(field.Tag.Get("json"), ",")[0]
				if jsonField == "-" {
					continue
				}
				if jsonField == "" {
					jsonField = field.Name
				}
				responseProperties = append(responseProperties, SchemaProperty{
					Name:        jsonField,
					Description: field.Name,
					Type:        getType(field),
				})
			}

			var responses = []Response{
				{
					StatusCode:  "200",
					Description: "OK",
					Content: []ResponseContentType{
						{
							ContentType: "application/json",
							Schema: &Schema{
								Type:       "object",
								Properties: responseProperties,
							},
						},
					},
				},
			}
			var api = APIEndpoint{
				Method:      string(action.Method),
				Summary:     action.Description,
				Description: action.Description,
				Tags:        []string{resource.Name},
				Responses:   append(responses, stdResponse...),
			}

			if action.AcceptData {
				api.RequestBody = body
			}
			pathItem.Operations = append(pathItem.Operations, api)

		}
		var orderedPaths = orderedKeys(paths)
		for idx, _ := range orderedPaths {
			o.Paths.AddPath(orderedPaths[idx])
		}

	}
}

// getOpenAPIType is a helper that returns a naive type mapping for common Go/GORM field types.
// You can customize it to better map your decimal, time, etc.
func getType(field *schema.Field) string {
	goType := field.DataType

	switch goType {
	case schema.String:
		return "string"
	case schema.Int, schema.Uint:
		return "integer"
	case schema.Bool:
		return "boolean"
	case schema.Float:
		return "number"
	case schema.Time:
		return "string" // date-time could be used, but we'll keep it simple
	case schema.Bytes:
		return "string" // or "binary"
	}
	return "string"
}

// GetRequestBody builds an OpenAPI RequestBody for the given model,
// including only direct columns (no foreign-key relationships).
func GetRequestBody(resource *restify.Resource) (*RequestBody, error) {

	// Prepare schema properties
	var properties []SchemaProperty

	for _, field := range resource.Schema.Fields {
		if field.AutoIncrement {
			continue // ignore auto-increment fields
		}
		if field.DBName == "" {
			continue // ignore fields without a DBName
		}
		// Build a property
		propType := getType(field)
		isPtr := resource.Ref.FieldByName(field.Name).Kind() == reflect.Ptr
		var description = "<ul>"
		var optional = true
		if field.Comment != "" {
			description += "<li>" + strings.TrimSpace(field.Comment) + "</li>"
		}
		if v, ok := field.TagSettings["FK"]; ok {
			var selfRef = ""
			if v == resource.Table {
				selfRef = "(Self-reference)"
			}
			description = "<li>Foreign Key: <b>" + v + selfRef + "</b>" + "</li>"
			if !isPtr {
				optional = false
			}
		}
		if v, ok := field.Tag.Lookup("validation"); ok {
			description += "<li>Validation: " + v + "</li>"
			if strings.Contains(v, "required") {
				optional = false
			}
		}

		if v, ok := field.TagSettings["TYPE"]; ok {
			description += "<li>Type: " + v + "</li>"
		}
		if field.PrimaryKey {
			description += "<li><b>PrimaryKey</b></li>"
		}
		if field.Unique {
			description += "<li><b>Unique</b></li>"
		}
		if v := field.UniqueIndex; v != "" {
			description += "<li>Unique Index: " + v + "</li>"
		}

		if v, ok := field.TagSettings["INDEX"]; ok {
			description += "<li>Index: " + v + "</li>"
		}
		if optional {
			description += "<li><b>Optional</b></li>"
		} else {
			description += "<li><b>Required</b></li>"
		}

		description += "</ul>"
		prop := SchemaProperty{
			Name: field.DBName, // e.g. "payment_transaction_id"
			Type: propType,     // e.g. "integer"
			// optionally set a description if needed
			Description: description,
		}
		properties = append(properties, prop)
	}

	// Build the schema
	s := Schema{
		Type:       "object",
		Properties: properties,
	}

	// Derive the model name from its type for the description
	modelName := resource.Type.String()
	if resource.Type.Kind() == reflect.Ptr {
		modelName = resource.Type.Elem().String()
	}
	desc := fmt.Sprintf("Request body for %s", modelName)

	// Build the final RequestBody
	requestBody := RequestBody{
		Description: desc,
		Content: []RequestContentType{
			{
				ContentType: "application/x-www-form-urlencoded",
				Schema:      &s,
			},
			{
				ContentType: "application/json",
				Schema:      &s,
			},
		},
	}

	return &requestBody, nil
}

func orderedKeys(paths map[string]*PathItem) []*PathItem {
	// Extract keys from the map
	keys := make([]string, 0, len(paths))
	for k := range paths {
		keys = append(keys, k)
	}

	// Sort keys
	sort.Strings(keys)

	// Build the ordered slice of PathItem
	ordered := make([]*PathItem, 0, len(keys))
	for _, k := range keys {
		ordered = append(ordered, paths[k])
	}

	return ordered
}
