package vm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja/ast"
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
	v := runtime.NewObject()
	if err := v.Set("SetModule", r.SetModule); err != nil {
		logrus.Warnln("registry: error initializing runtime:", err)
	}
	runtime.Set("Registry", v)
}

func (r *Registry) SetModule(module *Module) {
	module.registry = r
	r.modules[module.Name] = module
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
		etag := sha256.Sum256([]byte(module.Body))
		equal := func(a, b [sha256.Size]byte) bool {
			for i := 0; i < sha256.Size; i++ {
				if a[i] != b[i] {
					return false
				}
			}
			return true
		}
		parse := func(body string) (*ast.Program, error) {
			return parser.ParseFile(nil, module.FullPath(), "var res = (function(require, module, exports) {\n"+body+"\n}); res", parser.Mode(0))
		}
		if !equal(module.etag, etag) {
			body := module.Body
			p, err := parse(body)
			if err != nil && parent.registry.Transform != nil {
				// try transforming and parsing again after a failure
				body, err = parent.registry.Transform(body)
				if err == nil {
					p, err = parse(body)
				}
			}
			if err != nil {
				panic(runtime.NewGoError(err))
			}
			module.prog, err = goja.CompileAST(p, true)
			if err != nil {
				panic(runtime.NewGoError(err))
			}
			module.etag = etag
		}
		res, err := runtime.RunProgram(module.prog)
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

	etag [sha256.Size]byte
	prog *goja.Program

	root     *Module
	registry *Registry

	value goja.Value
}

func (m *Module) Require(name string) (*Module, error) {
	if strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../") {
		return m.requireRelative(name)
	}
	if m.root != nil {
		return m.root.Require(name)
	}
	if mo, ok := m.registry.modules[name]; ok {
		return mo, nil
	}
	p := filepath.Clean(filepath.Join(m.Path, name))
	mod := &Module{Name: name, Path: p, root: m, registry: m.registry}
	if !strings.HasSuffix(p, ".js") {
		b, err := ioutil.ReadFile(filepath.Join(p, "package.json"))
		if err == nil {
			err = json.Unmarshal(b, &mod)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to read package.json for %s", name)
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
	return mod.requireRelative("./" + mod.Main)
}

func (m *Module) requireRelative(name string) (*Module, error) {
	p := filepath.Clean(filepath.Join(m.Path, name))
	if !strings.HasSuffix(p, ".js") {
		if info, err := os.Stat(p); err == nil {
			if info.IsDir() {
				p = filepath.Join(p, "index.js")
			}
		} else if os.IsNotExist(err) {
			p = p + ".js"
		}
	}
	if mo, ok := m.registry.modules[p]; ok {
		return mo, nil
	}
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to require %s", name)
	}
	body := string(b)
	mo := &Module{Name: name, Path: filepath.Dir(p), Main: filepath.Base(p), Body: body, root: m, registry: m.registry}
	m.registry.modules[p] = mo
	return mo, nil
}

func (m *Module) FullPath() string {
	return filepath.Clean(filepath.Join(m.Path, m.Name))
}

func (m *Module) String() string {
	return fmt.Sprintf("%s/%s (has body? %v has value? %v)", m.Path, m.Main, len(m.Body) > 0, m.value != nil)
}
