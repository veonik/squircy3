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
//         config.WithGenericSection("Bio", config.WithInitValue(co.Bio), config.WithRequiredOption("Age")))
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

type Value interface{}

type Config interface {
	Get(key string) (Value, bool)
	String(key string) (string, bool)
	Bool(key string) (bool, bool)
	Int(key string) (int, bool)
	Set(key string, val Value)

	Section(key string) (Config, error)
}

type Section interface {
	Prefix() string
	Prototype() Value
	Singleton() bool
}

func New(options ...SetupOption) (Config, error) {
	s := newSetup("root")
	if err := s.apply(options...); err != nil {
		return nil, err
	}
	if err := walkAndWrap(s); err != nil {
		return nil, err
	}
	return s.config, s.validate()
}

func Wrap(wrapped Value, options ...SetupOption) (Config, error) {
	return New(append([]SetupOption{WithInitValue(wrapped)}, options...)...)
}
