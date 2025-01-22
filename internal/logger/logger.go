package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/akatranlp/concur/internal/config"
	hc "github.com/akatranlp/concur/internal/health_check"
	"github.com/akatranlp/concur/internal/prefix"
)

type Message struct {
	ID   int
	Text string
}

type PrefixLogger struct {
	prefix *prefix.Prefix
	out    *os.File

	healthCheckers      []hc.HealthChecker
	healthCheckInterval time.Duration
	healthCheckerPrefix string
	done                chan struct{}
	msgCh               chan Message
}

func NewPrefixLogger(p *prefix.Prefix, output *os.File, healthCheckers []hc.HealthChecker, cfg config.StatusConfig) *PrefixLogger {
	var healthCheckerPrefix string
	if cfg.Enabled {
		healthCheckerPrefix = cfg.Sequence.Apply("["+cfg.Text+"]") + " "
	}
	return &PrefixLogger{
		prefix:              p,
		out:                 output,
		healthCheckers:      healthCheckers,
		healthCheckInterval: 1,
		healthCheckerPrefix: healthCheckerPrefix,
		msgCh:               make(chan Message, 100),
		done:                make(chan struct{}),
	}
}

func (l *PrefixLogger) GetMessageChannel() chan<- Message {
	return l.msgCh
}

func (l *PrefixLogger) Close() {
	close(l.msgCh)
}

func (l *PrefixLogger) Run(ctx context.Context) {
	defer close(l.done)
	ticker := time.NewTicker(l.healthCheckInterval * time.Second)

	var done bool

	go func() {
		<-ctx.Done()
		done = true
	}()

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		select {
		case <-ticker.C:
			if done || len(l.healthCheckers) == 0 {
				continue
			}

			healthMessages := make([]string, 0)
			oldHelthMessageRows := 0

			for _, hc := range l.healthCheckers {
				message, oldRows := hc.GetHealthCheckMessage(ctx)
				healthMessages = append(healthMessages, message...)
				oldHelthMessageRows += oldRows
			}

			if oldHelthMessageRows > 0 {
				l.out.WriteString(fmt.Sprintf("\033[%dA\033[0J", oldHelthMessageRows))
			}
			l.RenderHealthCheck(healthMessages)
		case msg, ok := <-l.msgCh:
			if !ok {
				cancel()
				return
			}

			prefix := l.prefix.Render(msg.ID, true)

			if !done && len(l.healthCheckers) > 0 {
				healthMessages := make([]string, 0)
				oldHelthMessageRows := 0

				for _, hc := range l.healthCheckers {
					message, oldRows := hc.GetHealthCheckMessage(ctx)
					healthMessages = append(healthMessages, message...)
					oldHelthMessageRows += oldRows
				}
				if oldHelthMessageRows > 0 {
					l.out.WriteString(fmt.Sprintf("\033[%dA\033[0J", oldHelthMessageRows))
				}

				l.out.WriteString(prefix)
				l.out.WriteString(msg.Text)
				l.RenderHealthCheck(healthMessages)

			} else {
				l.out.WriteString(prefix)
				l.out.WriteString(msg.Text)
			}
		}
		cancel()
	}
}

func (l *PrefixLogger) RenderHealthCheck(rows []string) {
	for _, message := range rows {
		l.out.WriteString(fmt.Sprintf("%s%s\033[0m\n", l.healthCheckerPrefix, message))
	}
}

func (l *PrefixLogger) Wait() {
	<-l.done
}
