package vm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Registry struct {
	basePath  string
	modules   map[string]*Module
	main      *Module
	Transform func(in string) (string, error)
}

func NewRegistry(basePath string) *Registry {
	if filepath.Base(basePath) == "node_modules" {
		basePath = filepath.Dir(basePath)
	}
	r := &Registry{basePath: basePath, modules: make(map[string]*Module)}
	r.main = &Module{
		Name: ".",
		Path: r.basePath,
		root: &Module{
			Name:     "node_modules",
			Path:     filepath.Join(r.basePath, "node_modules"),
			registry: r,
		},
		registry: r,
	}
	return r
}

func (r *Registry) reset() {
	for _, m := range r.modules {
		// clear the evaluated values
		m.value = nil
	}
}

func (r *Registry) Enable(runtime *goja.Runtime) {
	r.reset()
	runtime.Set("require", require(runtime, r.main, nil))
}

func require(runtime *goja.Runtime, parent *Module, stack []string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 1 {
			panic(runtime.NewGoError(errors.New("require expects exactly one argument")))
		}
		v := call.Argument(0).String()
		if len(v) == 0 {
			panic(runtime.NewGoError(errors.New("argument cannot be blank")))
		}
		var err error
		module, err := parent.Require(v)
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		if module.value != nil {
			return module.value
		}
		logrus.Warnln("requiring", module.FullPath())
		body := module.Body
		p, err := parser.ParseFile(nil, module.FullPath(), "var res = (function(require, module, exports) {\n"+body+"\n}); res", parser.Mode(0))
		if err != nil {
			if parent.registry.Transform != nil {
				body, err = parent.registry.Transform(body)
				if err == nil {
					p, err = parser.ParseFile(nil, module.FullPath(), "var res = (function(require, module, exports) {\n"+body+"\n}); res", parser.Mode(0))
				}
			}
		}
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		prog, err := goja.CompileAST(p, true)
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		res, err := runtime.RunProgram(prog)
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		cb, ok := goja.AssertFunction(res)
		if !ok {
			panic(errors.New("expected function!"))
		}
		pk := module.FullPath()
		for _, m := range stack {
			if m == pk {
				panic(runtime.NewGoError(errors.Errorf("loop detected, %s is already being required", pk)))
			}
		}
		stack = append(stack, pk)
		defer func() {
			stack = stack[:len(stack)-1]
		}()
		req := runtime.ToValue(require(runtime, module, stack))
		mod := runtime.NewObject()
		err = mod.Set("exports", runtime.NewObject())
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		_, err = cb(nil, req, mod, mod.Get("exports"))
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		module.value = mod.Get("exports")
		return module.value
	}
}

type Module struct {
	Name string
	Path string
	Main string
	Body string

	root     *Module
	registry *Registry

	value goja.Value
}

func (m *Module) FullPath() string {
	return filepath.Clean(filepath.Join(m.Path, m.Name))
}

func (m *Module) String() string {
	return fmt.Sprintf("%s/%s (has body? %v has value? %v)", m.Path, m.Main, len(m.Body) > 0, m.value != nil)
}

func (m *Module) Require(name string) (*Module, error) {
	if strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../") {
		p := filepath.Clean(filepath.Join(m.Path, name))
		if !strings.HasSuffix(p, ".js") {
			p = p + ".js"
		}
		if mo, ok := m.registry.modules[p]; ok {
			return mo, nil
		}
		b, err := ioutil.ReadFile(p)
		if err != nil {
			return nil, err
		}
		body := string(b)
		mo := &Module{Name: name, Path: filepath.Dir(p), Main: filepath.Base(p), Body: body, root: m, registry: m.registry}
		m.registry.modules[p] = mo
		return mo, nil
	} else {
		if m.root != nil {
			return m.root.Require(name)
		}
		p := filepath.Clean(filepath.Join(m.Path, name))
		mod := &Module{Name: name, Path: p, root: m, registry: m.registry}
		if !strings.HasSuffix(p, ".js") {
			b, err := ioutil.ReadFile(filepath.Join(p, "package.json"))
			if err == nil {
				err = json.Unmarshal(b, &mod)
				if err != nil {
					return nil, errors.Wrapf(err, "unable to require %s", name)
				}
			} else {
				if info, err := os.Stat(p); err == nil {
					if info.IsDir() {
						mod.Path = p
						mod.Main = "index.js"
					}
				} else if os.IsNotExist(err) {
					p = p + ".js"
					mod.Path = filepath.Dir(p)
					mod.Main = filepath.Base(p)
				}
			}
		} else {
			mod.Path = filepath.Dir(p)
			mod.Main = filepath.Base(p)
		}
		return mod.Require("./" + mod.Main)
	}
}
