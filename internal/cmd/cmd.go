package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"
)

type Prefix struct {
	template *template.Template
	input    string
	Index    int
	Name     string
	Command  string
	Pid      int
	Time     string
}

var templateRegex = regexp.MustCompile(`\{\{.*\}\}`)

func NewPrefix(input string) (p Prefix, err error) {
	switch input {
	case "idx", "index", "name", "", "pid", "time":
		p.input = input
		return
	default:
		if !templateRegex.MatchString(input) {
			err = fmt.Errorf("invalid prefix template: %s", input)
			return
		}
		p.template, err = template.New("prefix").Parse(input)
		return
	}
}

func (p Prefix) Apply(seq sequence) string {
	var prefix string

	if p.template != nil {
		p.Time = time.Now().Format("15:04:05")
		var buf strings.Builder
		if err := p.template.Execute(&buf, p); err != nil {
			panic(err)
		}
		prefix = fmt.Sprintf("[%s]", buf.String())
	} else {
		switch p.input {
		case "":
			if p.Name != "" {
				prefix = fmt.Sprintf("[%s]", p.Name)
			} else {
				prefix = fmt.Sprintf("[%d]", p.Index)
			}
		case "idx", "index":
			prefix = fmt.Sprintf("[%d]", p.Index)
		case "name":
			if p.Name != "" {
				prefix = fmt.Sprintf("[%s]", p.Name)
			} else {
				prefix = fmt.Sprintf("[%s]", p.Command)
			}
		case "pid":
			prefix = fmt.Sprintf("[%d]", p.Pid)
		case "time":
			prefix = fmt.Sprintf("[%s]", time.Now().Format("15:04:05"))
		default:
			panic("unreachable")
		}
	}

	return seq.Apply(prefix) + " "
}

type Command struct {
	cfg    RunCommandConfig
	idx    int
	prefix *Prefix
	cmd    *exec.Cmd
}

func NewCommand(ctx context.Context, idx int, prefix Prefix, cfg RunCommandConfig) *Command {
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
	return &Command{cfg: cfg, cmd: cmd, idx: idx, prefix: &prefix}
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

	c.prefix.Index = c.idx
	c.prefix.Name = c.cfg.Name
	c.prefix.Command = c.cfg.Command
	c.prefix.Pid = c.cmd.Process.Pid
	prefix := c.prefix

	done := make(chan struct{})
	go func() {
		defer close(done)

		scanner := bufio.NewScanner(r)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			buf := scanner.Bytes()
			buf = append([]byte(prefix.Apply(c.cfg.PrefixColor)), buf...)
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

	err = c.cmd.Wait()
	_ = w.Close()
	<-done
	_, _ = fmt.Fprintf(output, "%s%s exited with %s\n", prefix.Apply(c.cfg.PrefixColor), c.cfg.Command, c.cmd.ProcessState)
	return err
}
