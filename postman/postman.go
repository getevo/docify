package postman

import (
	"fmt"
	"github.com/getevo/docify/serializer"
	"github.com/getevo/evo/v2/lib/gpath"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/restify"
	"reflect"
	"strconv"
	"strings"
)

func Generate(project *serializer.Doc) {
	var collections []Collection

	var collection = NewCollection(project.Title+" Restify", project.Description)

	for _, entity := range project.Entities {
		fmt.Println("Postman Entity: " + entity.Name)
		folder := collection.CreateFolder(entity.Pkg+"."+entity.Name, entity.Name+" API List")

		for _, action := range entity.Endpoints {
			req := Request{
				Url: &Url{
					Raw: "{{ restify_base }}" + action.AbsoluteURI,
					Host: []string{
						"{{ restify_base }}",
					},
					Path: []string{
						action.AbsoluteURI,
					},
				},
				Method:      string(action.Method),
				Description: GenerateDescription(entity, action),
				Body: &Body{
					Mode: BodyModeRaw,
					Raw:  "",
				},
			}

			req.Body.SetLanguage("json")

			if action.AcceptData {

				if action.Batch {
					req.Body.Raw = entity.DataSample.BatchJSON
				} else {
					if action.Method == "PUT" {
						req.Body.Raw = entity.DataSample.CreateJSON
					} else {
						req.Body.Raw = entity.DataSample.UpdateJSON
					}

				}

			}
			if action.Pagination {
				req.Url.AddQuery("page", ":page", "specify page to load (optional)", true)
				req.Url.AddQuery("size", ":size", "specify size of results (optional, default 10, max 100)", true)
			}
			if len(entity.Association) > 0 && action.Method == "GET" {
				var associations = ""
				for _, association := range entity.Association {
					associations += fmt.Sprintf("%s,", association.Name)
				}
				associations = strings.TrimSuffix(associations, ",")
				req.Url.AddQuery("associations", "", fmt.Sprintf("Accepts \"%s\" as loadable association(s) (optional, accept comma seperated values)", associations), true)
			}
			if action.Method == "GET" {
				req.Url.AddQuery("sort", "field.desc", "sort results by field. field.asc or field.desc (optional, accept comma seperated values)", true)
				req.Url.AddQuery("filters", "field[op]=value", "filter results by field value. (optional)", true)
				for _, item := range entity.Fields {
					req.Url.AddQuery(item.DBName, item.DBName+"[eq]=:value", fmt.Sprintf("filter results by %s (optional, possible operators: eq, neq, gt, gte, lt, lte, in, notin, between, contains, search, isnull, notnull)", item.Name), true)
				}
			}
			req.Url.AddQuery("debug", "debug=restify", "enable debug mode (optional, default false)", true)

			folder.AppendItem(Item{
				Name:        action.Name,
				Description: action.Description,
				Request:     &req,
			})
		}

	}

	if gpath.IsFileExist("./docify/restify.json") {
		err := gpath.Remove("./docify/restify.json")
		if err != nil {
			log.Error("Error writing to file:", err)
		}
	}
	var b, _ = collection.ToJson()
	err := gpath.Write("./docify/restify.json", b)

	if err != nil {
		log.Error("Error writing to file:", err)
	}

	collections = append(collections, *collection)

}

