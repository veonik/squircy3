package main

import (
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/plugins/script"
)

func main() {
	plugin.Main(script.PluginName)
}

// Initialize initializes the script plugin.
func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	return script.Initialize(m)
}
