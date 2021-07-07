package main

import (
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/plugins/node_compat"
)

func main() {
	plugin.Main(node_compat.PluginName)
}

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	return node_compat.Initialize(m)
}
