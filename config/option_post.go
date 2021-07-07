package config

import (
	"flag"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// populateValuesFromTOMLFile runs after other SetupOptions and does the actual
// filling of values in the Config from the TOML file.
func populateValuesFromTOMLFile(filename string) postSetupOption {
	return func(s *Setup) error {
		if s.raw == nil {
			s.raw = make(map[string]interface{})
		}
		if _, err := toml.DecodeFile(filename, &s.raw); err != nil {
			if os.IsNotExist(err) {
				logrus.Warnf("config: unable to load toml config, but not aborting: %s", err)
				return nil
			}
			return err
		}
		return nil
	}
}

// WithValuesFromTOMLFile will populate the Config with values parsed from a
// TOML file.
func WithValuesFromTOMLFile(filename string) SetupOption {
	return func(s *Setup) error {
		return s.prependPostSetup(populateValuesFromTOMLFile(filename))
	}
}

var camelCaseMatcher = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")
var dashAndSpaceMatcher = regexp.MustCompile("([-\\s])")

// a nameFieldMapper converts a flag name into path parts that correspond to
// sections and options defined in the Setup.
type nameFieldMapper struct {
	s *Setup
}

func newNameFieldMapper(s *Setup) *nameFieldMapper {
	return &nameFieldMapper{s}
}

// normalizeName converts the given name into a normal, underscorized name.
// Dashes are converted to underscores, camel case is separated by underscores
// and converts everything to lower-case.
func normalizeName(name string) string {
	name = dashAndSpaceMatcher.ReplaceAllString(name, "_")
	var a []string
	for _, sub := range camelCaseMatcher.FindAllStringSubmatch(name, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToLower(strings.Join(a, "_"))
}

// Map converts a flag name into a path based on sections and options.
// If a Config has a section "Templating" which contains the section "Twig"
// which has an option "views_path", then:
//   -templating-twig-views-path
// is converted into the path:
// 	 ["Templating","Twig","views_path"]
func (fm *nameFieldMapper) Map(flagName string) (path []string) {
	normal := normalizeName(flagName)
	s := fm.s
loop:
	for s != nil {
		// check for a matching section, using the name as a prefix
		for k, ks := range s.sections {
			kn := normalizeName(k) + "_"
			if strings.HasPrefix(normal, kn) {
				// found the next step in the path
				normal = strings.Replace(normal, kn, "", 1)
				path = append(path, k)
				logrus.Tracef("config: descending into section %s (from %s) to find match for option '%s'", ks.name, s.name, normal)
				s = ks
				goto loop
			}
		}
		// check for a match in options next
		for k := range s.options {
			kn := normalizeName(k)
			if kn == normal {
				// found it
				path = append(path, k)
				logrus.Tracef("config: found option with name '%s' in section '%s'", normal, s.name)
				s = nil
				goto loop
			}
		}
		// finally, try a valueInspector here to handle struct tag aliases.
		is, err := inspect(s.initial)
		if err != nil {
			logrus.Debugf("config: unable to create valueInspector for section '%s': %s", s.name, err)
		} else {
			if name, err := is.Normalize(normal); err == nil {
				path = append(path, name)
				logrus.Tracef("config: valueInspector found match for field '%s' in section '%s'", name, s.name)
				s = nil
				goto loop
			} else {
				if !strings.Contains(err.Error(), "no field with name") {
					logrus.Debugf("config: valueInspector returned error for '%s' in section '%s': %s", normal, s.name, err)
				}
			}
		}
		return nil
	}
	return path
}

func visitNamedOption(name string, raw map[string]interface{}, f string, fv interface{}, m *nameFieldMapper, override bool) {
	path := m.Map(f)
	if len(path) == 0 {
		logrus.Debugf("config: did not match anything for named option '%s' for section %s", f, name)
		return
	}
	logrus.Debugf("config: named option '%s' mapped to path %v and value %T(%v)", f, path, fv, fv)
	val := fv
	v := raw
	i := 0
	// iterate over all but the last part of the path, descending into a
	// new section with each iteration.
	for i = 0; i < len(path)-1; i++ {
		if vs, ok := v[path[i]].(map[string]interface{}); ok {
			v = vs
		} else {
			if vr, ok := v[path[i]]; ok {
				if vrn, ok := vr.(map[string]interface{}); ok {
					// value exists and is already a map[string]interface{}, use it.
					v = vrn
					continue
				}
				// there is no path[0-1] so figure out the name accordingly
				secn := name
				if i > 0 {
					secn = path[i-1]
				}
				logrus.Debugf("config: overriding existing value in section %s for option '%s' -- was type %T", secn, path[i], vr)
			}
			nv := make(map[string]interface{})
			v[path[i]] = nv
			v = nv
		}
	}
	if !override {
		if _, ok := v[path[i]]; ok {
			// value is already set, don't override it.
			return
		}
	}
	// use the last element in the path to set the right option.
	v[path[i]] = val
}

func populateValuesFromFlagSet(fs *flag.FlagSet) postSetupOption {
	return func(s *Setup) error {
		logrus.Tracef("config: populating values from FlagSet %s", fs.Name())
		m := newNameFieldMapper(s)
		fs.Visit(func(f *flag.Flag) {
			var v interface{} = f.Value.String()
			if fv, ok := f.Value.(flag.Getter); ok {
				v = fv.Get()
			}
			visitNamedOption(s.name, s.raw, f.Name, v, m, true)
		})
		return nil
	}
}

// WithValuesFromFlagSet populates the Config using command-line flags.
func WithValuesFromFlagSet(fs *flag.FlagSet) SetupOption {
	return func(s *Setup) error {
		if !fs.Parsed() {
			return errors.Errorf("given FlagSet must be parsed")
		}
		if s.raw == nil {
			s.raw = make(map[string]interface{})
		}
		m := newNameFieldMapper(s)
		fs.VisitAll(func(f *flag.Flag) {
			var v interface{} = f.Value.String()
			if fv, ok := f.Value.(flag.Getter); ok {
				v = fv.Get()
			}
			visitNamedOption(s.name, s.raw, f.Name, v, m, false)
		})

		return s.appendPostSetup(populateValuesFromFlagSet(fs))
	}
}

func populateValuesFromMap(vs *map[string]interface{}) postSetupOption {
	return func(s *Setup) error {
		logrus.Tracef("config: populating values from map: %p", vs)
		if s.raw == nil {
			s.raw = make(map[string]interface{})
		}
		m := newNameFieldMapper(s)
		for f, fv := range *vs {
			visitNamedOption(s.name, s.raw, f, fv, m, true)
		}
		return nil
	}
}

// WithValuesFromMap populates the Config using the given map.
func WithValuesFromMap(vs *map[string]interface{}) SetupOption {
	return func(s *Setup) error {
		return s.appendPostSetup(populateValuesFromMap(vs))
	}
}
