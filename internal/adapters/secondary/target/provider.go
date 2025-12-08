package target

import (
	"fmt"

	d2target "github.com/holydocs/holydocs/internal/adapters/secondary/target/d2"
	"github.com/holydocs/holydocs/internal/config"
	"github.com/holydocs/holydocs/internal/core/domain"
	do "github.com/samber/do/v2"
)

// NewTargetProvider creates a domain.Target from config and registers it in DI.
func NewTargetProvider(i do.Injector) (domain.Target, error) {
	cfg := do.MustInvoke[*config.Config](i)

	target, err := d2target.NewTarget(cfg.Diagram.D2)
	if err != nil {
		return nil, fmt.Errorf("creating D2 target: %w", err)
	}

	return target, nil
}
