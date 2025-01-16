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

type Command struct {
	cfg    RunCommandConfig
	idx    int
	prefix *Prefix
	cmd    *exec.Cmd
	r      *os.File
	w      *os.File
}

func NewCommand(ctx context.Context, idx int, prefix Prefix, killSignal syscall.Signal, cfg RunCommandConfig) *Command {
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
			err = cmd.Process.Signal(killSignal)
		}
		return err
	}
	cmd.Dir = cfg.CWD
	return &Command{cfg: cfg, cmd: cmd, idx: idx, prefix: &prefix}
}

func (c *Command) Start() error {
	if *c.cfg.Raw {
		return c.startRaw()
	} else {
		return c.startWithPrefix()
	}
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

func (c *Command) Wait(output io.Writer) error {
	if output == nil {
		output = os.Stdout
	}

	if *c.cfg.Raw {
		return c.waitRaw(output)
	} else {
		return c.waitWithPrefix(output)
	}
}

func (c *Command) startRaw() error {
	c.cmd.Stderr = os.Stderr
	c.cmd.Stdout = os.Stdout

	return c.cmd.Start()
	// _, _ = fmt.Printf("%s exited with %s\n", c.cfg.Command, c.cmd.ProcessState)
}

func (c *Command) startWithPrefix() error {
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	c.cmd.Stdout = w
	c.cmd.Stderr = w
	c.r = r
	c.w = w

	if err := c.cmd.Start(); err != nil {
		return err
	}

	c.prefix.Index = c.idx
	c.prefix.Name = c.cfg.Name
	c.prefix.Command = c.cfg.Command
	c.prefix.Pid = c.cmd.Process.Pid

	return nil
}

func (c *Command) runRaw(output io.Writer) error {
	if err := c.startRaw(); err != nil {
		return err
	}
	return c.waitRaw(output)
}

func (c *Command) runWithPrefix(output io.Writer) error {
	if err := c.startWithPrefix(); err != nil {
		return err
	}
	return c.waitWithPrefix(output)
}

func (c *Command) waitRaw(output io.Writer) error {
	err := c.cmd.Wait()
	_, _ = fmt.Fprintf(output, "%s exited with %s\n", c.cfg.Command, c.cmd.ProcessState)
	return err
}

func (c *Command) waitWithPrefix(output io.Writer) error {
	prefix := c.prefix

	done := make(chan struct{})
	go func() {
		defer close(done)

		scanner := bufio.NewScanner(c.r)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			buf := scanner.Bytes()
			buf = append([]byte(prefix.Apply(&c.cfg.PrefixColor)), buf...)
			buf = append(buf, '\n')
			buf = append(buf, "\033[0m"...)

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

	err := c.cmd.Wait()
	_ = c.w.Close()
	<-done
	_, _ = fmt.Fprintf(output, "%s%s exited with %s\n", prefix.Apply(&c.cfg.PrefixColor), c.cfg.Command, c.cmd.ProcessState)
	return err
}
