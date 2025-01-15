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
	PrefixID      prefixType = "ID"
	PrefixPID     prefixType = "PID"
	PrefixName    prefixType = "Name"
	PrefixCommand prefixType = "Command"
)

type Command struct {
	cfg    RunCommandConfig
	prefix []byte
	cmd    *exec.Cmd
}

func NewCommand(ctx context.Context, idx int, cfg RunCommandConfig) *Command {
	cmd := exec.CommandContext(ctx, "sh", "-c", cfg.Command)
	var prefix string
	if cfg.Name != "" {
		prefix = fmt.Sprintf("[%s] ", cfg.Name)
	} else {
		prefix = fmt.Sprintf("[%d] ", idx)
	}

	cmd.Cancel = func() error {
		// CHeck which signal is the best to use SIGINT or SIGTERM or SIGKILL
		var err error
		if runtime.GOOS == "windows" {
			err = cmd.Process.Kill()
		} else {
			err = cmd.Process.Signal(syscall.SIGINT)
		}
		fmt.Println(cfg.Command, "received signal", "SIGINT", err)
		return err
	}
	cmd.Dir = cfg.CWD
	return &Command{cfg: cfg, cmd: cmd, prefix: []byte(ColorizeString(cfg.PrefixColor, prefix))}
}

func (c *Command) Run(output io.Writer) error {
	if output == nil {
		output = os.Stdout
	}

	var err error
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	defer w.Close()

	if *c.cfg.Raw {
		c.cmd.Stderr = os.Stderr
		c.cmd.Stdout = os.Stdout
	} else {
		c.cmd.Stdout = w
	}

	if err := c.cmd.Start(); err != nil {
		return err
	}

	go func() {
		if !*c.cfg.Raw {
			scanner := bufio.NewScanner(r)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				buf := scanner.Bytes()
				buf = append(c.prefix, buf...)
				buf = append(buf, '\n')

				if c.cfg.Debug {
					for _, b := range buf {
						fmt.Printf("%s", strconv.QuoteRuneToASCII(rune(b)))
					}
					fmt.Println()
				}

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
