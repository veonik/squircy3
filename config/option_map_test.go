package config_test

import (
	"fmt"
	"testing"

	"code.dopame.me/veonik/squircy3/config"
)

func TestWithValuesFromMap(t *testing.T) {
	type Config struct {
		Name string
		Bio  *struct {
			Age int
		}
	}
	co := &Config{"veonik", &struct{ Age int }{30}}
	opts := map[string]interface{}{
		"Name": "tyler",
		"Bio": map[string]interface{}{
			"Age": 31,
		},
	}
	c, err := config.Wrap(co,
		config.WithRequiredOption("Name"),
		config.WithGenericSection("Bio", config.WithRequiredOption("Age")),
		config.WithValuesFromMap(&opts))
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
