package main

import (
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/plugins/discord"
)

func main() {
	plugin.Main(discord.PluginName)
}

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	return discord.Initialize(m)
}
