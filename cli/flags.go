package cli

import (
	"flag"
	"fmt"
	"strings"
)

func PluginOptsFlag(fs *flag.FlagSet, name, usage string) {
	val := make(pluginOptsFlag)
	PluginOptsFlagVar(fs, &val, name, usage)
}

func PluginOptsFlagVar(fs *flag.FlagSet, val flag.Value, name, usage string) {
	fs.Var(val, name, usage)
}

type pluginOptsFlag map[string]interface{}

func (s pluginOptsFlag) String() string {
	var res []string
	for k, v := range s {
		res = append(res, fmt.Sprintf("%s=%v", k, v))
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

func (s pluginOptsFlag) Get() interface{} {
	return map[string]interface{}(s)
}

type stringsFlag []string

func (s stringsFlag) String() string {
	return strings.Join(s, "")
}

func (s *stringsFlag) Set(str string) error {
	*s = append(*s, str)
	return nil
}

func (s stringsFlag) Get() interface{} {
	return []string(s)
}

func DefaultFlags(fs *flag.FlagSet) {
	CoreFlags(fs, "~/.squircy")
	IRCFlags(fs)
	VMFlags(fs)
}

func CoreFlags(fs *flag.FlagSet, root string) {
	extraPlugins := stringsFlag{}
	fs.String("root", root, "path to folder containing application data")
	fs.String("log-level", "info", "controls verbosity of logging output")
	fs.Var(&extraPlugins, "plugin", "path to shared plugin .so file, multiple plugins may be given")
	PluginOptsFlag(fs, "plugin-option", "specify extra plugin configuration option, format: key=value")
}

func IRCFlags(fs *flag.FlagSet) {
	fs.Bool("irc-auto", false, "automatically connect to irc")
	fs.String("irc-nick", "squishyjones", "specify irc nickname")
	fs.String("irc-user", "mrjones", "specify irc user")
	fs.String("irc-network", "chat.freenode.net:6697", "specify irc network")
	fs.Bool("irc-tls", true, "use tls encryption when connecting to irc")
	fs.Bool("irc-sasl", false, "use sasl authentication")
	fs.String("irc-sasl-username", "", "specify sasl username")
	fs.String("irc-sasl-password", "", "specify sasl password")
	fs.String("irc-server-password", "", "specify server password")
}

func VMFlags(fs *flag.FlagSet) {
	fs.String("vm-modules-path", "node_modules", "specify javascript modules path")
}
