package vm

import (
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

// A Registry provides basic commonjs-compatible facilities for a VM.
type Registry struct {
	basePath  string
	modules   map[string]*Module
	main      *Module
	Transform func(in string) (string, error)
}

// NewRegistry creates a new registry with the given base path.
// A Registry is designed to provide NodeJS type require() functions to goja.
func NewRegistry(basePath string) *Registry {
	if filepath.Base(basePath) == "node_modules" {
		// use the path right above node_modules.
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

func (r *Registry) Modules() []string {
	var res []string
	for k := range r.modules {
		res = append(res, k)
	}
	return res
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
	if err := v.Set("Modules", r.Modules); err != nil {
		logrus.Warnln("registry: error initializing runtime:", err)
	}
	runtime.Set("Registry", v)
}

// SetModule adds the Module to the Registry.
func (r *Registry) SetModule(module *Module) {
	module.registry = r
	module.root = r.main.root
	r.modules[module.Name] = module
}

// require performs the actual execution of required modules and files.
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
			return module.value.Get("exports")
		}
		logrus.Debugln("requiring", module.FullPath())
		parse := func(body string) (*ast.Program, error) {
			return parser.ParseFile(nil, module.FullPath(), "(function(require, module, exports) {\n"+body+"\n})", parser.Mode(0))
		}
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
		module.value = runtime.NewObject()
		err = module.value.Set("exports", runtime.NewObject())
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		_, err = cb(nil, req, module.value, module.value.Get("exports"))
		if err != nil {
			panic(runtime.NewGoError(err))
		}

		return module.value.Get("exports")
	}
}

// A Module is a javascript module identified by a name and full path.
type Module struct {
	Name string
	Path string
	Main string
	Body string

	root     *Module
	registry *Registry

	// value is the evaluated value in the currently running VM.
	value *goja.Object
}

// Require loads the given name within the context of the Module.
// Relative paths are supported, as are implicit index.js requires, and suffix-less requires.
// If the name is a module, its package.json will be parsed to determine which script to execute.
// This method does not evaluate the loaded module, see instead the package-level require function.
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
	return filepath.Clean(filepath.Join(m.Path, m.Main))
}

func (m *Module) String() string {
	return fmt.Sprintf("%s/%s (has body? %v has value? %v)", m.Path, m.Main, len(m.Body) > 0, m.value != nil)
}
