package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
var logLevel = stringLevel(logrus.DebugLevel)
var interactive bool

func init() {
	flag.StringVar(&rootDir, "root", "~/.squircy", "path to folder containing squircy data")
	flag.Var(&logLevel, "log-level", "controls verbosity of logging output")
	flag.Var(&extraPlugins, "plugin", "path to shared plugin .so file, multiple plugins may be given")
	flag.BoolVar(&interactive, "interactive", false, "start interactive-read-evaluate-print (REPL) session")

	flag.Usage = func() {
		fmt.Println("Usage: ", os.Args[0], "[options]")
		fmt.Println()
		fmt.Println("squircy is a proper IRC bot.")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()
	bp, err := tilde.Expand(rootDir)
	if err != nil {
		logrus.Fatalln(err)
	}
	err = os.MkdirAll(bp, os.FileMode(0644))
	if err != nil {
		logrus.Fatalln(err)
	}
	rootDir = bp
}

func main() {
	logrus.SetLevel(logrus.Level(logLevel))
	m, err := cli.NewManager(rootDir, extraPlugins...)
	if err != nil {
		logrus.Fatalln("error initializing squircy:", err)
	}
	if err := m.Start(); err != nil {
		logrus.Fatalln("error starting squircy:", err)
	}
	if interactive {
		go func() {
			Repl(m)
		}()
	}
	if err = m.Loop(); err != nil {
		logrus.Fatalln("exiting main loop with error:", err)
	}
}
