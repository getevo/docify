package openapi

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"strings"
)

// OpenAPI represents the root structure of an OpenAPI document.
type OpenAPI struct {
	OpenAPI      string         `yaml:"openapi"`
	Info         Info           `yaml:"info"`
	ExternalDocs ExternalDocs   `yaml:"externalDocs,omitempty"`
	Servers      []Server       `yaml:"servers,omitempty"`
	Tags         []Tag          `yaml:"tags,omitempty"`
	Paths        Paths          `yaml:"paths"` // Custom type to preserve order
	Components   Components     `yaml:"components,omitempty"`
	Security     []SecurityItem `yaml:"security,omitempty"`
}

func (o *OpenAPI) GenerateYaml() ([]byte, error) {
	return yaml.Marshal(o)
}

// --------------------------
//         METADATA
// --------------------------

// Info contains metadata about the API.
type Info struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

// ExternalDocs provides links to external API documentation.
type ExternalDocs struct {
	Description string `yaml:"description"`
	URL         string `yaml:"url"`
}

// Server represents an API server.
type Server struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description"`
}

// Tag categorizes API endpoints.
type Tag struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// --------------------------
//         PATHS
// --------------------------

// Paths is our custom type that stores PathItem data in a slice
// but marshals into a YAML map (required by OpenAPI).
type Paths struct {
	Items []*PathItem
}

// AddPath lets us append a new PathItem (preserving order).
func (p *Paths) AddPath(pathItem *PathItem) {
	p.Items = append(p.Items, pathItem)
}

// MarshalYAML emits a YAML map keyed by path string:
//
// paths:
//
//	/some-path:
//	  get: ...
//	  post: ...
func (p Paths) MarshalYAML() (interface{}, error) {
	root := yaml.Node{
		Kind: yaml.MappingNode, // root "paths" must be a map
	}

	for _, pi := range p.Items {
		// The path itself is the map key:
		keyNode := yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: pi.Path,
		}

		// The value is another map, with methods (get, post, etc.) as sub-keys.
		valueNode := yaml.Node{
			Kind: yaml.MappingNode,
		}

		// Convert each operation into a methodName -> operationObject
		for _, op := range pi.Operations {
			// Typically "GET" / "POST" / "PATCH", etc. → "get" / "post" / "patch"
			methodName := strings.ToLower(op.Method)

			methodKeyNode := yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: methodName,
			}

			methodValueNode, err := marshalOperation(op)
			if err != nil {
				return nil, err
			}

			valueNode.Content = append(valueNode.Content, &methodKeyNode, methodValueNode)
		}

		// Add "path: { ...methods... }" to the root
		root.Content = append(root.Content, &keyNode, &valueNode)
	}

	return &root, nil
}

// PathItem holds a path string and an ordered list of operations.
type PathItem struct {
	Path       string        // Not marshaled directly; used as map key
	Operations []APIEndpoint `yaml:"operations"`
}

// --------------------------
//         OPERATIONS
// --------------------------

// APIEndpoint defines an API operation (method, summary, tags, etc.).
type APIEndpoint struct {
	Method      string         `yaml:"method"` // e.g., "GET", "POST"
	Summary     string         `yaml:"summary"`
	Description string         `yaml:"description"`
	Tags        []string       `yaml:"tags,omitempty"`
	Parameters  []Parameter    `yaml:"parameters,omitempty"`
	RequestBody *RequestBody   `yaml:"requestBody,omitempty"`
	Responses   []Response     `yaml:"responses,omitempty"`
	Security    []SecurityItem `yaml:"security,omitempty"`
	Deprecated  bool           `yaml:"deprecated,omitempty"`
}

// marshalOperation converts an APIEndpoint into a YAML sub-map node
// containing fields like summary, description, tags, etc.
func marshalOperation(op APIEndpoint) (*yaml.Node, error) {
	// We'll build this as a map. Each field is a key-value pair in the mapping node.
	opNode := yaml.Node{
		Kind: yaml.MappingNode,
	}

	// Add "summary: ..."
	opNode.Content = append(opNode.Content,
		&yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "summary",
		},
		&yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: op.Summary,
		},
	)

	// Add "description: ..."
	opNode.Content = append(opNode.Content,
		&yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "description",
		},
		&yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: op.Description,
		},
	)

	// Add "tags: [...]" if present
	if len(op.Tags) > 0 {
		tagsNode := yaml.Node{
			Kind: yaml.SequenceNode,
		}
		for _, t := range op.Tags {
			tagsNode.Content = append(tagsNode.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: t,
			})
		}
		opNode.Content = append(opNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "tags",
			},
			&tagsNode,
		)
	}

	// Add "parameters" if present
	if len(op.Parameters) > 0 {
		paramsNode, err := marshalParameters(op.Parameters)
		if err != nil {
			return nil, err
		}
		opNode.Content = append(opNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "parameters",
			},
			paramsNode,
		)
	}

	// Add "requestBody" if present
	if op.RequestBody != nil {
		rbNode, err := marshalRequestBody(op.RequestBody)
		if err != nil {
			return nil, err
		}
		opNode.Content = append(opNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "requestBody",
			},
			rbNode,
		)
	}

	// Add "responses" if present
	if len(op.Responses) > 0 {
		respNode, err := marshalResponses(op.Responses)
		if err != nil {
			return nil, err
		}
		opNode.Content = append(opNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "responses",
			},
			respNode,
		)
	}

	// Add "security" if present
	if len(op.Security) > 0 {
		secNode, err := marshalSecurity(op.Security)
		if err != nil {
			return nil, err
		}
		opNode.Content = append(opNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "security",
			},
			secNode,
		)
	}

	// Add "deprecated: true" if needed
	if op.Deprecated {
		opNode.Content = append(opNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "deprecated",
			},
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "true",
			},
		)
	}

	return &opNode, nil
}

