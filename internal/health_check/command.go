package healthcheck

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"time"
)

type CommandHealthChecker struct {
	command  string
	interval time.Duration

	messages []string
	lastRows int
}

func NewCommandHealthChecker(command string, interval time.Duration) *CommandHealthChecker {
	return &CommandHealthChecker{
		command:  command,
		interval: interval,
	}
}

func (c *CommandHealthChecker) GetHealthCheckMessage(context.Context) (messageRows []string, rows int) {
	newMessages := make([]string, len(c.messages))
	copy(newMessages, c.messages)
	lastRows := c.lastRows
	c.lastRows = len(newMessages)
	return newMessages, lastRows
}

func (c *CommandHealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Millisecond)
	first := true
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if first {
				first = false
				ticker = time.NewTicker(c.interval)
			}
			c.runCommand(ctx)
		}
	}
}

func (c *CommandHealthChecker) runCommand(ctx context.Context) error {
	var buf bytes.Buffer

	cmd := exec.CommandContext(ctx, "sh", "-c", c.command)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return err
	}

	err := cmd.Wait()

	if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	if err != nil {
		buf.WriteString(err.Error())
	}

	var messages []string
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		text := scanner.Text()
		messages = append(messages, text)
	}

	c.messages = messages
	return scanner.Err()
}
