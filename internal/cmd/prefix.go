package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const timeFormat = "15:04:05.000"

type Prefix struct {
	template *template.Template
	input    string
	Index    int
	Name     string
	Command  string
	Pid      int
	Time     string
	Padding  string
}

var templateRegex = regexp.MustCompile(`\{\{.*\}\}`)

func NewPrefix(input string) (p Prefix, err error) {
	p.input = input
	switch input {
	case "idx", "index", "name", "", "pid", "time":
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

func (p Prefix) Apply(seq *Sequence) string {
	var prefix string

	if p.template != nil {
		p.Time = time.Now().Format(timeFormat)
		var buf strings.Builder
		if err := p.template.Execute(&buf, p); err != nil {
			panic(err)
		}
		prefix = fmt.Sprintf("[%s]", buf.String())
	} else {
		switch p.input {
		case "":
			if p.Name != "" {
				prefix = p.Name
			} else {
				prefix = strconv.Itoa(p.Index)
			}
		case "idx", "index":
			prefix = strconv.Itoa(p.Index)
		case "name":
			if p.Name != "" {
				prefix = p.Name
			} else {
				prefix = p.Command
			}
		case "pid":
			prefix = strconv.Itoa(p.Pid)
		case "time":
			prefix = time.Now().Format(timeFormat)
		default:
			panic("unreachable")
		}

		prefix = fmt.Sprintf("[%s%s]", prefix, p.Padding)
	}

	if seq == nil {
		return prefix + " "
	}
	return seq.Apply(prefix) + " "
}

var paddingRegex = regexp.MustCompile(`\{\{.*\.Padding.*\}\}`)

func ApplyEvenPadding(cmds ...*Command) {
	var maxLength int
	for _, c := range cmds {
		if !paddingRegex.MatchString(c.prefix.input) && c.prefix.template != nil {
			c.prefix.input += "{{.Padding}}"
			c.prefix.template, _ = template.New("prefix").Parse(c.prefix.input)
		}
		prefix := c.prefix.Apply(nil)
		maxLength = max(maxLength, len(prefix))
	}

	for _, c := range cmds {
		prefix := c.prefix.Apply(nil)
		padding := maxLength - len(prefix)
		if padding > 0 {
			c.prefix.Padding = fmt.Sprintf("%*s", padding, " ")
		}
	}
}
