package main_test

import (
	"crypto/sha1"
	"fmt"
	"os"
	"testing"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	babel "code.dopame.me/veonik/squircy3/plugins/babel/transformer"
	node_compat "code.dopame.me/veonik/squircy3/plugins/node_compat"
	"code.dopame.me/veonik/squircy3/plugins/node_compat/internal"
	"code.dopame.me/veonik/squircy3/vm"
)

func init() {
	if _, err := os.Stat("../../testdata/node_modules"); os.IsNotExist(err) {
		panic("tests in this package require node dependencies to be installed in the testdata directory")
	}
}

func HandleRuntimeInit(vmp *vm.VM) func(*goja.Runtime) {
	return func(gr *goja.Runtime) {
		vmp.SetTransformer(nil)
		b, err := babel.New(gr)
		if err != nil {
			logrus.Warnln("unable to run babel init script:", err)
			return
		}
		vmp.SetTransformer(b.Transform)

		v := gr.NewObject()
		if err := v.Set("Sum", func(b []byte) (string, error) {
			return fmt.Sprintf("%x", sha1.Sum(b)), nil
		}); err != nil {
			logrus.Warnf("%s: error initializing runtime: %s", node_compat.PluginName, err)
		}
		gr.Set("sha1", v)

		v = gr.NewObject()
		if err := v.Set("Dial", internal.Dial); err != nil {
			logrus.Warnf("%s: error initializing runtime: %s", node_compat.PluginName, err)
		}
		if err := v.Set("Listen", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) != 3 {
				panic(gr.NewGoError(errors.New("expected exactly 3 arguments")))
			}
			arg0 := call.Arguments[0]
			fn, ok := goja.AssertFunction(arg0)
			if !ok {
				panic(gr.NewGoError(errors.New("expected argument 0 to be callable")))
			}
			kind := call.Arguments[1].String()
			addr := call.Arguments[2].String()
			srv, err := internal.Listen(vmp, fn, kind, addr)
			if err != nil {
				panic(gr.NewGoError(err))
			}
			return gr.ToValue(srv)
		}); err != nil {
			logrus.Warnf("%s: error initializing runtime: %s", node_compat.PluginName, err)
		}
		gr.Set("internal", v)

		_, err = gr.RunString(`this.global = this.global || this;
require('core-js-bundle');
this.process = this.process || require('process/browser');
require('regenerator-runtime');`)
		if err != nil {
			logrus.Warnf("%s: error initializing runtime: %s", node_compat.PluginName, err)
		}
	}
}

var registry = vm.NewRegistry("../../testdata")

func TestNodeCompat_Net(t *testing.T) {
	vmp, err := vm.New(registry)
	if err != nil {
		t.Fatalf("unexpected error creating VM: %s", err)
	}
	vmp.SetModule(node_compat.EventEmitter)
	vmp.SetModule(node_compat.ChildProcess)
	vmp.SetModule(node_compat.Crypto)
	vmp.SetModule(node_compat.Stream)
	vmp.SetModule(node_compat.Net)
	vmp.SetModule(node_compat.Http)
	vmp.OnRuntimeInit(HandleRuntimeInit(vmp))
	if err = vmp.Start(); err != nil {
		t.Fatalf("unexpected error starting VM: %s", err)
	}
	res, err := vmp.RunString(`

import {Socket, Server} from 'net';

const sleep = async (d) => {
	return new Promise(resolve => {
		setTimeout(() => resolve(), d);
	});
};

let resolve;
let output = '';
let result = new Promise(_resolve => {
	resolve = _resolve;
});

// let originalLog = console.log;
// console.log = function log() {
// 	let args = Array.from(arguments).map(arg => arg.toString());
// 	originalLog(args.join(' '));
// };

(async () => {
	var srv = new Server();
	srv.listen(3333, 'localhost', async conn => {
		console.log('connected');
		conn.on('data', data => {
			console.log('server received', data.toString());
		});
		conn.on('close', () => console.log('server side disconnected'));
		conn.on('end', () => {
			console.log('ending server connection from user code!');
			srv.close();
		});
		conn.on('ready', () => {
			console.log('server: ' + conn.localAddress + ':' + conn.localPort);
			console.log('client: ' + conn.remoteAddress + ':' + conn.remotePort);
		});
		conn.write('hi');
		await sleep(500);
		conn.write('exit\n');
	});
	srv.on('close', () => {
		resolve(output);
	});
	console.log('listening on', srv.address());
})();

(async () => {
	let sock = new Socket();
	console.log('wot');
	sock.on('data', d => {
		let data = d.toString();
		console.log('received', data);
		if(data.replace(/\n$/, '') === 'exit') {
			sock.end('peace!');
			sock.destroy();
			return;
		} else {
			output += d;
		}
	});
	sock.on('close', () => console.log('client side disconnected'));
	await sock.connect({host: 'localhost', port: 3333});
	sock.write('hello there!\r\n');
	console.log('wot2');
})();

result;

`).Await()
	if err != nil {
		t.Fatalf("error requiring module: %s", err)
	}
	expected := "hi"
	if res.String() != expected {
		t.Fatalf("expected: %s\ngot: %s", expected, res.String())
	}
}
