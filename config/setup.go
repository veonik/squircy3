package config

import (
	"github.com/pkg/errors"
)

// A SetupOption is a function that modifies the given Setup in some way.
type SetupOption func(c *Setup) error

// A postSetupOption is a SetupOption that runs after all other SetupOptions.
// SetupOptions that populate the config from a data source (ie. options with
// method name like "WithValuesFrom*") are examples of postSetupOptions. This
// allows postSetupOptions to consume the metadata stored within the Setup
// while populating values from the data source.
type postSetupOption func(c *Setup) error

// A protoFunc is a function that returns the initial value for a config.
type protoFunc func() Value

// An optionFilter modifies the received value and returns the result.
type optionFilter func(name string, val Value) (Value, error)

// An optionValidator returns an error if the given value is invalid.
type optionValidator func(name string, val Value) error

// Setup is a container struct with information on how to setup a given Config.
type Setup struct {
	name      string
	prototype protoFunc
	singleton bool
	initial   Value
	config    Config

	parent *Setup

	raw map[string]interface{}

	sectionsOrdered []*Setup
	sections        map[string]*Setup
	optionsOrdered  []string
	options         map[string][]optionValidator
	filters         map[string][]optionFilter
	inherits        []string

	post []postSetupOption
}

func newSetup(name string, parent *Setup) *Setup {
	return &Setup{
		name:      name,
		prototype: nil,
		singleton: false,
		initial:   nil,
		config:    nil,
		parent:    parent,
		raw:       make(map[string]interface{}),
		sections:  make(map[string]*Setup),
		options:   make(map[string][]optionValidator),
		filters:   make(map[string][]optionFilter),
	}
}

// appendPostSetup adds one or more postSetupOptions to the end.
func (s *Setup) appendPostSetup(options ...postSetupOption) error {
	s.post = append(s.post, options...)
	return nil
}

// prependPostSetup adds one or more postSetupOptions to the beginning.
func (s *Setup) prependPostSetup(options ...postSetupOption) error {
	s.post = append(append([]postSetupOption{}, options...), s.post...)
	return nil
}

// apply calls each SetupOption, halting on the first error encountered.
func (s *Setup) apply(options ...SetupOption) error {
	// clear post options, they will be re-added by the regular options.
	s.post = []postSetupOption{}
	// apply regular options.
	for _, o := range options {
		if err := o(s); err != nil {
			return err
		}
	}
	// apply post-setup options.
	for _, o := range s.post {
		if err := o(s); err != nil {
			return err
		}
	}
	return nil
}

// validate checks that all options and sections are valid, recursively.
func (s *Setup) validate() error {
	if s.config == nil {
		return errors.New(`expected config to be populated, found nil`)
	}
	for o, ovs := range s.options {
		for _, validator := range ovs {
			v, _ := s.config.Get(o)
			if err := validator(o, v); err != nil {
				return errors.Wrapf(err, `failed to validate option "%s"`, o)
			}
		}
	}
	for sn, ss := range s.sections {
		if err := ss.validate(); err != nil {
			return errors.Wrapf(err, `config "%s" contains an invalid section "%s"`, s.name, sn)
		}
	}
	return nil
}

// walkAndWrap populates the Config and all nested sections.
func walkAndWrap(s *Setup) error {
	wrapErr := func(err error) error {
		if s.name != "root" {
			return errors.WithMessage(err, "section "+s.name)
		}
		return err
	}
	if isNil(s.initial) && s.prototype != nil {
		s.initial = s.prototype()
	}
	if isNil(s.initial) && s.parent != nil {
		vo, _ := s.parent.config.Get(s.name)
		s.initial = pointerTo(vo)
	}
	if isNil(s.initial) {
		s.initial = make(map[string]interface{})
	}
	if rc, ok := s.initial.(Config); ok {
		s.config = rc
	} else {
		co, err := newConfigurable(s)
		if err != nil {
			return wrapErr(err)
		}
		s.config = co
	}
	if err := walkInherits(s); err != nil {
		return wrapErr(err)
	}
	if err := walkSections(s); err != nil {
		return wrapErr(err)
	}
	return nil
}

// walkInherits synchronizes inherited options and sections between this and
// the parent.
func walkInherits(s *Setup) error {
	for _, si := range s.inherits {
		if s.parent == nil {
			return errors.Errorf("unable to inherit option %s from non-existent parent", si)
		}
		if sec, ok := s.parent.sections[si]; ok {
			// its a section
			s.config.Set(si, sec.config)
		} else if vo, ok := s.parent.config.Get(si); ok {
			// its an option
			s.config.Set(si, vo)
		} else {
			return errors.Errorf("unable inherit non-existent option %s from parent %s", si, s.parent.name)
		}
	}
	return nil
}

// walkSections walks through each section, populating a Config for each.
func walkSections(s *Setup) error {
	for _, ns := range s.sectionsOrdered {
		if v, ok := s.raw[ns.name].(map[string]interface{}); ok {
			ns.raw = v
		}
		if err := walkAndWrap(ns); err != nil {
			return err
		}
		s.config.Set(ns.name, ns.config)
	}
	return nil
}
