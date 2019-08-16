package squircy2_compat

import (
	"github.com/pkg/errors"
	"net/http"

	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/plugins/squircy2_compat/data"
	"code.dopame.me/veonik/squircy3/vm"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"
)

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	im, err := irc.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrap(err, "squircy2_compat: required dependency missing (irc)")
	}
	ev, err := event.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrap(err, "squircy2_compat: required dependency missing (event)")
	}
	v, err := vm.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrap(err, "squircy2_compat: required dependency missing (vm)")
	}
	return &shimPlugin{
		events: ev,
		vm: v,
		irc: ircHelper{im},
		http: httpHelper{
			Client: &http.Client{Transport: &http.Transport{}},
		},
	}, nil
}

type shimPlugin struct {
	events *event.Dispatcher
	vm     *vm.VM

	db *data.DB

	http httpHelper
	file fileHelper
	conf configHelper
	irc  ircHelper
}

func (p *shimPlugin) Configure(c config.Config) error {
	ef, _ := c.Bool("enable_file_api")
	pf, _ := c.String("file_api_root")
	on, _ := c.String("owner_nick")
	oh, _ := c.String("owner_host")
	da, _ := c.String("data_path")
	p.file = fileHelper{ef, pf}
	p.conf = configHelper{on, oh}
	p.db = data.NewDatabaseConnection(da, logrus.StandardLogger())
	return nil
}

func (p *shimPlugin) Options() []config.SetupOption {
	return []config.SetupOption{}
}

func (p *shimPlugin) Name() string {
	return "squircy2_compat"
}

func (p *shimPlugin) PrependRuntimeInitHandler() bool {
	return true
}

func (p *shimPlugin) HandleRuntimeInit(gr *goja.Runtime) {
	p.initRuntime(gr)
}
