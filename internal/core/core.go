package core

import (
	"github.com/holydocs/holydocs/internal/core/app"
	do "github.com/samber/do/v2"
)

//nolint:gochecknoglobals // Package variable is required for dependency injection setup
var Package = do.Package(
	do.Lazy[*app.App](NewApp),
)

func NewApp(_ do.Injector) (*app.App, error) {
	return app.NewApp(), nil
}
