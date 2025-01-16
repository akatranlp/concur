package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

var colorMap = map[string]string{
	"black":     "30",
	"red":       "31",
	"green":     "32",
	"yellow":    "33",
	"blue":      "34",
	"magenta":   "35",
	"cyan":      "36",
	"white":     "37",
	"HiBlack":   "90",
	"HiRed":     "91",
	"HiGreen":   "92",
	"HiYellow":  "93",
	"HiBlue":    "94",
	"HiMagenta": "95",
	"HiCyan":    "96",
	"HiWhite":   "97",
}

type color struct {
	segments string
}

func (c color) Validate() error {
	return nil
}

func (c *color) SetString(s string) error {
	if v, err := strconv.Atoi(s); err == nil {
		return c.SetInt(v)
	}

	if len(s) == 0 {
		return fmt.Errorf("empty color")
	}

	if s[0] == '#' {
		return c.setHex(s)
	}

	color, ok := colorMap[strings.ToLower(s)]
	if !ok {
		return fmt.Errorf("invalid color: %s", s)
	}
	c.segments = color
	return nil
}

func (c *color) SetInt(i int) error {
	if i < 0 || i > 255 {
		return fmt.Errorf("invalid color: %d", i)
	}
	c.segments = fmt.Sprintf("38;5;%d", i)
	return nil
}

func (c *color) setHex(s string) error {
	if len(s) != 7 {
		return fmt.Errorf("invalid hex color: %s", s)
	}

	r, err := strconv.ParseInt(s[1:3], 16, 64)
	if err != nil {
		return fmt.Errorf("invalid hex color: %s", s)
	}
	g, err := strconv.ParseInt(s[3:5], 16, 64)
	if err != nil {
		return fmt.Errorf("invalid hex color: %s", s)
	}
	b, err := strconv.ParseInt(s[5:7], 16, 64)
	if err != nil {
		return fmt.Errorf("invalid hex color: %s", s)
	}

	c.segments = fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
	return nil
}

// Satisfy the flag package  Value interface.
func (c *color) Set(s string) error {
	return c.SetString(s)
}

// Satisfy the pflag package Value interface.
func (c *color) Type() string { return "color" }

// Satisfy the encoding.TextUnmarshaler interface.
func (c *color) UnmarshalText(text []byte) error {
	return c.Set(string(text))
}

// Satisfy the flag package Getter interface.
func (c *color) Get() interface{} { return color(*c) }

type sequence struct {
	Color     color
	Bold      bool
	Underline bool
}

func (c sequence) Validate() error {
	return c.Color.Validate()
}

func (c sequence) Apply(str string) string {
	sequence := c.Color.segments
	if c.Bold {
		sequence += ";1"
	}
	if c.Underline {
		sequence += ";4"
	}
	return fmt.Sprintf("\033[%sm%s\033[0m", sequence, str)
}
