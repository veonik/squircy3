package transformer // import "code.dopame.me/veonik/squircy3/plugins/babel/transformer"

import (
	"github.com/dop251/goja"
	"github.com/pkg/errors"
)

type Babel struct {
	runtime *goja.Runtime

	transform goja.Callable
}

func New(r *goja.Runtime) (*Babel, error) {
	b := &Babel{runtime: r}
	v, err := b.runtime.RunString(`
this.global = this.global || this;
this.process = this.process || require('process/browser');
// core-js does not set this up correctly, so we do it explicitly here
Object.setPrototypeOf = function(O, proto) {
	if(! (O instanceof Object)) {
		throw new TypeError("Expected object");
	}
	if(proto !== null && ! (proto instanceof Object)) {
		throw new TypeError("Expected prototype to be object or null");
	}
	O.__proto__ = proto;
	return O;
};
Object.getPrototypeOf = function(O) {
	if(! (O instanceof Object)) {
		throw new TypeError("Expected object");
	}
	return O.__proto__;
}
require('core-js-bundle');
this.Babel = require('@babel/standalone');
var plugin = require('regenerator-transform');
(function(src) { 
    var res = Babel.transform(src, {presets: ['es2015'], plugins: [plugin]}); 
    return res.code; 
})`)
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
