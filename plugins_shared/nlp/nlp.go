package main

import "code.dopame.me/veonik/squircy3/plugin"

func main() {}

type nlp struct{}

func (p *nlp) Name() string {
	return "nlp"
}

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	return &nlp{}, nil
}
