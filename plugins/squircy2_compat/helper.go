package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"code.dopame.me/veonik/squircy3/plugins/squircy2_compat/data"
	"code.dopame.me/veonik/squircy3/vm"
)

type Config struct {
	EnableFileAPI bool   `toml:"enable_file_api"`
	FileAPIPath   string `toml:"file_api_root"`
	OwnerNick     string `toml:"owner_nick"`
	OwnerHost     string `toml:"owner_host"`
	DataPath      string `toml:"data_path"`

	RootDir string `flag:"root_path"`
}

type HelperSet struct {
	*Config

	events *event.Dispatcher
	vm     *vm.VM

	db *data.DB

	http httpHelper
	file fileHelper
	conf configHelper
	irc  ircHelper

	funcs map[string]*callback
}

func NewHelperSet(e *event.Dispatcher, v *vm.VM, i *irc.Manager) *HelperSet {
	return &HelperSet{
		events: e,
		vm:     v,
		irc:    ircHelper{i},
		http: httpHelper{
			Client: &http.Client{Transport: &http.Transport{}},
		},
	}
}

func (p *HelperSet) Configure(c Config) error {
	if !filepath.IsAbs(c.FileAPIPath) {
		c.FileAPIPath = filepath.Join(c.RootDir, c.FileAPIPath)
	}
	if c.EnableFileAPI {
		// only need to bother with this check if it is indeed enabled
		if _, err := os.Stat(c.FileAPIPath); os.IsNotExist(err) {
			logrus.Warnf("squircy2_compat: file_api_path '%s' does not exist, disabling file api", c.FileAPIPath)
			c.EnableFileAPI = false
		}
	}
	if !filepath.IsAbs(c.DataPath) {
		c.DataPath = filepath.Join(c.RootDir, c.DataPath)
	}
	if _, err := os.Stat(c.DataPath); os.IsNotExist(err) {
		logrus.Warnf("squircy2_compat: data_path '%s' does not exist, creating", c.DataPath)
		if err := os.MkdirAll(c.DataPath, 0755); err != nil {
			logrus.Warnf("squircy2_compat: db functions may not work, failed to create data_path '%s': %s", c.DataPath, err)
		}
	}
	p.Config = &c
	p.file = fileHelper{c.EnableFileAPI, c.FileAPIPath}
	p.conf = configHelper{c.OwnerNick, c.OwnerHost}
	p.db = data.NewDatabaseConnection(c.DataPath, logrus.StandardLogger())
	return nil
}

type httpHelper struct {
	*http.Client
}

func (client *httpHelper) Get(uri string, headers ...string) string {
	h := map[string][]string{}
	for _, v := range headers {
		p := strings.Split(v, ":")
		if len(p) != 2 {
			continue
		}
		if _, ok := h[p[0]]; !ok {
			h[p[0]] = make([]string, 0)
		}
		h[p[0]] = append(h[p[0]], p[1])
	}
	req := &http.Request{
		Method: "GET",
		Header: http.Header(h),
	}
	req.URL, _ = url.Parse(uri)
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(b)
}

func (client *httpHelper) Post(uri string, body string, headers ...string) string {
	h := map[string][]string{}
	for _, v := range headers {
		p := strings.Split(v, ":")
		if len(p) != 2 {
			continue
		}
		if _, ok := h[p[0]]; !ok {
			h[p[0]] = make([]string, 0)
		}
		h[p[0]] = append(h[p[0]], p[1])
	}
	req := &http.Request{
		Method:        "POST",
		Body:          ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:        http.Header(h),
		ContentLength: int64(len(body)),
	}
	req.URL, _ = url.Parse(uri)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return string(b)
}

type configHelper struct {
	ownerNick string
	ownerHost string
}

func (h *configHelper) OwnerNick() string {
	return h.ownerNick
}

func (h *configHelper) OwnerHost() string {
	return h.ownerHost
}

type ircHelper struct {
	manager *irc.Manager
}

func (h *ircHelper) Connect() error {
	return h.manager.Connect()
}

func (h *ircHelper) Disconnect() error {
	return h.manager.Disconnect()
}

func (h *ircHelper) Privmsg(target, message string) error {
	return h.manager.Do(func(conn *irc.Connection) error {
		conn.Privmsg(target, message)
		return nil
	})
}

func (h *ircHelper) Action(target, message string) error {
	return h.manager.Do(func(conn *irc.Connection) error {
		conn.Action(target, message)
		return nil
	})
}

func (h *ircHelper) Join(target string) error {
	return h.manager.Do(func(conn *irc.Connection) error {
		conn.Join(target)
		return nil
	})
}

func (h *ircHelper) Part(target string) error {
	return h.manager.Do(func(conn *irc.Connection) error {
		conn.Part(target)
		return nil
	})
}

func (h *ircHelper) CurrentNick() (string, error) {
	var res string
	err := h.manager.Do(func(conn *irc.Connection) error {
		res = conn.GetNick()
		return nil
	})
	return res, err
}

func (h *ircHelper) Nick(newNick string) error {
	return h.manager.Do(func(conn *irc.Connection) error {
		conn.Nick(newNick)
		return nil
	})
}

func (h *ircHelper) Raw(command string) error {
	return h.manager.Do(func(conn *irc.Connection) error {
		conn.SendRaw(command)
		return nil
	})
}

type fileHelper struct {
	EnableFileAPI bool
	FileAPIRoot   string
}

func (h *fileHelper) ReadAll(name string) (string, error) {
	if !h.EnableFileAPI {
		return "", errors.New("file: file api is disabled")
	}
	p := filepath.Clean(filepath.Join(h.FileAPIRoot, name))
	if !strings.HasPrefix(p, h.FileAPIRoot) {
		return "", fmt.Errorf("file: path does not exist within configured root: %s", p)
	}
	res, err := ioutil.ReadFile(p)
	return string(res), errors.Wrapf(err, "file: failed to read file: %s", p)
}
