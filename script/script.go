package script // import "code.dopame.me/veonik/squircy3/script"

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"code.dopame.me/veonik/squircy3/vm"
)

type Script struct {
	Name string
	Body string
}

type Manager struct {
	rootDir string
}

func (m *Manager) RunAll(vm *vm.VM) error {
	ss, err := m.LoadAll()
	if err != nil {
		return err
	}
	for _, s := range ss {
		_, err = vm.RunScript(s.Name, s.Body).Await()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) LoadAll() ([]Script, error) {
	fs, err := ioutil.ReadDir(m.rootDir)
	if err != nil {
		return nil, err
	}
	var res []Script
	for _, f := range fs {
		if f.IsDir() {
			continue
		}
		n := f.Name()
		if !strings.HasSuffix(n, ".js") {
			continue
		}
		b, err := ioutil.ReadFile(filepath.Join(m.rootDir, n))
		if err != nil {
			return nil, err
		}
		res = append(res, Script{Name: n, Body: string(b)})
	}
	return res, nil
}
