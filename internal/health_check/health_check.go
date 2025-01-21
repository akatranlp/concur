package healthcheck

import (
	"context"
	"fmt"

	"github.com/akatranlp/concur/internal/config"
)

type HealthChecker interface {
	Start(ctx context.Context)
	GetHealthCheckMessage(ctx context.Context) (messageRows []string, rows int)
}

func HealthCheckFactory(cfg config.StatusCheckConfig) (HealthChecker, error) {
	switch cfg.Type {
	case config.CheckTypeCommand:
		return NewCommandHealthChecker(cfg.Command, cfg.Interval), nil
	case config.CheckTypeHTTP:
		return NewHTTPHealthChecker(cfg.URL, cfg.Template, cfg.Interval)
	}
	return nil, fmt.Errorf("invalid check type: %s", cfg.Type)
}
