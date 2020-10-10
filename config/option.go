package config

import (
	"github.com/pkg/errors"
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
// Alias for WithInheritedOption.
func WithInheritedSection(name string) SetupOption {
	return WithInheritedOption(name)
}
