package color

import (
	"strconv"
	"strings"

	"github.com/mmonterroca/docxgo/v2/domain"
)

type Color = domain.Color

func RGB(r, g, b uint8) Color {
	return Color{R: r, G: g, B: b}
}

func FromHex(value string) Color {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 {
		return Color{}
	}
	r, errR := strconv.ParseUint(value[0:2], 16, 8)
	g, errG := strconv.ParseUint(value[2:4], 16, 8)
	b, errB := strconv.ParseUint(value[4:6], 16, 8)
	if errR != nil || errG != nil || errB != nil {
		return Color{}
	}
	return Color{R: uint8(r), G: uint8(g), B: uint8(b)}
}
