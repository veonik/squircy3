package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squircy3/cli"
	"code.dopame.me/veonik/squircy3/irc"
)

var Version = "SNAPSHOT"

var interactive bool

func unboxAll(rootDir string) error {
	if _, err := os.Stat(rootDir); err == nil {
		// root directory already exists, don't muck with it
		return nil
	}
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create root directory")
	}
	box := packr.New("defconf", "./defconf")
	for _, f := range box.List() {
		dst := filepath.Join(rootDir, f)
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return errors.Wrap(err, "failed to recreate directory")
			}
			logrus.Infof("Creating default %s", dst)
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
	printVersion := false
	flag.BoolVar(&interactive, "interactive", false, "start interactive-read-evaluate-print (REPL) session")
	flag.BoolVar(&printVersion, "version", false, "print version information")
	cli.DefaultFlags(flag.CommandLine)

	flag.Usage = func() {
		fmt.Println("Usage: ", os.Args[0], "[options]")
		fmt.Println()
		fmt.Println("squircy is a proper IRC bot.")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if printVersion {
		fmt.Printf("squircy3 %s\n", Version)
		os.Exit(0)
	}
}

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	m, err := cli.NewManager()
	if err != nil {
		logrus.Fatalln("core: error initializing squircy:", err)
	}
	m.LinkedPlugins = append(m.LinkedPlugins, linkedPlugins...)
	logrus.SetLevel(m.LogLevel)
	if err := unboxAll(m.RootDir); err != nil {
		logrus.Fatalln("core: failed to unbox defaults:", err)
	}
	if err := m.Start(); err != nil {
		logrus.Fatalln("core: error starting squircy:", err)
	}
	ircm, err := irc.FromPlugins(m.Plugins())
	if err != nil {
		logrus.Errorln("core: failed to set irc version string:", err)
	}
	ircm.SetVersionString(fmt.Sprintf("squircy3 %s", Version))
	if interactive {
		go Repl(m)
	}
	if err = m.Loop(); err != nil {
		logrus.Fatalln("core: exiting main loop with error:", err)
	}
}
