package main

import (
	"github.com/dop251/goja"
	"github.com/pkg/errors"
)

type Babel struct {
	runtime *goja.Runtime

	transform goja.Callable
}

func NewBabel(r *goja.Runtime) (*Babel, error) {
	b := &Babel{runtime: r}
	v, err := b.runtime.RunString(`
this.global = this.global || this;
this.process = this.process || require('process/browser');
require('core-js-bundle');
this.Babel = require('@babel/standalone');
var plugin = require('regenerator-transform');
var transform = function(src) { 
    var res = Babel.transform(src, {presets: ['es2015'], plugins: [plugin]}); 
    return res.code; 
};
transform`)
	if err != nil {
		return nil, err
	}
	fn, ok := goja.AssertFunction(v)
	if !ok {
		return nil, errors.Errorf("expected result to be goja.Callable, got %T", v)
	}
	b.transform = fn
	return b, nil
}

func (b *Babel) Compile(name, in string) (*goja.Program, error) {
	r, err := b.Transform(in)
	if err != nil {
		return nil, err
	}
	return goja.Compile(name, r, true)
}

func (b *Babel) Transform(in string) (string, error) {
	vs := b.runtime.ToValue(in)
	v, err := b.transform(nil, vs)
	if err != nil {
		return "", err
	}
	var res string

	err = b.runtime.ExportTo(v.ToObject(b.runtime), &res)
	if err != nil {
		return "", err
	}
	return res, nil
}
