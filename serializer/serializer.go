package serializer

import (
	"github.com/getevo/restify"
	"gopkg.in/yaml.v3"
	"os"
)

type Doc struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Entities    []Entity `json:"entities"`
}

func (d *Doc) ParseYaml(s string) error {
	data, err := os.ReadFile(s)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, d)
	return err
}

type Entity struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Pkg         string              `json:"pkg"`
	Path        string              `json:"path"`
	Fields      []Field             `json:"fields"`
	Association []Association       `json:"associations"`
	Endpoints   []*restify.Endpoint `json:"endpoints"`
	PrimaryKey  []Field             `json:"primary_key"`
	Definition  *StructDefinition   `json:"definition"`
	Resource    *restify.Resource   `json:"resource"`
	DataSample  DataSample          `json:"data_sample"`
}

type Field struct {
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	JsonTag       string      `json:"json_tag"`
	JsonType      string      `json:"json_type"`
	DBType        string      `json:"db_type"`
	GoType        string      `json:"go_type"`
	DBName        string      `json:"db_name"`
	Validation    string      `json:"validation"`
	PrimaryKey    bool        `json:"primary_key"`
	AutoIncrement bool        `json:"auto_increment"`
	Nullable      bool        `json:"nullable"`
	Unique        bool        `json:"unique"`
	UniqueIndex   string      `json:"unique_index"`
	Default       string      `json:"default"`
	Enum          []string    `json:"enum"`
	Indexed       bool        `json:"indexed"`
	Index         string      `json:"index"`
	ForeignKey    *ForeignKey `json:"foreign_key"`
	SampleData    interface{} `json:"sample_data"`
}

type Association struct {
	Name       string  `json:"name"`
	EntityName string  `json:"entity_name"`
	Entity     *Entity `json:"entity"`
	Array      bool    `json:"array"`
}

type ForeignKey struct {
	Table  string  `json:"table"`
	Field  string  `json:"field"`
	Entity *Entity `json:"entity"`
}

type StructDefinition struct {
	File        string
	Description string
	Body        string
	Fields      []FieldDefinition
}

type FieldDefinition struct {
	Name        string
	Type        string
	Tag         string
	Description string
}

type DataSample struct {
	CreateJSON           string `json:"create_json"`
	UpdateJSON           string `json:"update_json"`
	BatchJSON            string `json:"batch_json"`
	SingleResponseJSON   string `json:"single_response_json"`
	MultipleResponseJSON string `json:"multiple_response_json"`
}
