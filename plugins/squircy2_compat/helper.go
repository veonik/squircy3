package squircy2_compat

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"code.dopame.me/veonik/squircy3/irc"
)

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
	return string(res), err
}
