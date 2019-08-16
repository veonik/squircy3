package main

import (
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/plugins/squircy2_compat"
)

func main() {
	panic("squircy2_compat v1.0.0")
}

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	return squircy2_compat.Initialize(m)
}