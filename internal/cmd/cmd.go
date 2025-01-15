package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"
)

type prefixType string

const (
	PrefixIdx     prefixType = "idx"
	PrefixPID     prefixType = "pid"
	PrefixName    prefixType = "Name"
	PrefixCommand prefixType = "command"
)

type Command struct {
	cfg        RunCommandConfig
	idx        int
	prefixType prefixType
	cmd        *exec.Cmd
}

func NewCommand(ctx context.Context, idx int, prefixType prefixType, cfg RunCommandConfig) *Command {
	var arg0, arg1 string
	if runtime.GOOS == "windows" {
		arg0, arg1 = "cmd", "/c"
	} else {
		arg0, arg1 = "sh", "-c"
	}
	cmd := exec.CommandContext(ctx, arg0, arg1, cfg.Command)

	cmd.Cancel = func() error {
		var err error
		if runtime.GOOS == "windows" {
			err = cmd.Process.Kill()
		} else {
			// Check which signal is the best to use SIGINT or SIGTERM or SIGKILL
			err = cmd.Process.Signal(syscall.SIGINT)
		}
		return err
	}
	cmd.Dir = cfg.CWD
	return &Command{cfg: cfg, cmd: cmd, idx: idx, prefixType: prefixType}
}

func (c *Command) Run(output io.Writer) error {
	if output == nil {
		output = os.Stdout
	}

	if *c.cfg.Raw {
		return c.runRaw(output)
	} else {
		return c.runWithPrefix(output)
	}
}

func (c *Command) runRaw(output io.Writer) error {
	c.cmd.Stderr = os.Stderr
	c.cmd.Stdout = os.Stdout

	err := c.cmd.Run()
	_, _ = fmt.Fprintf(output, "%s exited with %s\n", c.cfg.Command, c.cmd.ProcessState)
	return err
}

func (c *Command) runWithPrefix(output io.Writer) error {
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	c.cmd.Stdout = w
	c.cmd.Stderr = w

	if err := c.cmd.Start(); err != nil {
		return err
	}

	prefix := c.getPrefix()

	done := make(chan struct{})
	go func() {
		defer close(done)

		scanner := bufio.NewScanner(r)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			buf := scanner.Bytes()
			buf = append(prefix, buf...)
			buf = append(buf, '\n')
			buf = append(buf, colorReset...)

			_, err := output.Write(buf)
			if c.cfg.Debug {
				for _, b := range buf {
					fmt.Printf("%s", strconv.QuoteRuneToASCII(rune(b)))
				}
				fmt.Println()
			}
			if err != nil {
				log.Println(err)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Println(err)
		}
	}()

	err = c.cmd.Wait()
	_ = w.Close()
	<-done
	_, _ = fmt.Fprintf(output, "%s%s exited with %s\n", string(prefix), c.cfg.Command, c.cmd.ProcessState)
	return err
}

func (c *Command) getPrefix() []byte {
	var prefix string

	switch c.prefixType {
	case PrefixIdx:
		prefix = fmt.Sprintf("[%d]", c.idx)
	case PrefixPID:
		prefix = fmt.Sprintf("[%d]", c.cmd.Process.Pid)
	case PrefixName:
		prefix = fmt.Sprintf("[%s]", c.cfg.Name)
	case PrefixCommand:
		prefix = fmt.Sprintf("[%s]", c.cfg.Command)
	default:
		if c.cfg.Name != "" {
			prefix = fmt.Sprintf("[%s]", c.cfg.Name)
		} else {
			prefix = fmt.Sprintf("[%d]", c.idx)
		}
	}

	return append([]byte(ColorizeString(c.cfg.PrefixColor, prefix)), ' ')
}
