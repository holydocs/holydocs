package config

import (
	do "github.com/samber/do/v2"
)

// Package provides dependency injection for configuration.
// Note: Config is loaded from command context, not from DI container.
//
//nolint:gochecknoglobals // Package variable is required for dependency injection setup
var Package = do.Package()
