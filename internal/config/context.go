package config

import "context"

// Context key type for configuration.
type configKey struct{}

// GetFromContext retrieves the configuration from the command context.
func GetFromContext(ctx context.Context) (*Config, bool) {
	cfg, ok := ctx.Value(configKey{}).(*Config)

	return cfg, ok
}

// WithContext stores the configuration in the context.
func WithContext(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configKey{}, cfg)
}
