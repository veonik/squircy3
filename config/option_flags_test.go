package config_test

import (
	"flag"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"

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
	fs.Int("bio-age", 0, "your age")
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

func TestWithValuesFromFlagSetDefaultValues(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	co := map[string]interface{}{}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.String("name", "veonik", "your name")
	fs.Int("bio-age", 31, "your age")
	if err := fs.Parse([]string{}); err != nil {
		t.Errorf("unexpected error parsing flagset: %s", err)
		return
	}
	c, err := config.Wrap(co,
		config.WithRequiredOption("name"),
		config.WithGenericSection("bio", config.WithRequiredOption("age")),
		config.WithValuesFromFlagSet(fs))
	if err != nil {
		t.Errorf("expected config to be valid, but got error: %s", err)
		return
	}
	fmt.Printf("Hi, %s!\n", co["name"])
	n, ok := c.String("name")
	if !ok {
		t.Errorf("expected name to be a string")
		return
	}
	b, err := c.Section("bio")
	if err != nil {
		t.Errorf("expected config to be valid, but got error: %s", err)
		return
	}
	a, ok := b.Int("age")
	if !ok {
		t.Errorf("expected age to be an int")
		return
	}
	if co["name"] != n {
		t.Errorf("expected Name option (%s) to match Name field on Config struct (%s)", n, co["name"])
		return
	}
	bio, ok := co["bio"].(map[string]interface{})
	if !ok {
		t.Errorf("expected map to contain another map with key bio, but got: %v", bio)
		return
	}
	if bio["age"] != a {
		t.Errorf("expected bio.age option (%d) to match Age field on Bio struct (%d)", a, bio["age"])
	}
	fmt.Printf("%s is %d.\n", n, a)
	// Outputs:
	// Hi, tyler!
	// tyler is 31.
}
