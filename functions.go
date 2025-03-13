package docify

import (
	"encoding/json"
	"fmt"
	"github.com/getevo/docify/serializer"
	"github.com/getevo/evo/v2/lib/db"
	scm "github.com/getevo/evo/v2/lib/db/schema"
	"github.com/getevo/evo/v2/lib/gpath"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/text"
	"github.com/getevo/restify"
	"github.com/go-faker/faker/v4"
	"github.com/shopspring/decimal"
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
	for i, resource := range resources {
		def, err := GetStructDefinition(resource.Type)
		var chunks = strings.Split(resource.Name, ".")
		var entity = serializer.Entity{
			ID:          resource.Name,
			Name:        chunks[1],
			Description: def.Description,
			Pkg:         chunks[0],
			Path:        resource.Type.PkgPath(),
			Resource:    resources[i],
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
				GoType:        field.FieldType.String(),
				DBName:        field.DBName,
				AutoIncrement: field.AutoIncrement,
				PrimaryKey:    field.PrimaryKey,
				Unique:        field.Unique,
				Nullable:      field.NotNull || field.FieldType.Kind() == reflect.Ptr,
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
		entity.DataSample = ModelDataFaker(&entity)
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

func ModelDataFaker(entity *serializer.Entity) serializer.DataSample {
	var sample = serializer.DataSample{}
	sample.CreateJSON = "{"
	sample.UpdateJSON = "{"
	// try get a data from database
	var object = reflect.Indirect(reflect.New(entity.Resource.Type))
	ptr := object.Addr().Interface()
	if db.First(ptr).RowsAffected == 0 {
		_ = faker.FakeData(ptr)
		for _, item := range entity.Fields {
			if len(item.Enum) > 0 {
				object.FieldByName(item.Name).SetString(item.Enum[0])
			}
			if item.GoType == "decimal.Decimal" {
				object.FieldByName(item.Name).Set(reflect.ValueOf(decimal.NewFromFloat(3.14)))
			}
		}
	}

	var comment = ""
	for _, item := range entity.Fields {
		if item.AutoIncrement {
			continue
		}
		if item.DBName == "created_at" || item.DBName == "updated_at" || item.DBName == "deleted_at" {
			continue
		}
		var v, _ = json.Marshal(object.FieldByName(item.Name).Interface())
		var description = []string{
			item.GoType,
		}
		if len(item.Enum) > 0 {
			description = append(description, "enum: "+strings.Join(item.Enum, ", "))
		}
		if item.Description != "" {
			description = append(description, item.Description)
		}
		if item.Nullable {
			description = append(description, "optional")
		}
		if item.Unique {
			description = append(description, "unique")
		}
		if item.Validation != "" {
			description = append(description, "validation: "+item.Validation)
		}
		if item.PrimaryKey {
			description = append(description, "pk")
		}
		if item.AutoIncrement {
			description = append(description, "autoIncr.")
		}

		var row = fmt.Sprintf(comment+"\n\t\"%s\":%s,", item.JsonTag, string(v))
		comment = " // " + strings.Join(description, ",")
		sample.CreateJSON += row
		if !item.PrimaryKey {
			sample.UpdateJSON += row
		}
	}

	sample.CreateJSON = strings.Trim(sample.CreateJSON, ",") + comment + "\n}"
	sample.UpdateJSON += strings.Trim(sample.UpdateJSON, ",") + comment + "\n}"
	sample.BatchJSON = "[\n" + shift(sample.CreateJSON) + "\n]"
	sample.SingleResponseJSON = text.ToJSON(object.Interface())
	sample.MultipleResponseJSON = "[\n" + shift(sample.SingleResponseJSON) + "\n]"
	return sample
}

func shift(s string) string {
	return "\t" + strings.Join(strings.Split(s, "\n"), "\n\t")
}