func GenerateDescription(entity serializer.Entity, action *restify.Endpoint) string {
	var description = []string{
		action.Description,
		"---",
	}

	if action.AcceptData {
		description = append(description, "- Accepts body in `application/json`,`application/x-www-form-urlencoded` and `multipart/form-data` format.")
		description = append(description, "- Supports data validation. Refer to [Validation](https://github.com/getevo/restify/blob/master/docs/validation.md)")
		if action.Method == "PATCH" {
			description = append(description, "- Avoid sending primary key in body to update primary key. updating primary key is not allowed.")
		}
	}

	if action.Batch {
		description = append(description, "- Supports batch operations. You can send multiple requests in one request body, this mode requires body `application/json` as array of objects formatted.")
	}

	if action.Pagination {
		description = append(description, "- Supports pagination. You can specify the page number and size using query parameters: `page` and `size`.")
	}

	if action.Filterable {
		description = append(description, "- Supports filterable Resources. You can filter Resources by using query parameters: `field[op]=value`. refer to [Query Parameters Explanation](https://github.com/getevo/restify/blob/master/docs/endpoints.md#query-parameters-explanation)")
	}

	if action.PKUrl {
		description = append(description, "- This endpoint requires a primary key in the URL as following format "+action.AbsoluteURI)
	}

	if action.AcceptData {
		description = append(description, "---")
		description = append(description, "### Acceptable fields and their types:")
		description = append(description, "| Field | Type | Description | Validation |")
		description = append(description, "| ------ | ------ | ------ | ------ |")
		for _, field := range action.Resource.Schema.Fields {
			var jsonField = strings.Split(field.Tag.Get("json"), ",")[0]
			if field.Tag.Get("json") == "-" || strings.Contains(field.Tag.Get("json"), "omit_decode") {
				jsonField = field.Name
			}

			if strings.TrimSpace(string(field.GORMDataType)) == "" {
				continue
			}

			var additional []string
			if t, ok := field.TagSettings["TYPE"]; ok && strings.HasPrefix(t, "enum") {
				additional = append(additional, "`Type:"+t)
			}
			if field.PrimaryKey {
				additional = append(additional, "`Primary Key`")
			}
			if field.AutoIncrement {
				additional = append(additional, "`AutoIncrement`")
			}
			if field.Unique {
				additional = append(additional, "`Unique`")
			}
			if field.HasDefaultValue {
				additional = append(additional, "`Default:"+field.DefaultValue+"`")
			}
			if field.FieldType.Kind() == reflect.Ptr {
				additional = append(additional, "`Accept Null`")
			}
			if field.Size > 0 {
				additional = append(additional, "`Size:"+strconv.Itoa(field.Size)+"`")
			}
			if field.Precision > 0 {
				additional = append(additional, "`Precision:"+strconv.Itoa(field.Precision)+"`")
			}
			if field.Scale > 0 {
				additional = append(additional, "`Scale:"+strconv.Itoa(field.Scale)+"`")
			}
			if !field.Updatable {
				additional = append(additional, "`Cannot be updated`")
			}
			if !field.Creatable {
				additional = append(additional, "`Cannot be created`")
			}
			if !field.Readable {
				additional = append(additional, "`Unreadable`")
			}
			var validation = "`none`"
			if field.Tag.Get("validation") != "" {
				validation = field.Tag.Get("validation")
			}

			description = append(description, fmt.Sprintf("| `%s` | %s | %s | %s |", jsonField, field.GORMDataType, strings.Join(additional, ","), validation))
		}

	}

	if action.Filterable {
		var associations []string
		for _, field := range action.Resource.Schema.Fields {
			if strings.TrimSpace(string(field.GORMDataType)) == "" {

				if field.FieldType.Kind() == reflect.Slice {
					associations = append(associations, fmt.Sprintf("| %s | `%s` | %s |", field.Name, "has many", "associations="+field.Name))
				} else {
					associations = append(associations, fmt.Sprintf("| %s | `%s` | %s |", field.Name, "belongs to", "associations="+field.Name))
				}

			}
		}
		if len(associations) > 0 {
			description = append(description, "---")
			description = append(description, "### Loadable Associations:")
			description = append(description, "| Association | Type | URL Pattern |")
			description = append(description, "| ------ | ------ | ------ |")
			description = append(description, associations...)
			description = append(description, "\n\nmore information: [Query Parameters Explanation](https://github.com/getevo/restify/blob/master/docs/endpoints.md#query-parameters-explanation)")
		}
	}

	return strings.Join(description, "\n")
}
