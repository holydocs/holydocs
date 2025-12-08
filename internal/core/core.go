package core

import (
	docsgen "github.com/holydocs/holydocs/internal/adapters/secondary/docs"
	"github.com/holydocs/holydocs/internal/adapters/secondary/schema"
	"github.com/holydocs/holydocs/internal/config"
	"github.com/holydocs/holydocs/internal/core/app"
	"github.com/holydocs/holydocs/internal/core/domain"
	do "github.com/samber/do/v2"
)

//nolint:gochecknoglobals // Package variable is required for dependency injection setup
var Package = do.Package(
	do.Lazy[*app.App](NewApp),
)

func NewApp(i do.Injector) (*app.App, error) {
	return app.NewApp(
		do.MustInvoke[*schema.Loader](i),
		do.MustInvoke[*docsgen.Generator](i),
		do.MustInvoke[domain.Target](i),
		do.MustInvoke[*config.Config](i),
	), nil
}
