package cmd

import (
	"fmt"
	"strings"
	"syscall"
)

type KillSignal syscall.Signal

func (c KillSignal) Validate() error {
	return nil
}

func (c KillSignal) Sys() syscall.Signal {
	return syscall.Signal(c)
}

// Satisfy the flag package  Value interface.
func (s *KillSignal) Set(str string) error {
	if strings.ToUpper(str) == "SIGINT" {
		*s = KillSignal(syscall.SIGINT)
	} else if strings.ToUpper(str) == "SIGTERM" {
		*s = KillSignal(syscall.SIGTERM)
	} else if strings.ToUpper(str) == "SIGKILL" {
		*s = KillSignal(syscall.SIGKILL)
	} else {
		return fmt.Errorf("invalid kill signal: %s", str)
	}
	return nil
}

// Satisfy the pflag package Value interface.
func (s *KillSignal) Type() string { return "kill-signal" }

// Satisfy the encoding.TextUnmarshaler interface.
func (s *KillSignal) UnmarshalText(text []byte) error {
	return s.Set(string(text))
}

// Satisfy the flag package Getter interface.
func (s *KillSignal) Get() interface{} { return KillSignal(*s) }
