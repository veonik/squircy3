package config

import (
	"flag"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// A Section describes the pre-configured state of a nested configuration
// section.
type Section interface {
	// Name is used as the name of the section.
	Name() string
	// Prototype returns the zero value for the section.
	Prototype() Value
	// Singleton is true if the section may only exist once.
	Singleton() bool
}

// section is a default implementation of a Section.
type section struct {
	name      string
	prototype protoFunc
	singleton bool
}

func (sec section) Name() string {
	return sec.name
}

func (sec section) Prototype() Value {
	if sec.prototype != nil {
		return sec.prototype()
	}
	return nil
}

func (sec section) Singleton() bool {
	return sec.singleton
}

// WithGenericSection will add a basic section with the given name and options.
func WithGenericSection(name string, options ...SetupOption) SetupOption {
	return WithSection(&section{name: name}, options...)
}

// WithSection will add a Section with the given options.
func WithSection(sec Section, options ...SetupOption) SetupOption {
	return func(s *Setup) error {
		n := sec.Name()
		if _, ok := s.sections[n]; ok {
			return errors.Errorf(`section "%s" already exists`, n)
		}
		opts := make([]SetupOption, len(options))
		copy(opts, options)
		if sec.Singleton() {
			opts = append(opts, WithSingleton(true))
		}
		// Don't bother settings prototypes that return nil.
		if pr := sec.Prototype(); !isNil(pr) {
			opts = append(opts, WithInitPrototype(sec.Prototype))
		}
		ns := newSetup(n, s)
		if err := ns.apply(opts...); err != nil {
			return err
		}
		s.sections[ns.name] = ns
		return nil
	}
}

// WithSingleton will enable or disable a section's singleton property.
func WithSingleton(singleton bool) SetupOption {
	return func(s *Setup) error {
		s.singleton = singleton
		return nil
	}
}

// WithInitValue uses the given Value as the starting point for the section.
// Initial values are updated via reflection and kept in sync with changes made
// to the Config.
func WithInitValue(value Value) SetupOption {
	return func(s *Setup) error {
		s.prototype = nil
		s.initial = value
		return nil
	}
}

// WithInitPrototype sets the given func as the prototype.
// The prototype func will be invoked and its return value will be used to
// populate the initial value in the Config.
func WithInitPrototype(proto func() Value) SetupOption {
	return func(c *Setup) error {
		c.initial = nil
		c.prototype = proto
		return nil
	}
}

// WithOption adds an optional option to the Config.
func WithOption(name string) SetupOption {
	return WithOptions(name)
}

// WithOptions adds multiple optional options to the Config.
func WithOptions(names ...string) SetupOption {
	return func(s *Setup) error {
		for _, n := range names {
			s.options[n] = false
		}
		return nil
	}
}

// WithRequiredOption adds a required option to the Config.
func WithRequiredOption(name string) SetupOption {
	return WithRequiredOptions(name)
}

// WithRequiredOptions adds multiple required options to the Config.
func WithRequiredOptions(names ...string) SetupOption {
	return func(s *Setup) error {
		for _, n := range names {
			s.options[n] = true
		}
		return nil
	}
}

// WithInheritedOption will inherit an option from the parent Config.
func WithInheritedOption(name string) SetupOption {
	return func(s *Setup) error {
		ps := s.parent
		if ps == nil {
			return errors.Errorf("config: unable to inherit option '%s' for section %s; no parent found", name, s.name)
		}
		s.inherits[name] = struct{}{}
		return nil
	}
}

// WithInheritedSection will inherit a section from the parent Config.
func WithInheritedSection(name string) SetupOption {
	return WithInheritedOption(name)
}

// WithValuesFromTOMLFile will populate the Config with values parsed from a
// TOML file.
func WithValuesFromTOMLFile(filename string) SetupOption {
	return func(s *Setup) error {
		if s.raw == nil {
			s.raw = make(map[string]interface{})
		}
		if _, err := toml.DecodeFile(filename, &s.raw); err != nil {
			return err
		}
		return nil
	}
}

type flagMapper struct {
	s *Setup
}

func newFlagMapper(s *Setup) *flagMapper {
	return &flagMapper{s}
}

func (fm *flagMapper) normalize(name string) string {
	return strings.ToLower(
		strings.ReplaceAll(name, "-", "_"))
}

// Map converts a flag name into a path based on sections and options.
// If a Config has sections "A" and "B" has an option "c", then the flag
// named "-a-b-c" would be converted into the path ["A","B","c"].
func (fm *flagMapper) Map(flagName string) (path []string) {
	normal := fm.normalize(flagName)
	s := fm.s
loop:
	for s != nil {
		// a valueInspector here handles struct tag aliases.
		is, err := inspect(s.initial)
		if err != nil {
			logrus.Debugf("config: unable to create valueInspector for %s: %s", s.name, err)
		} else {
			if _, err := is.Get(normal); err == nil {
				path = append(path, normal)
				s = nil
				goto loop
			} else {
				logrus.Debugf("config: valueInspector returned error for %s in %s: %s", normal, s.name, err)
			}
		}
		// check for a match in options next
		for k := range s.options {
			kn := fm.normalize(k)
			if kn == normal {
				// found it
				path = append(path, k)
				s = nil
				goto loop
			}
		}
		// check for a matching section, using the name as a prefix
		for k, ks := range s.sections {
			kn := fm.normalize(k) + "_"
			if strings.HasPrefix(normal, kn) {
				// found the next step in the path
				normal = strings.Replace(normal, kn, "", 1)
				path = append(path, k)
				s = ks
				goto loop
			}
		}
		return nil
	}
	return path
}

// WithValuesFromFlagSet populates the Config using command-line flags.
func WithValuesFromFlagSet(fs *flag.FlagSet) SetupOption {
	return func(s *Setup) error {
		if !fs.Parsed() {
			return errors.Errorf("given FlagSet must be parsed")
		}
		if s.raw == nil {
			s.raw = make(map[string]interface{})
		}
		m := newFlagMapper(s)
		fs.Visit(func(f *flag.Flag) {
			path := m.Map(f.Name)
			if len(path) == 0 {
				logrus.Debugf("config: did not match anything for flag '%s' for section %s", f.Name, s.name)
				return
			}
			var val interface{} = f.Value.String()
			if fg, ok := f.Value.(flag.Getter); ok {
				val = fg.Get()
			}
			v := s.raw
			i := 0
			for i = 0; i < len(path)-1; i++ {
				if vs, ok := v[path[i]].(map[string]interface{}); ok {
					v = vs
				} else {
					if vr, ok := v[path[i]]; ok {
						logrus.Debugf("config: overriding existing value in raw config for %s -- was type %T", f.Name, vr)
					}
					nv := make(map[string]interface{})
					v[path[i]] = nv
					v = nv
				}
			}
			v[path[i]] = val
		})
		return nil
	}
}
