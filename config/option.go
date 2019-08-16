package config

import (
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type section struct {
	name      string
	prototype protoFunc
	singleton bool
}

func (sec section) Prefix() string {
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

func WithGenericSection(name string, options ...SetupOption) SetupOption {
	return WithSection(&section{name: name}, options...)
}

func WithSection(sec Section, options ...SetupOption) SetupOption {
	return func(s *Setup) error {
		n := sec.Prefix()
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
		ns := newSetup(n)
		if err := ns.apply(opts...); err != nil {
			return err
		}
		s.sections[ns.name] = ns
		return nil
	}
}

func WithSingleton(singleton bool) SetupOption {
	return func(s *Setup) error {
		s.singleton = singleton
		return nil
	}
}

func WithInitValue(value Value) SetupOption {
	return func(s *Setup) error {
		s.prototype = nil
		s.initial = value
		return nil
	}
}

func WithInitPrototype(proto func() Value) SetupOption {
	return func(c *Setup) error {
		c.initial = nil
		c.prototype = proto
		return nil
	}
}

func WithOption(name string) SetupOption {
	return WithOptions(name)
}

func WithOptions(names ...string) SetupOption {
	return func(s *Setup) error {
		for _, n := range names {
			s.options[n] = false
		}
		return nil
	}
}

func WithRequiredOption(name string) SetupOption {
	return WithRequiredOptions(name)
}

func WithRequiredOptions(names ...string) SetupOption {
	return func(s *Setup) error {
		for _, n := range names {
			s.options[n] = true
		}
		return nil
	}
}

func WithValuesFromTOMLFile(filename string) SetupOption {
	return func(s *Setup) error {
		s.raw = make(map[string]interface{})
		if _, err := toml.DecodeFile(filename, &s.raw); err != nil {
			return err
		}
		return nil
	}
}
