package healthcheck

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"text/template"
	"time"
)

type HTTPHealthChecker struct {
	url      *url.URL
	template *template.Template
	interval time.Duration

	messages []string
	lastRows int
}

type HTTPHealthCheckData struct {
	URL        string
	StatusCode int
	Error      string
	Body       string
}

var bodyRegex = regexp.MustCompile(`{{.*\.Body.*}}`)
var tmplFnMap = template.FuncMap{
	"jsonParse": func(body string) map[string]interface{} {
		b := make(map[string]interface{})
		_ = json.Unmarshal([]byte(body), &b)
		return b
	},
}

func NewHTTPHealthChecker(u, t string, interval time.Duration) (*HTTPHealthChecker, error) {
	url, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	template, err := template.New("http").Funcs(tmplFnMap).Parse(t)
	if err != nil {
		return nil, err
	}
	if !bodyRegex.MatchString(t) {
		var data HTTPHealthCheckData
		if err := template.Execute(io.Discard, data); err != nil {
			return nil, err
		}
	}

	return &HTTPHealthChecker{
		url:      url,
		template: template,
		interval: interval,
	}, nil
}

func (c *HTTPHealthChecker) GetHealthCheckMessage(context.Context) (messageRows []string, rows int) {
	newMessages := make([]string, len(c.messages))
	copy(newMessages, c.messages)
	lastRows := c.lastRows
	c.lastRows = len(newMessages)
	return newMessages, lastRows
}

func (c *HTTPHealthChecker) Start(ctx context.Context) {
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

func (c *HTTPHealthChecker) runCommand(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, c.interval/2)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.url.String(), nil)
	if err != nil {
		panic("unreachable")
	}

	data := &HTTPHealthCheckData{
		URL:        c.url.String(),
		Error:      "",
		Body:       "",
		StatusCode: -1,
	}

	defer func() {
		var messages []string
		var buf bytes.Buffer
		if err := c.template.Execute(&buf, data); err != nil {
			panic(err)
		}

		scanner := bufio.NewScanner(&buf)
		for scanner.Scan() {
			text := scanner.Text()
			messages = append(messages, text)
		}

		c.messages = messages
	}()

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		data.Error = err.Error()
		return
	}
	data.StatusCode = res.StatusCode

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		data.Error = err.Error()
		return
	}
	data.Body = string(body)
	return
}
