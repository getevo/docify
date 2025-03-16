package markdown

import (
	"fmt"
	"github.com/getevo/docify/serializer"

	"github.com/getevo/evo/v2/lib/gpath"
	md "github.com/nao1215/markdown"
	"github.com/olekukonko/tablewriter"
	"os"
	"strings"
)

var Br = "------------------------------------------------------------------------------------------\n"

type Doc struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Entities    []serializer.Entity `json:"entities"`
}

type Attributes []string

func (a *Attributes) Add(attr string) {
	*a = append(*a, attr)
}

func (a Attributes) Render() string {
	if len(a) == 0 {
		return ""
	}
	return "`" + strings.Join(a, "`  `") + "`"
}

func Generate(project *serializer.Doc) {
	_ = gpath.MakePath("./docify")
	// Open the file with create/truncate mode (overwrite if it exists)
	file, err := os.OpenFile("./docify/readme.md", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Initialize markdown writer
	doc := md.NewMarkdown(file)

	doc.H1(project.Title)
	doc.H2(project.Description)

	doc.LF()

	doc.H2("Table of Entities")

	var links []string

	for _, item := range project.Entities {
		var path = "./docify/" + item.Pkg + "." + item.Name + ".md"
		GenerateEntityDoc(path, item)
		links = append(links, md.Link(item.Pkg+"."+item.Name, item.Pkg+"."+item.Name+".md"))
	}
	doc.BulletList(links...)

	doc.LF()
	doc.H2("Postman Collections")
	doc.PlainText(md.Link("Download Restify Collection", "./restify.json"))

	err = doc.Build()
	if err != nil {
		panic(err)
	}
}

func GenerateEntityDoc(path string, entity serializer.Entity) {
	fmt.Println("Postman Entity: " + entity.Name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Initialize markdown writer
	doc := md.NewMarkdown(file)

	doc.H1(entity.Name)
	doc.H2(entity.Pkg + "." + entity.Name)
	doc.PlainText(entity.Definition.Description)
	doc.LF()

	var filePath = trimDir(entity.Definition.File)
	doc.PlainText("\n> Source: " + md.Link(filePath, "../"+filePath))

	doc.LF()

	doc.H2("Definition")
	doc.CodeBlocks(md.SyntaxHighlightGo, entity.Definition.Body)

	doc.PlainText(Br)

	doc.H2("Fields")

	var tb = md.TableSet{
		Header: []string{"Name", "Data Type", "Specifications", "Validation", "Description"},
	}
	for _, param := range entity.Fields {
		var attributes Attributes
		if param.PrimaryKey {
			attributes.Add("Primary Key")
		}

		if param.Nullable {
			attributes.Add("Accepts Null")
		}
		if param.AutoIncrement {
			attributes.Add("Auto Increment")
		}
		if param.Unique {
			attributes.Add("Unique")
		}
		if param.UniqueIndex != "" {
			attributes.Add("Unique Index: " + param.UniqueIndex)
		}
		if param.Indexed {
			attributes.Add("Indexed")
		}
		if len(param.Enum) > 0 {
			attributes.Add(fmt.Sprintf("Enum: %s", strings.Join(param.Enum, ",")))
		}
		if param.Default != "" {
			attributes.Add(fmt.Sprintf("Default Value: %s", param.Default))
		}

		var row = []string{
			param.JsonTag,
			param.JsonType,
			attributes.Render(),
			param.Validation,
			param.Description,
		}
		tb.Rows = append(tb.Rows, row)
	}

	doc.PlainText(GetTable(tb))

	doc.H2("APIs:")

	for _, item := range entity.Endpoints {
		doc.PlainText("<details>")
		doc.PlainTextf("<summary><code>%s</code> <code><b>%s</b></code> <code>%s</code> <code>%s</code></summary>\n", item.Method, item.AbsoluteURI, item.Name, item.Description)
		doc.H5("Parameters")

		if item.AcceptData {
			doc.PlainText("> Accepts: `application/json`,`application/x-www-form-urlencoded`,`multipart/form-data`")
			var p = "\n> "
			if item.Batch {
				p += md.Link("[]"+entity.Name, "#fields") + " (Array of Objects)"
			} else {
				p += md.Link(entity.Name, "#fields") + " (Object)"
			}

			doc.PlainText(p)
			doc.LF()
			doc.PlainText("<details>")
			doc.PlainTextf("<summary><code>JSON Example</code></summary>\r\n")
			var body = ""
			if item.Batch {
				body = entity.DataSample.BatchJSON
			} else {
				if item.Method == "PUT" {
					body = entity.DataSample.CreateJSON
				} else {
					body = entity.DataSample.UpdateJSON
				}

			}
			doc.CodeBlocks(md.SyntaxHighlightJSON, body)
			doc.PlainText("</details>")

		} else {
			doc.PlainText("> None")
		}
		doc.LF()
		doc.H5("Query Parameters")
		if item.Filterable {
			doc.PlainTextf("\n> Filters: %s", md.Link("Filters Guide", "https://github.com/getevo/restify/blob/master/docs/endpoints.md#query-parameters-explanation"))
			if item.Method == "GET" {
				doc.PlainTextf("\n> Offset, Limit and Pagination: %s", md.Link("Pagination Guide", "https://github.com/getevo/restify/blob/master/docs/endpoints.md#offset-and-limit"))
				doc.PlainTextf("\n> Limiting fields: %s", md.Link("Select Specific Fields Guide", "https://github.com/getevo/restify/blob/master/docs/endpoints.md#select-specific-fields"))
				doc.PlainTextf("\n> Aggregations: %s", md.Link("Aggregation Guide", "https://github.com/getevo/restify/blob/master/docs/endpoints.md#aggregation"))
				if len(entity.Association) > 0 {
					var tb = md.TableSet{
						Header: []string{"Assoc. Query Parameter", "Data Type", "Type", "Example"},
					}
					for _, assoc := range entity.Association {
						if assoc.Entity == nil {
							continue
						}
						var t = "Object"
						if assoc.Array {
							t = "Array of Objects"
						}
						tb.Rows = append(tb.Rows, []string{
							assoc.Name,
							md.Link(assoc.Entity.Pkg+"."+assoc.Entity.Name, "./"+assoc.Entity.Pkg+"."+assoc.Entity.Name+".md"),
							t,
							fmt.Sprintf("%s?associations=%s", item.AbsoluteURI, assoc.Name),
						})

					}

					doc.PlainTextf("\n> Loading Associations: %s", md.Link("Associations Guide", "https://github.com/getevo/restify/blob/master/docs/endpoints.md#loading-associations"))
					doc.PlainTextf("\n")
					doc.Table(tb)
				}
			}

		} else {
			doc.PlainText("> None")
		}

		doc.H5("Response")
		var tb = md.TableSet{
			Header: []string{"Status Code", "Content Type", "Response Type", "Data"},
		}
		var body = ""

		if item.Batch {
			body += md.Link("[]"+entity.Name, "#fields") + " (Array of Objects)"
		} else {
			body += md.Link(entity.Name, "#fields") + " (Object)"
		}

		if item.Method == "DELETE" {
			body += "No Content"
		}

		tb.Rows = [][]string{
			{"200", "application/json", "Success", body},
			{"400", "application/json", "Validation Error", md.Link("Validation Guide", "https://github.com/getevo/restify/blob/master/docs/developer.md#validation-in-restify")},
			{"400", "application/json", "Bad Request"},
			{"401", "application/json", "Unauthorized"},
			{"403", "application/json", "Forbidden"},
			{"404", "application/json", "Not Found"},
			{"500", "application/json", "Internal Server Error"},
		}
		doc.PlainText(GetTable(tb))

		doc.PlainText("\n")

		doc.PlainText("</details>\n")
		doc.PlainText(Br)
	}

	err = doc.Build()
	if err != nil {
		panic(err)
	}
}

func GetTable(tb md.TableSet) string {
	buf := &strings.Builder{}
	table := tablewriter.NewWriter(buf)
	table.SetNewLine("\r\n")
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetCenterSeparator("|")
	table.SetHeader(tb.Header)
	table.AppendBulk(tb.Rows)
	table.Render()
	return buf.String()
}

func trimDir(path string) string {
	// Split on backslash
	parts := strings.Split(path, `\`)

	// We're looking specifically for a leading ".." plus one more directory
	if len(parts) > 2 && parts[0] == ".." {
		// Skip the first two parts: ".." and the next directory
		return strings.Join(parts[2:], `\`)
	}
	// Return as-is if it doesn't match the pattern
	return path
}
