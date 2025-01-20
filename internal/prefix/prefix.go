package prefix

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/akatranlp/concur/internal/config"
)

var templateRegex = regexp.MustCompile(`\{\{.*\}\}`)
var paddingRegex = regexp.MustCompile(`\{\{.*\.Padding.*\}\}`)
var timeRegex = regexp.MustCompile(`\{\{.*\.Time.*\}\}`)
var timeSinceStart = time.Now()

type Prefix struct {
	template         *template.Template
	input            string
	maxCommandLength int
	timeFormat       string
	timesince        bool
	isTimed          bool
	data             []*PrefixData
}

type PrefixData struct {
	sequence *config.Sequence
	Index    int
	Name     string
	Command  string
	Pid      int
	Time     string
	Padding  string
	cache    string
}

func NewPrefix(cfg config.PrefixConfig) (*Prefix, error) {
	p := &Prefix{
		timeFormat:       cfg.TimestampFormat,
		timesince:        cfg.TimeSinceStart,
		maxCommandLength: cfg.PrefixLength,
		input:            cfg.Template,
	}

	input := cfg.Template
	switch input {
	case "time":
		p.isTimed = true
		fallthrough
	case "idx", "index", "name", "", "pid", "command":
		return p, nil
	}

	if !templateRegex.MatchString(input) {
		return nil, fmt.Errorf("invalid prefix template: %s", input)
	}
	template, err := template.New("prefix").Parse(input)
	if err != nil {
		return nil, err
	}
	var data PrefixData
	if err := template.Execute(io.Discard, data); err != nil {
		return nil, err
	}
	if timeRegex.MatchString(input) {
		p.isTimed = true
	}
	p.template = template
	return p, nil
}

func (p *Prefix) Add(name, command string, pid int, seq *config.Sequence) int {
	idx := len(p.data)
	p.data = append(p.data, &PrefixData{
		Index:    idx,
		Name:     name,
		Command:  command,
		Pid:      pid,
		sequence: seq,
	})
	return idx
}

func (p *Prefix) Render(idx int, withColor bool) string {
	if idx < 0 || idx >= len(p.data) {
		panic("invalid index")
	}

	data := p.data[idx]
	if data.cache != "" {
		return data.cache
	}

	var prefix string

	if p.template != nil {
		if p.timesince {
			data.Time = time.Since(timeSinceStart).Round(time.Millisecond).String()
		} else {
			data.Time = time.Now().Format(p.timeFormat)
		}
		var buf strings.Builder
		if err := p.template.Execute(&buf, data); err != nil {
			panic(err)
		}
		prefix = fmt.Sprintf("[%s]", buf.String())
	} else {
		switch p.input {
		case "":
			if data.Name != "" {
				prefix = data.Name
			} else {
				prefix = strconv.Itoa(data.Index)
			}
		case "idx", "index":
			prefix = strconv.Itoa(data.Index)
		case "name":
			if data.Name != "" {
				prefix = data.Name
			} else {
				if len(data.Command) > p.maxCommandLength {
					prefix = data.Command[:p.maxCommandLength]
				} else {
					prefix = data.Command
				}
			}
		case "command":
			if len(data.Command) > p.maxCommandLength {
				prefix = data.Command[:p.maxCommandLength]
			} else {
				prefix = data.Command
			}
		case "pid":
			prefix = strconv.Itoa(data.Pid)
		case "time":
			if p.timesince {
				prefix = time.Since(timeSinceStart).Round(time.Millisecond).String()
			} else {
				prefix = time.Now().Format(p.timeFormat)
			}
		default:
			panic("unreachable")
		}

		prefix = fmt.Sprintf("[%s%s]", prefix, data.Padding)
	}

	if data.sequence == nil || !withColor {
		prefix += " "
	} else {
		prefix = data.sequence.Apply(prefix) + " "
	}

	if !p.isTimed {
		data.cache = prefix
	}
	return prefix
}

func (p *Prefix) ApplyEvenPadding() {
	var maxLength int

	if !paddingRegex.MatchString(p.input) && p.template != nil {
		p.input += "{{.Padding}}"
		p.template, _ = template.New("prefix").Parse(p.input)
	}

	for i := range p.data {
		prefix := p.Render(i, false)
		maxLength = max(maxLength, len(prefix))
	}

	for i, data := range p.data {
		prefix := p.Render(i, false)
		padding := maxLength - len(prefix)
		if padding > 0 {
			data.Padding = fmt.Sprintf("%*s", padding, " ")
		}
		data.cache = ""
	}
}
