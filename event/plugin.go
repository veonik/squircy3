package event

import (
	"code.dopame.me/veonik/squircy3/plugin"

	"github.com/pkg/errors"
)

// FromPlugins returns the event plugin's Dispatcher or an error if it fails.
func FromPlugins(m *plugin.Manager) (*Dispatcher, error) {
	plg, err := m.Lookup("event")
	if err != nil {
		return nil, err
	}
	mplg, ok := plg.(*eventPlugin)
	if !ok {
		return nil, errors.Errorf("event: received unexpected plugin type")
	}
	return mplg.dispatcher, nil
}

// Initialize is a plugin.Initializer that initializes an event plugin.
func Initialize(*plugin.Manager) (plugin.Plugin, error) {
	p := &eventPlugin{NewDispatcher()}
	return p, nil
}

type eventPlugin struct {
	dispatcher *Dispatcher
}

func (p *eventPlugin) Name() string {
	return "event"
}

func (p *eventPlugin) HandlePluginInit(o plugin.Plugin) {
	p.dispatcher.Emit("plugin.INIT", map[string]interface{}{"name": o.Name(), "plugin": o})
}

func (p *eventPlugin) HandleShutdown() {
	p.dispatcher.Stop()
}
