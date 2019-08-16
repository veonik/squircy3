package config

import (
	"github.com/pkg/errors"
)

type SetupOption func(c *Setup) error

type protoFunc func() Value

// Setup is a container struct with information on how to setup a given Config.
type Setup struct {
	name      string
	prototype protoFunc
	singleton bool
	initial   Value
	config    Config

	raw map[string]interface{}

	sections map[string]*Setup
	options  map[string]bool
}

func newSetup(name string) *Setup {
	return &Setup{
		name:      name,
		prototype: nil,
		singleton: false,
		initial:   nil,
		config:    nil,
		raw:       make(map[string]interface{}),
		sections:  make(map[string]*Setup),
		options:   make(map[string]bool),
	}
}

func (s *Setup) apply(options ...SetupOption) error {
	for _, o := range options {
		if err := o(s); err != nil {
			return err
		}
	}
	return nil
}

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

func walkAndWrap(s *Setup) error {
	if isNil(s.initial) && s.prototype != nil {
		s.initial = s.prototype()
	}
	if isNil(s.initial) {
		s.initial = make(map[string]interface{})
	}
	if rc, ok := s.initial.(Config); ok {
		s.config = rc
	} else {
		co, err := newConfigurable(s)
		if err != nil {
			return err
		}
		s.config = co
	}
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
