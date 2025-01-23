package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/akatranlp/concur/internal/config"
	hc "github.com/akatranlp/concur/internal/health_check"
)

type RawLogger struct {
	healthCheckers      []hc.HealthChecker
	healthCheckInterval time.Duration
	healthCheckerPrefix string
	done                chan struct{}
	msgCh               chan Message
}

func NewRawLogger(healthCheckers []hc.HealthChecker, cfg config.StatusConfig) *RawLogger {
	var healthCheckerPrefix string
	if cfg.Enabled {
		healthCheckerPrefix = cfg.Sequence.Apply("["+cfg.Text+"]") + " "
	}
	return &RawLogger{
		healthCheckers:      healthCheckers,
		healthCheckInterval: 1,
		healthCheckerPrefix: healthCheckerPrefix,
		msgCh:               make(chan Message, 100),
		done:                make(chan struct{}),
	}
}

func (l *RawLogger) Run(ctx context.Context) {
	if len(l.healthCheckers) == 0 {
		return
	}
	defer close(l.done)
	ticker := time.NewTicker(l.healthCheckInterval * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healthMessages := make([]string, 0)

			for _, hc := range l.healthCheckers {
				message, _ := hc.GetHealthCheckMessage(ctx)
				healthMessages = append(healthMessages, message...)
			}

			l.RenderHealthCheck(healthMessages)
		}
	}
}

func (l *RawLogger) RenderHealthCheck(rows []string) {
	for _, message := range rows {
		os.Stdout.WriteString(fmt.Sprintf("%s%s\033[0m\n", l.healthCheckerPrefix, message))
	}
}

func (l *RawLogger) Wait() {
	<-l.done
}
