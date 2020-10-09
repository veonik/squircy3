package config

import (
	"fmt"
	"reflect"

	"github.com/fatih/structtag"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// configurable must implement Config.
var _ Config = &configurable{}

type configurableOpts map[string]Value

// configurable is the default Config implementation.
type configurable struct {
	Value
	options   configurableOpts
	inspector *valueInspector
}

func newConfigurable(s *Setup) (*configurable, error) {
	if isNil(s.initial) {
		return nil, errors.New("unable to wrap <nil>")
	}
	is, err := inspect(s.initial)
	if err != nil {
		return nil, err
	}
	c := &configurable{Value: s.initial, options: make(configurableOpts), inspector: is}
	for k, v := range s.raw {
		c.Set(k, v)
	}
	for k := range s.options {
		v, err := c.inspector.Get(k)
		if err != nil {
			return nil, err
		}
		c.Set(k, v)
	}
	return c, nil
}

func (c *configurable) Self() Value {
	return c.Value
}

func (c *configurable) Get(key string) (Value, bool) {
	if v, err := c.inspector.Get(key); err == nil {
		return v, true
	} else {
		if v, ok := c.options[key]; ok {
			return v, true
		}
		logrus.Warnln("error getting value from config:", err)
	}
	return nil, false
}

func (c *configurable) String(key string) (string, bool) {
	v, ok := c.Get(key)
	if !ok {
		return "", false
	}
	if vs, ok := v.(string); ok {
		return vs, true
	}
	return "", false
}

func (c *configurable) Bool(key string) (bool, bool) {
	v, ok := c.Get(key)
	if !ok {
		return false, false
	}
	if vs, ok := v.(bool); ok {
		return vs, true
	}
	return false, false
}

func (c *configurable) Int(key string) (int, bool) {
	v, ok := c.Get(key)
	if !ok {
		return 0, false
	}
	if vs, ok := v.(int); ok {
		return vs, true
	}
	return 0, false
}

func (c *configurable) Set(key string, val Value) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	c.options[key] = val
	c.inspector.Set(key, val)
}

func (c *configurable) Section(key string) (Config, error) {
	v := c.options[key]
	if vv, ok := v.(*configurable); ok {
		return vv, nil
	}
	if s, ok := v.(Config); ok {
		return s, nil
	}
	return nil, errors.Errorf(`section "%s" contains unexpected type %T: %v`, key, v, v)
}

// A valueInspector abstracts the reading and modifying of a Value,
// particularly structs, struct pointers, and map.
type valueInspector struct {
	value reflect.Value
	typ   reflect.Type
	tags  map[string]structField
}

func inspect(v Value) (*valueInspector, error) {
	vo := reflect.ValueOf(v)
	if vo.Kind() == reflect.Ptr {
		vo = reflect.Indirect(vo)
	}
	tags := map[string]structField{}
	if vo.Kind() == reflect.Struct {
		t, f, err := structTags(v)
		if err != nil {
			return nil, err
		}
		for _, fd := range f {
			tags[fd.Name] = fd
		}
		for _, rt := range t {
			tags[rt.Name] = rt.Field
		}
	}
	t := reflect.TypeOf(vo)
	return &valueInspector{
		value: vo,
		typ:   t,
		tags:  tags,
	}, nil
}

func (i *valueInspector) Get(name string) (Value, error) {
	return i.valueNamed(name)
}

func (i *valueInspector) Set(name string, val Value) {
	if vc, ok := val.(*configurable); ok {
		val = vc.Value
	}
	rv := reflect.ValueOf(val)
	if i.value.Kind() == reflect.Map {
		i.value.SetMapIndex(reflect.ValueOf(name), rv)
		return
	}
	var m reflect.Value
	if t, ok := i.tags[name]; ok {
		m = i.value.FieldByIndex(t.Index)
	}
	if !m.CanSet() {
		return
	}
	if m.Kind() == reflect.Slice {
		switch m.Type().Elem().Kind() {
		case reflect.String:
			var res []string
			if v, ok := val.([]interface{}); ok {
				for _, vv := range v {
					if vs, ok := vv.(string); ok {
						res = append(res, vs)
					}
				}
			}
			rv = reflect.ValueOf(res)
		default:
			logrus.Warnf("config: unsupported slice type: %s", i.value.Index(0).Kind())
			return
		}
	}
	trySet(m, rv)
}

func trySet(m reflect.Value, rv reflect.Value) {
	defer func() {
		if v := recover(); v != nil {
			logrus.Debugln("config: failed to set value using reflection:", v)
		}
	}()
	want := m.Kind()
	have := rv.Kind()
	if want != have {
		if want == reflect.Ptr {
			rv = reflect.Indirect(rv)
		} else if have == reflect.Ptr {
			rv = rv.Elem()
		}
	}
	m.Set(rv)
}

func (i *valueInspector) valueNamed(name string) (Value, error) {
	var m reflect.Value
	if i.value.Kind() == reflect.Map {
		m = i.value.MapIndex(reflect.ValueOf(name))
		if !m.IsValid() {
			return nil, nil
		}
	} else {
		if t, ok := i.tags[name]; ok {
			m = i.value.FieldByIndex(t.Index)
		}
		if !m.IsValid() {
			return nil, errors.New("no field with name " + name)
		}
	}
	if m.CanInterface() {
		return m.Interface(), nil
	}
	return m.Elem(), nil
}

type structTag struct {
	Key   string
	Name  string
	Field structField
}

type structField struct {
	Name  string
	Index []int
}

func structTags(v Value) ([]structTag, []structField, error) {
	var tags []structTag
	var fields []structField
	t := reflect.ValueOf(v)
	if t.Kind() == reflect.Ptr {
		t = reflect.Indirect(t)
	}
	if t.Kind() != reflect.Struct {
		return nil, nil, errors.New("value is not a struct or ptr to struct")
	}
	tt := t.Type()
	for i := 0; i < tt.NumField(); i++ {
		f := tt.Field(i)
		ff := structField{f.Name, f.Index}
		fields = append(fields, ff)
		tgs, err := structtag.Parse(string(f.Tag))
		if err != nil {
			return nil, nil, err
		}
		for _, tg := range tgs.Tags() {
			tags = append(tags, structTag{Key: tg.Key, Name: tg.Name, Field: ff})
		}
	}
	return tags, fields, nil
}

func isNil(v Value) bool {
	if v == nil {
		return true
	}
	vo := reflect.ValueOf(v)
	if vo.Kind() == reflect.Ptr && !vo.IsNil() {
		vo = reflect.Indirect(vo)
	}
	if !vo.IsValid() {
		return true
	}
	return false
}

// pointerTo returns a pointer to the given Value if it is not already one.
func pointerTo(v Value) interface{} {
	if v == nil {
		return true
	}
	vo := reflect.ValueOf(v)
	if vo.Kind() != reflect.Ptr && vo.CanAddr() {
		return vo.Pointer()
	}
	return vo.Interface()
}
