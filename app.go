package docify

import (
	"github.com/getevo/docify/markdown"
	"github.com/getevo/docify/openapi"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/args"
	"os"
	"time"
)

var OpenAPI *openapi.OpenAPI

type App struct {
}

func (a App) Priority() application.Priority {
	return application.LOWEST
}

func (a App) Register() error {

	return nil
}

func (a App) Router() error {

	return nil
}

func (a App) WhenReady() error {
	if args.Exists("--docify") {

		go func() {
			time.Sleep(1 * time.Second)
			SerializeEntities()

			markdown.Generate(&doc)

			os.Exit(1)
		}()
	}
	return nil
}

func (a App) Name() string {
	return "docify"
}
