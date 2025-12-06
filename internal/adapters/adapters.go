package adapters

import (
	"github.com/holydocs/holydocs/internal/adapters/primary/cli"
	docsgen "github.com/holydocs/holydocs/internal/adapters/secondary/docs"
	"github.com/holydocs/holydocs/internal/adapters/secondary/schema"
	"github.com/holydocs/holydocs/internal/core/app"
	do "github.com/samber/do/v2"
)

//nolint:gochecknoglobals // Package variables are required for dependency injection setup
var PrimaryPackage = do.Package(
	do.Lazy[*cli.Command](NewCLICommand),
)

//nolint:gochecknoglobals // Package variables are required for dependency injection setup
var SecondaryPackage = do.Package(
	do.Lazy[*schema.Loader](NewSchemaLoader),
	do.Lazy[*docsgen.Generator](NewDocsGenerator),
)

func NewCLICommand(i do.Injector) (*cli.Command, error) {
	appInstance := do.MustInvoke[*app.App](i)
	schemaLoader := do.MustInvoke[*schema.Loader](i)
	docsGenerator := do.MustInvoke[*docsgen.Generator](i)

	return cli.NewCommand(appInstance, schemaLoader, docsGenerator), nil
}

func NewSchemaLoader(i do.Injector) (*schema.Loader, error) {
	appInstance := do.MustInvoke[*app.App](i)

	return schema.NewLoader(appInstance), nil
}

func NewDocsGenerator(i do.Injector) (*docsgen.Generator, error) {
	appInstance := do.MustInvoke[*app.App](i)

	return docsgen.NewGenerator(appInstance), nil
}
