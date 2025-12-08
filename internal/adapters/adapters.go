package adapters

import (
	"github.com/holydocs/holydocs/internal/adapters/primary/cli"
	docsgen "github.com/holydocs/holydocs/internal/adapters/secondary/docs"
	"github.com/holydocs/holydocs/internal/adapters/secondary/schema"
	"github.com/holydocs/holydocs/internal/adapters/secondary/target"
	do "github.com/samber/do/v2"
)

//nolint:gochecknoglobals // Package variables are required for dependency injection setup
var PrimaryPackage = do.Package(
	do.Lazy[*cli.Command](cli.NewCommand),
)

//nolint:gochecknoglobals // Package variables are required for dependency injection setup
var SecondaryPackage = do.Package(
	do.Lazy[*schema.Loader](schema.NewLoader),
	do.Lazy[*docsgen.Generator](docsgen.NewGenerator),
	do.Lazy(target.NewTargetProvider),
)
