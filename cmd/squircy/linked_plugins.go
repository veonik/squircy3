// +build linked_plugins

package main

import (
	"code.dopame.me/veonik/squircy3/plugin"

	babel "code.dopame.me/veonik/squircy3/plugins/babel"
	node_compat2 "code.dopame.me/veonik/squircy3/plugins/node_compat"
	script "code.dopame.me/veonik/squircy3/plugins/script"
	squircy2_compat "code.dopame.me/veonik/squircy3/plugins/squircy2_compat"
)

var linkedPlugins = []plugin.Initializer{
	plugin.InitializerFunc(babel.Initialize),
	plugin.InitializerFunc(node_compat2.Initialize),
	plugin.InitializerFunc(script.Initialize),
	plugin.InitializerFunc(squircy2_compat.Initialize)}
