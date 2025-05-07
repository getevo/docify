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
	"github.com/shopspring/decimal"
	"gorm.io/gorm/schema"
	"math/rand"
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
		if err != nil{
			log.Error(err)
			def = &serializer.StructDefinition{}
		}
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
		fmt.Println("Parsing fields for entity:", entity.Name)
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
		fmt.Println("fields parsed for entity:", entity.Name)
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
	fmt.Println("Faking data for entity:", entity.Name)
	var sample = serializer.DataSample{}
	sample.CreateJSON = "{"
	sample.UpdateJSON = "{"
	// try get a data from database
	var object = reflect.Indirect(reflect.New(entity.Resource.Type))
	ptr := object.Addr().Interface()
	if db.First(ptr).RowsAffected == 0 {
		fmt.Println("Database doesn't contain any data for entity:", entity.Name)
		fmt.Println("Faking data using faker...")

		//_ = gofakeit.Struct(ptr)

		for _, item := range entity.Fields {
			var field = object.FieldByName(item.Name)
			if len(item.Enum) > 0 {
				field.SetString(item.Enum[0])
				continue
			}
			if item.GoType == "decimal.Decimal" {
				field.Set(reflect.ValueOf(decimal.NewFromFloat(3.14)))
				continue
			}
			var baseField = field
			for field.Kind() == reflect.Ptr {
				field = field.Elem()
			}

			switch field.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				setFieldValue(baseField, rand.Intn(1000))
			case reflect.Float64:
				setFieldValue(baseField, rand.Float64())
			case reflect.Float32:
				setFieldValue(baseField, rand.Float32())
			case reflect.String:
				setFieldValue(baseField, text.Random(5))
			case reflect.Bool:
				setFieldValue(baseField, rand.Intn(2) == 0)
			default:

			}
			fmt.Println("Faked data.")
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

// Helper function to set value, handling pointers recursively
func setFieldValue(field reflect.Value, v interface{}) {
	var value = reflect.ValueOf(v)
	for field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}
	if field.CanSet() {
		if value.Type().ConvertibleTo(field.Type()) {
			field.Set(value.Convert(field.Type()))
		}
	}
}
