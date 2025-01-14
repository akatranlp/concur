package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
)

type prefixType string

const (
	PrefixID      prefixType = "ID"
	PrefixPID     prefixType = "PID"
	PrefixName    prefixType = "Name"
	PrefixCommand prefixType = "Command"
)

type Config struct {
	Prefix string `mapstructure:"prefix"`
	Name   string `mapstructure:"name"`
}

type Command struct {
	prefix []byte
	cmd    *exec.Cmd
}

func NewCommand(ctx context.Context, command string, id int) *Command {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Cancel = func() error {
		// CHeck which signal is the best to use SIGINT or SIGTERM or SIGKILL
		err := cmd.Process.Signal(syscall.SIGTERM)
		return err
	}
	return &Command{cmd: cmd, prefix: []byte(fmt.Sprintf("[%d] ", id))}
}

func (c *Command) Run(raw bool, output io.Writer) error {
	if output == nil {
		output = os.Stdout
	}

	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	defer w.Close()

	c.cmd.Stderr = w
	c.cmd.Stdout = w

	if err := c.cmd.Start(); err != nil {
		return err
	}

	go func() {
		if raw {
			n, err := io.Copy(output, r)
			if err != nil {
				log.Println(n, err)
			}
		} else {
			scanner := bufio.NewScanner(r)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				buf := scanner.Bytes()
				buf = append(c.prefix, buf...)
				buf = append(buf, '\n')

				_, err := output.Write(buf)
				if err != nil {
					log.Println(err)
				}
			}
			if err := scanner.Err(); err != nil {
				log.Println(err)
			}
		}
	}()

	err = c.cmd.Wait()
	log.Println("--------------------------------------------")
	log.Println(err)
	log.Println(c.cmd.ProcessState)
	log.Println("--------------------------------------------")
	return err
}
