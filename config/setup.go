package config

import (
	"fmt"

	"github.com/pkg/errors"
)

// A SetupOption is a function that modifies the given Setup in some way.
type SetupOption func(c *Setup) error

type protoFunc func() Value

// Setup is a container struct with information on how to setup a given Config.
type Setup struct {
	name      string
	prototype protoFunc
	singleton bool
	initial   Value
	config    Config

	parent *Setup

	raw map[string]interface{}

	sections map[string]*Setup
	options  map[string]bool
	inherits map[string]struct{}
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
		options:   make(map[string]bool),
		inherits:  make(map[string]struct{}),
	}
}

// apply calls each SetupOption, halting on the first error encountered.
func (s *Setup) apply(options ...SetupOption) error {
	for _, o := range options {
		if err := o(s); err != nil {
			return err
		}
	}
	return nil
}

// validate checks that all required options are set, recursively.
func (s *Setup) validate() error {
	if s.config == nil {
		return errors.New(`expected config to be populated, found nil`)
	}
	for o, reqd := range s.options {
		if reqd {
			var nilValue Value
			v, ok := s.config.Get(o)
			if !ok || v == nil || v == nilValue {
				return errors.Errorf(`required option "%s" is empty`, o)
			}
			if vs, ok := v.(string); ok && len(vs) == 0 {
				return errors.Errorf(`required option "%s" is empty`, o)
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
			return errors.WithMessage(err, fmt.Sprintf("section %s", s.name))
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
	for si := range s.inherits {
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
	for _, ns := range s.sections {
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
