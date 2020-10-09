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

func TestWithInheritedOption(t *testing.T) {
	type Config struct {
		BasePath string
		Name     string
		Bio      *struct {
			BasePath string
			Age      int
		}
	}
	co := &Config{"/root", "veonik", &struct {
		BasePath string
		Age      int
	}{"", 30}}
	c, err := config.Wrap(co,
		config.WithRequiredOption("BasePath"),
		config.WithGenericSection(
			"Bio",
			config.WithInheritedOption("BasePath")))
	if err != nil {
		t.Errorf("expected config to be valid, but got error: %s", err)
		return
	}
	fmt.Printf("root BasePath: %s\n", co.BasePath)
	fmt.Printf("bio BasePath: %s\n", co.Bio.BasePath)
	_, ok := c.String("BasePath")
	if !ok {
		t.Errorf("expected BasePath to be a string")
		return
	}
	b, err := c.Section("Bio")
	if err != nil {
		t.Errorf("expected to get section named Bio, but got error: %s", err)
		return
	}
	bp, ok := b.String("BasePath")
	if !ok {
		t.Errorf("expected base-path to be a string")
		return
	}
	if co.Bio.BasePath != bp {
		t.Errorf("expected Bio.BasePath option (%s) to match BasePath field on Bio struct (%s)", bp, co.Bio.BasePath)
	}
	if co.BasePath != bp {
		t.Errorf("expected BasePath field on Config struct (%s) to match BasePath field on Bio struct (%s)", bp, co.BasePath)
	}
	if co.Bio.Age != 30 {
		t.Errorf("expected unmanaged field Age on Bio struct (%d) to equal initially set value (30)", co.Bio.Age)
	}
	// Outputs:
	// root BasePath: /root
	// bio BasePath: /root
}

func TestWithInheritedSection(t *testing.T) {
	c, err := config.New(
		config.WithGenericSection(
			"Test",
			config.WithRequiredOption("Var"),
			config.WithInitValue(map[string]interface{}{"Var": "test"})),
		config.WithGenericSection(
			"Sub",
			config.WithInheritedOption("Test")))
	if err != nil {
		t.Errorf("expected config to be valid, but got error: %s", err)
		return
	}
	st, err := c.Section("Test")
	if err != nil {
		t.Errorf("expected to get section named Test, but got error: %s", err)
		return
	}
	ss, err := c.Section("Sub")
	if err != nil {
		t.Errorf("expected to get section named Sub, but got error: %s", err)
		return
	}
	sst, err := ss.Section("Test")
	if err != nil {
		t.Errorf("expected to get section named Sub, but got error: %s", err)
		return
	}
	sts, ok := st.String("Var")
	if !ok {
		t.Errorf("expected base-path to be a string")
		return
	}
	ssts, ok := sst.String("Var")
	if !ok {
		t.Errorf("expected base-path to be a string")
		return
	}
	fmt.Printf("%s == %s\n", sts, ssts)
	// Outputs:
	// test == test
}
