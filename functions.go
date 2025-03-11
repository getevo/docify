package docify

import (
	"fmt"
	"github.com/getevo/docify/serializer"

	scm "github.com/getevo/evo/v2/lib/db/schema"
	"github.com/getevo/evo/v2/lib/gpath"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/restify"
	"gorm.io/gorm/schema"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var doc = serializer.Doc{}

func SerializeEntities() {

	if gpath.IsFileExist("project.yml") {
		doc.ParseYaml("project.yml")
	}
	var resources []*restify.Resource
	for idx, _ := range restify.Resources {
		resources = append(resources, restify.Resources[idx])
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Table < resources[j].Table
	})
	var m = map[string]*serializer.Entity{}
	for _, resource := range resources {
		def, err := GetStructDefinition(resource.Type)
		var chunks = strings.Split(resource.Name, ".")
		var entity = serializer.Entity{
			ID:          resource.Name,
			Name:        chunks[1],
			Description: def.Description,
			Pkg:         chunks[0],
			Path:        resource.Type.PkgPath(),
		}
		var fields []serializer.Field

		entity.Definition = def
		if err != nil {
			log.Error(err)
		}
		for _, field := range resource.Schema.Fields {
			if field.DBName == "" {
				var f = field.FieldType
				for f.Kind() == reflect.Ptr {
					f = f.Elem()
				}
				switch f.Kind() {
				case reflect.Struct:

					fmt.Println(f)
					entity.Association = append(entity.Association, serializer.Association{
						Name:       field.Name,
						EntityName: f.String(),
						Array:      false,
					})
				case reflect.Slice:
					entity.Association = append(entity.Association, serializer.Association{
						Name:  field.Name,
						Array: true,
					})
				default:

				}
				continue
			}

			var fieldDoc = serializer.Field{
				Name:          field.Name,
				GoType:        field.FieldType.Name(),
				AutoIncrement: field.AutoIncrement,
				PrimaryKey:    field.PrimaryKey,
				Unique:        field.Unique,
				Nullable:      field.NotNull,
			}

			fieldDoc.JsonTag = strings.Split(field.Tag.Get("json"), ",")[0]
			if fieldDoc.JsonTag == "" {
				fieldDoc.JsonTag = field.Name
			}
			fieldDoc.JsonType = getJsonType(field)
			if v, ok := field.TagSettings["TYPE"]; ok && strings.HasPrefix(v, "enum") {
				fieldDoc.Enum = ExtractEnumValues(v)
				fieldDoc.JsonType = "string" // Enum fields are treated as strings for now.
			}
			fieldDoc.Default = field.DefaultValue
			fieldDoc.Description = field.Comment
			fieldDoc.Validation = field.Tag.Get("validation")

			entity.Endpoints = restify.Resources[resource.Table].Actions

			if v, ok := field.TagSettings["FK"]; ok {
				chunks = strings.Split(v, ".")
				var s = scm.Find(chunks[0])
				if s != nil {
					var fk = &serializer.ForeignKey{
						Table: chunks[0],
					}
					if len(chunks) > 1 {
						fk.Field = chunks[1]
					} else {
						if len(s.PrimaryKey) > 0 {
							fk.Field = s.PrimaryKey[0]
						}
					}
					fieldDoc.ForeignKey = fk
				}
			}
			fields = append(fields, fieldDoc)
		}
		entity.Fields = fields
		doc.Entities = append(doc.Entities, entity)
		m[entity.ID] = &entity
	}

	for idx, _ := range doc.Entities {
		for i, _ := range doc.Entities[idx].Association {
			if doc.Entities[idx].Association[i].EntityName != "" {
				doc.Entities[idx].Association[i].Entity = m[doc.Entities[idx].Association[i].EntityName]
			}
		}
	}

}

func getJsonType(field *schema.Field) string {
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

func ExtractEnumValues(input string) []string {
	// Define regex pattern to capture values inside the enum declaration
	re := regexp.MustCompile(`enum\((.*?)\)`)
	matches := re.FindStringSubmatch(input)

	if len(matches) < 2 {
		return nil
	}

	// Remove single quotes and split by comma
	values := strings.Split(matches[1], ",")
	for i := range values {
		values[i] = strings.Trim(values[i], "'")
	}

	return values
}
