package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tilde "gopkg.in/mattes/go-expand-tilde.v1"

	"code.dopame.me/veonik/squircy3/cli"
)

type stringsFlag []string

func (s stringsFlag) String() string {
	return strings.Join(s, "")
}
func (s *stringsFlag) Set(str string) error {
	*s = append(*s, str)
	return nil
}

type pluginOptsFlag map[string]interface{}

func (s pluginOptsFlag) String() string {
	var res []string
	for k, v := range s {
		res = append(res, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(res, " ")
}

func (s pluginOptsFlag) Set(str string) error {
	p := strings.SplitN(str, "=", 2)
	if len(p) == 1 {
		p = append(p, "true")
	}
	var v interface{}
	if p[1] == "true" {
		v = true
	} else if p[1] == "false" {
		v = false
	} else {
		v = p[1]
	}
	s[p[0]] = v
	return nil
}

type stringLevel logrus.Level

func (s stringLevel) String() string {
	return logrus.Level(s).String()
}
func (s *stringLevel) Set(str string) error {
	l, err := logrus.ParseLevel(str)
	if err != nil {
		return err
	}
	*s = stringLevel(l)
	return nil
}

var rootDir string
var extraPlugins stringsFlag
var pluginOptions = make(pluginOptsFlag)
var logLevel = stringLevel(logrus.InfoLevel)
var interactive bool

func unboxAll(rootDir string) error {
	box := packr.New("defconf", "./defconf")
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create root directory")
	}
	for _, f := range box.List() {
		dst := filepath.Join(rootDir, f)
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			logrus.Infof("Creating default %s as %s", f, dst)
			d, err := box.Find(f)
			if err != nil {
				return errors.Wrapf(err, "failed to get contents of boxed %s", f)
			}
			if err := ioutil.WriteFile(dst, d, 0644); err != nil {
				return errors.Wrapf(err, "failed to write unboxed file %s", f)
			}
		}
	}
	return nil
}

func init() {
	flag.StringVar(&rootDir, "root", "~/.squircy", "path to folder containing squircy data")
	flag.Var(&logLevel, "log-level", "controls verbosity of logging output")
	flag.Var(&extraPlugins, "plugin", "path to shared plugin .so file, multiple plugins may be given")
	flag.BoolVar(&interactive, "interactive", false, "start interactive-read-evaluate-print (REPL) session")
	flag.Var(&pluginOptions, "plugin-option", "specify extra plugin configuration option, format: key=value")
	flag.Bool("irc-auto", false, "automatically connect to irc")
	flag.String("irc-nick", "squishyjones", "specify irc nickname")
	flag.String("irc-user", "mrjones", "specify irc user")
	flag.String("irc-network", "chat.freenode.net:6697", "specify irc network")
	flag.Bool("irc-tls", true, "use tls encryption when connecting to irc")
	flag.Bool("irc-sasl", false, "use sasl authentication")
	flag.String("irc-sasl-username", "", "specify sasl username")
	flag.String("irc-sasl-password", "", "specify sasl password")
	flag.String("irc-server-password", "", "specify server password")
	flag.String("vm-modules-path", "node_modules", "specify javascript modules path")

	flag.Usage = func() {
		fmt.Println("Usage: ", os.Args[0], "[options]")
		fmt.Println()
		fmt.Println("squircy is a proper IRC bot.")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()
	var err error
	if rootDir, err = tilde.Expand(rootDir); err != nil {
		logrus.Fatalln("core: failed determine root directory:", err)
	}
	if err := unboxAll(rootDir); err != nil {
		logrus.Fatalln("core: failed to unbox defaults:", err)
	}
}

func main() {
	logrus.SetLevel(logrus.Level(logLevel))
	m, err := cli.NewManager(rootDir, pluginOptions, extraPlugins...)
	if err != nil {
		logrus.Fatalln("core: error initializing squircy:", err)
	}
	if err := m.Start(); err != nil {
		logrus.Fatalln("core: error starting squircy:", err)
	}
	if interactive {
		go func() {
			Repl(m)
		}()
	}
	if err = m.Loop(); err != nil {
		logrus.Fatalln("core: exiting main loop with error:", err)
	}
}
