package main

import (
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/plugins/squircy2_compat"
)

func main() {
	plugin.Main(squircy2_compat.PluginName)
}

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	return squircy2_compat.Initialize(m)
}
