package config_test

import (
	"flag"
	"fmt"
	"testing"

	"code.dopame.me/veonik/squircy3/config"
)

func TestWithValuesFromFlagSet(t *testing.T) {
	type Config struct {
		Name string
		Bio  *struct {
			Age int
		}
	}
	co := &Config{"veonik", &struct{ Age int }{30}}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.String("name", "", "your name")
	fs.Int("bio-age", 0, "you age")
	if err := fs.Parse([]string{"-name", "tyler", "-bio-age", "31"}); err != nil {
		t.Errorf("unexpected error parsing flagset: %s", err)
		return
	}
	c, err := config.Wrap(co,
		config.WithRequiredOption("Name"),
		config.WithGenericSection("Bio", config.WithRequiredOption("Age")),
		config.WithValuesFromFlagSet(fs))
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
	if co.Name != n {
		t.Errorf("expected Name option (%s) to match Name field on Config struct (%s)", n, co.Name)
	}
	if co.Bio.Age != a {
		t.Errorf("expected Bio.Age option (%d) to match Age field on Bio struct (%d)", a, co.Bio.Age)
	}
	fmt.Printf("%s is %d.\n", n, a)
	// Outputs:
	// Hi, tyler!
	// tyler is 31.
}
