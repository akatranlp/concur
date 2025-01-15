package cmd

import "fmt"

type prefixColor string

const (
	ColorTransparent prefixColor = ""
	ColorBlack       prefixColor = "black"
	ColorWhite       prefixColor = "white"
	ColorRed         prefixColor = "red"
	ColorGreen       prefixColor = "green"
	ColorYellow      prefixColor = "yellow"
)

const (
	colorBlack  = "\033[30;1m"
	colorWhite  = "\033[37;1m"
	colorRed    = "\033[31;1m"
	colorGreen  = "\033[32;1m"
	colorYellow = "\033[33;1m"
	colorReset  = "\033[0m"
)

func ColorizeString(color prefixColor, str string) string {
	switch color {
	case ColorTransparent:
		return str
	case ColorRed:
		return fmt.Sprintf("%s%s%s", colorRed, str, colorReset)
	case ColorGreen:
		return fmt.Sprintf("%s%s%s", colorGreen, str, colorReset)
	case ColorYellow:
		return fmt.Sprintf("%s%s%s", colorYellow, str, colorReset)
	default:
		return fmt.Sprintf("\033[38;2;255;0;255;48;2;18;18;18;1;4m%s%s", str, colorReset)
	}
}