type Parameter struct {
	Name        string  `yaml:"name"`
	In          string  `yaml:"in"`
	Required    bool    `yaml:"required"`
	Description string  `yaml:"description"`
	Schema      *Schema `yaml:"schema"`
}

func marshalParameters(params []Parameter) (*yaml.Node, error) {
	// "parameters" is typically a YAML sequence of parameter objects
	paramsNode := yaml.Node{
		Kind: yaml.SequenceNode,
	}

	for _, p := range params {
		paramNode := yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				// name
				{Kind: yaml.ScalarNode, Value: "name"},
				{Kind: yaml.ScalarNode, Value: p.Name},
				// in
				{Kind: yaml.ScalarNode, Value: "in"},
				{Kind: yaml.ScalarNode, Value: p.In},
				// required
				{Kind: yaml.ScalarNode, Value: "required"},
				{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", p.Required)},
				// description
				{Kind: yaml.ScalarNode, Value: "description"},
				{Kind: yaml.ScalarNode, Value: p.Description},
				// schema
				{Kind: yaml.ScalarNode, Value: "schema"},
			},
		}

		schemaNode, err := marshalSchema(p.Schema)
		if err != nil {
			return nil, err
		}
		paramNode.Content = append(paramNode.Content, schemaNode)

		paramsNode.Content = append(paramsNode.Content, &paramNode)
	}

	return &paramsNode, nil
}

type RequestBody struct {
	Description string               `yaml:"description"`
	Content     []RequestContentType `yaml:"content"`
}

type RequestContentType struct {
	ContentType string  `yaml:"contentType"`
	Schema      *Schema `yaml:"schema"`
}

func marshalRequestBody(rb *RequestBody) (*yaml.Node, error) {
	// We'll create a map with "description" + "content"
	rbNode := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			// description
			{Kind: yaml.ScalarNode, Value: "description"},
			{Kind: yaml.ScalarNode, Value: rb.Description},
			// content
			{Kind: yaml.ScalarNode, Value: "content"},
		},
	}

	contentNode, err := marshalRequestContent(rb.Content)
	if err != nil {
		return nil, err
	}
	rbNode.Content = append(rbNode.Content, contentNode)

	return &rbNode, nil
}

func marshalRequestContent(rcs []RequestContentType) (*yaml.Node, error) {
	// "content" is typically a map keyed by content type
	contentMap := yaml.Node{
		Kind: yaml.MappingNode,
	}

	for _, rc := range rcs {
		// key = rc.ContentType
		keyNode := yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: rc.ContentType,
		}

		// value = sub-map { schema: ... }
		valueNode := yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "schema"},
			},
		}
		schemaNode, err := marshalSchema(rc.Schema)
		if err != nil {
			return nil, err
		}
		valueNode.Content = append(valueNode.Content, schemaNode)

		contentMap.Content = append(contentMap.Content, &keyNode, &valueNode)
	}

	return &contentMap, nil
}

type Response struct {
	StatusCode  string                `yaml:"statusCode"` // e.g. "200"
	Description string                `yaml:"description"`
	Content     []ResponseContentType `yaml:"content,omitempty"`
	Headers     []Header              `yaml:"headers,omitempty"`
}

type ResponseContentType struct {
	ContentType string  `yaml:"contentType"`
	Schema      *Schema `yaml:"schema,omitempty"`
}

type Header struct {
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Schema      *Schema `yaml:"schema"`
}

func marshalResponses(resps []Response) (*yaml.Node, error) {
	// In OpenAPI, "responses" must be a map keyed by status code
	root := yaml.Node{
		Kind: yaml.MappingNode,
	}

	for _, r := range resps {
		keyNode := yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: r.StatusCode, // e.g., "200"
		}

		valNode, err := marshalSingleResponse(r)
		if err != nil {
			return nil, err
		}

		root.Content = append(root.Content, &keyNode, valNode)
	}

	return &root, nil
}

