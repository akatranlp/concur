package cmd

import "fmt"

type prefixColor string

const (
	ColorTransparent prefixColor = ""
	ColorRed         prefixColor = "red"
	ColorGreen       prefixColor = "green"
	ColorYellow      prefixColor = "yellow"
)

func ColorizeString(color prefixColor, str string) string {
	switch color {
	case ColorTransparent:
		return str
	case ColorRed:
		return fmt.Sprintf("\033[31m%s\033[0m", str)
	case ColorGreen:
		return fmt.Sprintf("\033[32m%s\033[0m", str)
	case ColorYellow:
		return fmt.Sprintf("\033[33m%s\033[0m", str)
	default:
		return fmt.Sprintf("INVALID COLOR: %s", str)
	}
}
