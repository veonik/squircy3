// Package config is a flexible configuration framework.
//
// This package defines a common interface for generic interaction with any
// structured configuration data. Configuration is organized by named sections
// that themselves contain values or other sections.
//
// In addition, this package defines a mutable configuration definition API
// for defining options, sections, and validation for options and sections.
//
// Underlying data for each section is stored in a map[string]interface{} or a
// struct with its exported fields being used as options. The underlying data
// is kept in sync when mutating with Config.Set() using reflection.
//
// Use config.New or config.Wrap to create a root section and specify whatever
// options desired.
//     type Config struct {
//         Name string
//         Bio  *struct {
//             Age int
//         }
//     }
//     co := &Config{"veonik", &struct{Age int}{30}}
//     c, err := config.Wrap(co,
//         config.WithRequiredOption("Name"),
//         config.WithGenericSection("Bio", config.WithRequiredOption("Age")))
//     if err != nil {
//         panic(err)
//     }
//     fmt.Printf("Hi, %s!\n", co.Name)
//     n, _ := c.String("Name")
//     b, _ := c.Section("Bio")
//     a, _ := b.Int("Age")
//     fmt.Printf("%s is %d.\n", n, a)
//     // Outputs:
//     // Hi, veonik!
//     // veonik is 30.
//
package config // import "code.dopame.me/veonik/squircy3/config"

// A Value is some value stored in a configuration.
type Value interface{}

// A Config represents a single, configured section.
// Configs are collections of Values each with one or more keys referencing
// each Value stored.
// Configs may be nested within other Configs by using sections.
type Config interface {
	// Self returns the Value stored for the Config itself.
	// This will be a map[string]interface{} unless otherwise set with an
	// initial value or prototype func.
	Self() Value
	// Get returns the Value stored with the given key.
	// The second return parameter will be false if the given key is unset.
	Get(key string) (Value, bool)
	// String returns the string stored with the given key.
	// The second return parameter will be false if the given key is unset
	// or not a string.
	String(key string) (string, bool)
	// Bool returns the bool stored with the given key.
	// The second return parameter will be false if the given key is unset
	// or not a bool.
	Bool(key string) (bool, bool)
	// Int returns the int stored with the given key.
	// The second return parameter will be false if the given key is unset
	// or not an int.
	Int(key string) (int, bool)
	// Set sets the given key to the given Value.
	Set(key string, val Value)

	// Section returns the nested configuration for the given key.
	// If the section does not exist, an error will be returned.
	Section(key string) (Config, error)
}

// New creates and populates a new Config using the given options.
func New(options ...SetupOption) (Config, error) {
	s := newSetup("root", nil)
	if err := s.apply(options...); err != nil {
		return nil, err
	}
	if err := walkAndWrap(s); err != nil {
		return nil, err
	}
	return s.config, s.validate()
}

// Wrap creates and populates a new Config using the given Value as the stored
// representation of the configuration.
func Wrap(wrapped Value, options ...SetupOption) (Config, error) {
	return New(append([]SetupOption{WithInitValue(wrapped)}, options...)...)
}
