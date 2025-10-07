package maps

import (
	"strings"

	rcon "github.com/floriansw/go-hll-rcon/rconv2"
)

func NoOffensive() rcon.MapFilter {
	return func(idx int, name string, _ []string) bool {
		return !(strings.Contains(name, "offensive") || strings.Contains(name, "off"))
	}
}

func Contains(s string) rcon.MapFilter {
	return func(idx int, name string, _ []string) bool {
		return strings.Contains(name, s)
	}
}

func Limit(limit int) rcon.MapFilter {
	return func(idx int, name string, res []string) bool {
		return len(res) < limit
	}
}
