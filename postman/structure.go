package postman

import "encoding/json"

type Collection struct {
	Info     Info        `json:"info"`
	Item     []*Item     `json:"item"`
	Auth     *Auth       `json:"auth,omitempty"`
	Event    []*Event    `json:"event,omitempty"`
	Variable []*Variable `json:"variable,omitempty"`
}

func NewCollection(name string, description string) *Collection {
	return &Collection{
		Info: Info{
			Name:        name,
			Description: description,
			Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item: []*Item{},
	}
}

type Info struct {
	PostmanID   string `json:"_postman_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
}

func (c *Collection) AppendItem(item Item) {
	c.Item = append(c.Item, &item)
}

func (c *Collection) SetVariable(key, value, description string) {
	c.Variable = append(c.Variable, &Variable{Key: key, Value: value, Description: description})
}

func (c *Collection) CreateFolder(name, description string) *Item {
	var item = Item{
		Name:        name,
		Description: description,
	}
	c.Item = append(c.Item, &item)
	return &item
}

func (c *Collection) AddAPI(item Item) {
	c.Item = append(c.Item, &item)
}

func (c *Collection) ToJson() ([]byte, error) {
	return json.Marshal(c)
}

type Item struct {
	Name                    string                   `json:"name"`
	Description             string                   `json:"description,omitempty"`
	Request                 *Request                 `json:"request,omitempty"`
	Response                []*Response              `json:"response,omitempty"`
	ProtocolProfileBehavior *ProtocolProfileBehavior `json:"protocolProfileBehavior,omitempty"`
	Item                    []*Item                  `json:"item,omitempty"` // Nested items for folders
	Event                   []*Event                 `json:"event,omitempty"`
	Variable                []*Variable              `json:"variable,omitempty"`
}

func (c *Item) AppendItem(item Item) *Item {
	c.Item = append(c.Item, &item)
	return &item
}

func (c *Item) SetVariable(key, value, description string) {
	c.Variable = append(c.Variable, &Variable{Key: key, Value: value, Description: description})
}

func (c *Item) CreateFolder(name, description string) *Item {
	var item = Item{
		Name:        name,
		Description: description,
	}
	c.Item = append(c.Item, &item)
	return &item
}

type Request struct {
	Method      string   `json:"method"`
	Header      []Header `json:"header,omitempty"`
	Body        *Body    `json:"body,omitempty"`
	Url         *Url     `json:"url,omitempty"`
	Auth        *Auth    `json:"auth,omitempty"`
	Description string   `json:"description,omitempty"`
}

type Header struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type Url struct {
	Raw      string     `json:"raw"`
	Protocol string     `json:"protocol,omitempty"`
	Host     []string   `json:"host,omitempty"`
	Path     []string   `json:"path,omitempty"`
	Query    []KeyValue `json:"query,omitempty"`
	Hash     string     `json:"hash,omitempty"`
}

func (u *Url) AddQuery(key, value, description string, disabled bool) {
	u.Query = append(u.Query, KeyValue{Key: key, Value: value, Description: description, Disabled: disabled})
}

type Event struct {
	Listen string `json:"listen"`
	Script Script `json:"script"`
}

type Script struct {
	ID          string   `json:"id,omitempty"`
	Type        string   `json:"type,omitempty"`
	Exec        []string `json:"exec,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
}

type Variable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type ProtocolProfileBehavior struct {
	DisableBodyPruning bool `json:"disableBodyPruning,omitempty"`
}

type Response struct {
	Name            string   `json:"name"`
	OriginalRequest *Request `json:"originalRequest,omitempty"`
	Status          string   `json:"status"`
	Code            int      `json:"code"`
	Header          []Header `json:"header,omitempty"`
	Body            string   `json:"body,omitempty"`
}
