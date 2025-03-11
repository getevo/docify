package docify

import (
	"fmt"
	"github.com/getevo/docify/serializer"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func GetStructDefinition(t reflect.Type) (*serializer.StructDefinition, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("provided type is not a struct")
	}

	// Get the directory of the package where the type is defined
	dir := "../" + t.PkgPath()

	// Get the name of the struct
	structName := t.Name()

	var structDef serializer.StructDefinition

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, path, nil, parser.AllErrors|parser.ParseComments)
			if err != nil {
				return err
			}

			var commentMap = map[int]string{}

			// ✅ Step 1: Build a map of comments by line number
			for _, commentGroup := range node.Comments {
				for _, comment := range commentGroup.List {
					pos := fset.Position(comment.Pos()).Line
					commentMap[pos+1] = strings.TrimPrefix(comment.Text, "// ")
				}
			}

			// ✅ Step 2: Inspect AST to find the struct definition
			ast.Inspect(node, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if ok && ts.Name.Name == structName {
					st, ok := ts.Type.(*ast.StructType)
					if ok {
						var sb strings.Builder
						var description strings.Builder

						// ✅ Step 3: Extract attached comments for the struct
						if ts.Doc != nil {
							for _, comment := range ts.Doc.List {
								text := strings.TrimPrefix(comment.Text, "// ")
								description.WriteString(text + "\n")
								sb.WriteString(comment.Text + "\n")
							}
						}

						sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

						for _, field := range st.Fields.List {
							names := []string{}
							for _, name := range field.Names {
								names = append(names, name.Name)
							}

							// ✅ Step 4: Use getTypeString to resolve all types
							typeStr := getTypeString(field.Type)

							// ✅ Step 5: Extract struct tag
							var tag string
							if field.Tag != nil {
								tag = strings.Trim(field.Tag.Value, "`")
							}

							// ✅ Step 6: Extract field comment from the map
							pos := fset.Position(field.Pos()).Line
							fieldComment := commentMap[pos]

							// ✅ Step 7: Store field information
							structDef.Fields = append(structDef.Fields, serializer.FieldDefinition{
								Name:        strings.Join(names, ", "),
								Type:        typeStr,
								Tag:         tag,
								Description: fieldComment,
							})

							// ✅ Step 8: Write fields to body
							if fieldComment != "" {
								sb.WriteString(fmt.Sprintf("    \n// %s\n", fieldComment))
							}
							sb.WriteString(fmt.Sprintf("    %s %s", strings.Join(names, ", "), typeStr))
							if tag != "" {
								sb.WriteString(fmt.Sprintf(" `%s`", tag))
							}

							sb.WriteString("\n")
							/*if tag != "" {
								if fieldComment != "" {
									sb.WriteString(fmt.Sprintf("    %s %s `%s` // %s\n", strings.Join(names, ", "), typeStr, tag, fieldComment))
								} else {
									sb.WriteString(fmt.Sprintf("    %s %s `%s`\n", strings.Join(names, ", "), typeStr, tag))
								}
							} else {
								if fieldComment != "" {
									sb.WriteString(fmt.Sprintf("    %s %s // %s\n", strings.Join(names, ", "), typeStr, fieldComment))
								} else {
									sb.WriteString(fmt.Sprintf("    %s %s\n", strings.Join(names, ", "), typeStr))
								}
							}*/
						}

						sb.WriteString("}\n")

						// ✅ Step 9: Format the generated code using go/format
						formattedCode, err := format.Source([]byte(sb.String()))
						if err != nil {
							return true // Ignore formatting errors and keep the original code
						}

						// ✅ Step 10: Store final values
						structDef = serializer.StructDefinition{
							Description: description.String(),
							Body:        string(formattedCode),
							Fields:      structDef.Fields,
							File:        path,
						}

						return false // Stop searching once the struct is found
					}
				}
				return true
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if structDef.Body == "" {
		return nil, fmt.Errorf("struct definition not found")
	}

	return &structDef, nil
}

// ✅ Recursive function to resolve field types (including generics)
func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr: // Pointers
		return "*" + getTypeString(t.X)
	case *ast.ArrayType: // Arrays and Slices
		if t.Len != nil {
			return fmt.Sprintf("[%s]%s", getTypeString(t.Len), getTypeString(t.Elt))
		}
		return "[]" + getTypeString(t.Elt)
	case *ast.MapType: // Maps
		return fmt.Sprintf("map[%s]%s", getTypeString(t.Key), getTypeString(t.Value))
	case *ast.SelectorExpr: // Qualified types (e.g., time.Time)
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	case *ast.InterfaceType: // Interfaces
		return "interface{}"
	case *ast.StructType: // Inline structs
		return "struct{...}"
	case *ast.IndexExpr: // Generics like types.JSONType[map[string]interface{}]
		base := getTypeString(t.X)
		index := getTypeString(t.Index)
		return fmt.Sprintf("%s[%s]", base, index)
	case *ast.IndexListExpr: // Multiple type parameters (e.g., T[K, V])
		base := getTypeString(t.X)
		var params []string
		for _, index := range t.Indices {
			params = append(params, getTypeString(index))
		}
		return fmt.Sprintf("%s[%s]", base, strings.Join(params, ", "))
	}
	return "unknown"
}
