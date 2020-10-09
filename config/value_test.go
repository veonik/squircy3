package config

import (
	"testing"
)

func TestConfigurable_withMap(t *testing.T) {
	co := map[string]interface{}{}
	s := newSetup("root", nil)
	err := s.apply(WithInitValue(co))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	c, err := newConfigurable(s)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	v, ok := c.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected value to be map[string]interface{}, got %T", c.Value)
	}
	if len(v) > 0 {
		t.Fatalf("expected value to be empty: %s", v)
	}
	c.Set("Test", "value")
	if co["Test"] != "value" {
		t.Fatalf("expected Test field on struct to contain 'value', but got '%s'", co["Test"])
	}
	vs, ok := c.String("Test")
	if !ok {
		t.Fatalf("expected Get call to return a value")
	}
	if vs != "value" {
		t.Fatalf("expected value to contain 'value', got '%s'", vs)
	}
}

type TestConfig struct {
	Test string
}

func TestConfigurable_withStruct(t *testing.T) {
	co := &TestConfig{}
	s := newSetup("root", nil)
	err := s.apply(WithInitValue(co))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	c, err := newConfigurable(s)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	v, ok := c.Value.(*TestConfig)
	if !ok {
		t.Fatalf("expected value to be *Config, got %T", c.Value)
	}
	if len(v.Test) > 0 {
		t.Fatalf("expected value to be empty: %s", v)
	}
	c.Set("Test", "value")
	if co.Test != "value" {
		t.Fatalf("expected Test field on struct to contain 'value', but got '%s'", co.Test)
	}
	vs, ok := c.String("Test")
	if !ok {
		t.Fatalf("expected Get call to return a value")
	}
	if vs != "value" {
		t.Fatalf("expected value to contain 'value', got '%s'", vs)
	}
}
