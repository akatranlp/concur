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
	URL        *url.URL
	StatusCode int
	Body       map[string]interface{}
}

var bodyRegex = regexp.MustCompile(`{{.*\.Body.*}}`)

func NewHTTPHealthChecker(u, t string, interval time.Duration) (*HTTPHealthChecker, error) {
	url, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	template, err := template.New("http").Parse(t)
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

func (c *HTTPHealthChecker) runCommand(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.interval/2)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.url.String(), nil)
	if err != nil {
		panic("unreachable")
	}

	var messages []string
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.messages = []string{err.Error()}
		return err
	}

	defer res.Body.Close()

	var body map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		c.messages = []string{err.Error()}
		return err
	}

	data := HTTPHealthCheckData{
		URL:        c.url,
		StatusCode: res.StatusCode,
		Body:       body,
	}
	var buf bytes.Buffer
	if err := c.template.Execute(&buf, data); err != nil {
		c.messages = []string{err.Error()}
		return err
	}

	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		text := scanner.Text()
		messages = append(messages, text)
	}

	c.messages = messages
	return scanner.Err()
}
