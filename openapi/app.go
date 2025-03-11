package openapi

import (
	"github.com/getevo/evo/v2/lib/gpath"
	"github.com/getevo/evo/v2/lib/log"
	"gopkg.in/yaml.v3"
	"os"
)

func Initialize() *OpenAPI {
	var obj OpenAPI
	var filename = "docify/openapi.yml"
	if gpath.IsFileExist(filename) {
		file, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
			return nil
		}

		err = yaml.Unmarshal(file, &obj)
		if err != nil {
			log.Fatal(err)
			return nil
		}

	}
	obj.ParseRestify()
	return &obj
}
