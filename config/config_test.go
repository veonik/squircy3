package config_test

import (
	"fmt"
	"testing"

	"code.dopame.me/veonik/squircy3/config"
)

func TestWrap(t *testing.T) {
	type Config struct {
		Name string
		Bio  *struct {
			Age int
		}
	}
	co := &Config{"veonik", &struct{ Age int }{30}}
	c, err := config.Wrap(co,
		config.WithRequiredOption("Name"),
		config.WithGenericSection("Bio", config.WithInitValue(co.Bio), config.WithRequiredOption("Age")))
	if err != nil {
		t.Errorf("expected config to be valid, but got error: %s", err)
		return
	}
	fmt.Printf("Hi, %s!\n", co.Name)
	n, ok := c.String("Name")
	if !ok {
		t.Errorf("expected name to be a string")
		return
	}
	b, err := c.Section("Bio")
	if err != nil {
		t.Errorf("expected config to be valid, but got error: %s", err)
		return
	}
	a, ok := b.Int("Age")
	if !ok {
		t.Errorf("expected age to be an int")
		return
	}
	fmt.Printf("%s is %d.\n", n, a)
	// Outputs:
	// Hi, veonik!
	// veonik is 30.
}

func TestWrap2(t *testing.T) {
	type Config struct {
		Name string
		Bio  struct {
			Age int
		}
	}
	co := &Config{"veonik", struct{ Age int }{30}}
	c, err := config.Wrap(co,
		config.WithRequiredOption("Name"),
		config.WithGenericSection("Bio", config.WithInitValue(&co.Bio), config.WithRequiredOption("Age")))
	if err != nil {
		t.Errorf("expected config to be valid, but got error: %s", err)
		return
	}
	fmt.Printf("Hi, %s!\n", co.Name)
	n, ok := c.String("Name")
	if !ok {
		t.Errorf("expected name to be a string")
		return
	}
	b, err := c.Section("Bio")
	if err != nil {
		t.Errorf("expected config to be valid, but got error: %s", err)
		return
	}
	a, ok := b.Int("Age")
	if !ok {
		t.Errorf("expected age to be an int")
		return
	}
	fmt.Printf("%s is %d.\n", n, a)
	// Outputs:
	// Hi, veonik!
	// veonik is 30.
}
