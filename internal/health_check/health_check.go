package healthcheck

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"time"
)

type HealthChecker interface {
	Start(ctx context.Context)
	GetHealthCheckMessage(ctx context.Context) (messageRows []string, rows int)
}

type CommandHealthChecker struct {
	commands []string

	messages []string
	lastRows int
}

func NewCommandHealthChecker(commands []string) *CommandHealthChecker {
	return &CommandHealthChecker{
		commands: commands,
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
				ticker = time.NewTicker(2 * time.Second)
			}
			c.runCommand(ctx)
		}
	}
}

func (c *CommandHealthChecker) runCommand(ctx context.Context) error {
	var buf bytes.Buffer

	for _, command := range c.commands {
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
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
	}

	var messages []string
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		text := scanner.Text()
		messages = append(messages, text)
	}

	lastRows := len(c.messages)
	c.messages = messages
	c.lastRows = lastRows
	return scanner.Err()
}