func marshalSingleResponse(r Response) (*yaml.Node, error) {
	respNode := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			// description
			{Kind: yaml.ScalarNode, Value: "description"},
			{Kind: yaml.ScalarNode, Value: r.Description},
		},
	}

	// Add content if present
	if len(r.Content) > 0 {
		contentMap := yaml.Node{
			Kind: yaml.MappingNode,
		}
		for _, c := range r.Content {
			ctKey := yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: c.ContentType,
			}
			ctVal := yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "schema"},
				},
			}
			schemaNode, err := marshalSchema(c.Schema)
			if err != nil {
				return nil, err
			}
			ctVal.Content = append(ctVal.Content, schemaNode)
			contentMap.Content = append(contentMap.Content, &ctKey, &ctVal)
		}

		respNode.Content = append(respNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "content",
			},
			&contentMap,
		)
	}

	// Add headers if present
	if len(r.Headers) > 0 {
		headersMap := yaml.Node{
			Kind: yaml.MappingNode,
		}
		for _, h := range r.Headers {
			hKey := yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: h.Name,
			}
			hVal := yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "description"},
					{Kind: yaml.ScalarNode, Value: h.Description},
					{Kind: yaml.ScalarNode, Value: "schema"},
				},
			}
			schemaNode, err := marshalSchema(h.Schema)
			if err != nil {
				return nil, err
			}
			hVal.Content = append(hVal.Content, schemaNode)
			headersMap.Content = append(headersMap.Content, &hKey, &hVal)
		}

		respNode.Content = append(respNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "headers",
			},
			&headersMap,
		)
	}

	return &respNode, nil
}

type Schema struct {
	Type        string           `yaml:"type,omitempty"`
	Properties  []SchemaProperty `yaml:"properties,omitempty"`
	Items       *Schema          `yaml:"items,omitempty"`
	Required    []string         `yaml:"required,omitempty"`
	Description string           `yaml:"description,omitempty"`
}

type SchemaProperty struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

func marshalSchema(s *Schema) (*yaml.Node, error) {
	// We'll create a map with "type", "description", "properties", etc.
	schemaNode := yaml.Node{
		Kind: yaml.MappingNode,
	}
	if s == nil {
		return &schemaNode, nil
	}
	if s.Type != "" {
		schemaNode.Content = append(schemaNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "type"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: s.Type},
		)
	}

	if s.Description != "" {
		schemaNode.Content = append(schemaNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "description"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: s.Description},
		)
	}

	if len(s.Properties) > 0 {
		propsNode := yaml.Node{
			Kind: yaml.MappingNode,
		}
		for _, sp := range s.Properties {
			propKeyNode := yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: sp.Name,
			}
			propValNode := yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "type"},
					{Kind: yaml.ScalarNode, Value: sp.Type},
				},
			}
			if sp.Description != "" {
				propValNode.Content = append(propValNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "description"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: sp.Description},
				)
			}
			propsNode.Content = append(propsNode.Content, &propKeyNode, &propValNode)
		}

		schemaNode.Content = append(schemaNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "properties"},
			&propsNode,
		)
	}

	if s.Items != nil {
		schemaNode.Content = append(schemaNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "items"},
		)
		child, err := marshalSchema(s.Items)
		if err != nil {
			return nil, err
		}
		schemaNode.Content = append(schemaNode.Content, child)
	}

	if len(s.Required) > 0 {
		reqNode := yaml.Node{
			Kind: yaml.SequenceNode,
		}
		for _, r := range s.Required {
			reqNode.Content = append(reqNode.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: r,
			})
		}

		schemaNode.Content = append(schemaNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "required"},
			&reqNode,
		)
	}

	return &schemaNode, nil
}

type Components struct {
	SecuritySchemes []SecurityScheme `yaml:"securitySchemes,omitempty"`
	Schemas         []SchemaItem     `yaml:"schemas,omitempty"`
}

type SecurityScheme struct {
	Name         string `yaml:"name"`
	Type         string `yaml:"type"`
	Description  string `yaml:"description,omitempty"`
	In           string `yaml:"in,omitempty"`
	Scheme       string `yaml:"scheme,omitempty"`
	BearerFormat string `yaml:"bearerFormat,omitempty"`
}

type SchemaItem struct {
	Name   string `yaml:"name"`
	Schema Schema `yaml:"schema"`
}

type SecurityItem struct {
	Name   string   `yaml:"name"`
	Scopes []string `yaml:"scopes,omitempty"`
}

func marshalSecurity(sec []SecurityItem) (*yaml.Node, error) {
	// "security" is a list of objects:
	// security:
	//   - ApiKeyAuth: []
	//   - ...
	seqNode := yaml.Node{
		Kind: yaml.SequenceNode,
	}

	for _, s := range sec {
		// Typically "Name: [scopes]" in a map
		itemNode := yaml.Node{
			Kind: yaml.MappingNode,
		}

		// key = s.Name, value = scopes array
		keyNode := yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: s.Name,
		}

		scopesSeq := yaml.Node{
			Kind: yaml.SequenceNode,
		}
		for _, sc := range s.Scopes {
			scopesSeq.Content = append(scopesSeq.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: sc,
			})
		}

		itemNode.Content = append(itemNode.Content, &keyNode, &scopesSeq)
		seqNode.Content = append(seqNode.Content, &itemNode)
	}

	return &seqNode, nil
}
