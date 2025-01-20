package logger

import (
	"context"
	"fmt"
	"os"
	"time"

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

	healthChecker       hc.HealthChecker
	healthCheckInterval time.Duration
	done                chan struct{}
	msgCh               chan Message
}

func NewPrefixLogger(p *prefix.Prefix, output *os.File, healthCheckEnabled bool, healthChecker hc.HealthChecker) *PrefixLogger {
	return &PrefixLogger{
		prefix:              p,
		out:                 output,
		healthChecker:       healthChecker,
		healthCheckInterval: 1,
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

func (l *PrefixLogger) Run() {
	defer close(l.done)
	ticker := time.NewTicker(l.healthCheckInterval * time.Second)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		select {
		case <-ticker.C:
			if l.healthChecker == nil {
				continue
			}
			message, oldRows := l.healthChecker.GetHealthCheckMessage(ctx)
			if oldRows > 0 {
				l.out.WriteString(fmt.Sprintf("\033[%dA\033[0J", oldRows))
			}
			l.RenderHealthCheck(message)
		case msg, ok := <-l.msgCh:
			if !ok {
				cancel()
				return
			}

			prefix := l.prefix.Render(msg.ID, true)

			if l.healthChecker != nil {
				message, oldRows := l.healthChecker.GetHealthCheckMessage(ctx)
				if oldRows > 0 {
					l.out.WriteString(fmt.Sprintf("\033[%dA\033[0J", oldRows))
				}

				l.out.WriteString(prefix)
				l.out.WriteString(msg.Text)
				l.RenderHealthCheck(message)

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
		l.out.WriteString(fmt.Sprintf("\033[31;1m[HEALTH]\033[0m %s\033[0m\n", message))
	}
}

func (l *PrefixLogger) Wait() {
	<-l.done
}
