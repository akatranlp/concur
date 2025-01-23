package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"

	"github.com/akatranlp/concur/internal/config"
	"github.com/akatranlp/concur/internal/logger"
)

type Command struct {
	cfg config.RunCommandConfig
	cmd *exec.Cmd
	r   *os.File
	w   *os.File
}

func NewCommand(ctx context.Context, killSignal syscall.Signal, cfg config.RunCommandConfig) *Command {
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
	return &Command{cfg: cfg, cmd: cmd}
}

func (c *Command) Kill() error {
	return c.cmd.Process.Kill()
}

func (c *Command) StartWithPrefix() (pid int, err error) {
	r, w, err := os.Pipe()
	if err != nil {
		return -1, err
	}

	c.cmd.Stdout = w
	c.cmd.Stderr = w
	c.r = r
	c.w = w

	if err := c.cmd.Start(); err != nil {
		return -1, err
	}

	return c.cmd.Process.Pid, nil
}

func (c *Command) StartRaw() error {
	c.cmd.Stderr = os.Stderr
	c.cmd.Stdout = os.Stdout

	return c.cmd.Start()
}

func (c *Command) WaitRaw() error {
	err := c.cmd.Wait()
	fmt.Printf("%s exited with %s\n", c.cfg.Command, c.cmd.ProcessState)
	return err
}

func (c *Command) RunRaw() error {
	if err := c.StartRaw(); err != nil {
		return err
	}
	return c.WaitRaw()
}

func (c *Command) WaitWithPrefix(id int, msgCh chan<- logger.Message) error {
	done := make(chan struct{})
	go func() {
		defer close(done)

		scanner := bufio.NewScanner(c.r)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			text := scanner.Text() + "\n\033[0m"
			msgCh <- logger.Message{ID: id, Text: text}

			if c.cfg.Debug {
				for _, b := range text {
					fmt.Printf("%s", strconv.QuoteRuneToASCII(rune(b)))
				}
				fmt.Println()
			}
		}
		if err := scanner.Err(); err != nil {
			log.Println(err)
		}
	}()

	err := c.cmd.Wait()
	_ = c.w.Close()
	<-done
	msgCh <- logger.Message{ID: id, Text: fmt.Sprintf("%s exited with %s\n", c.cfg.Command, c.cmd.ProcessState)}
	return err
}
