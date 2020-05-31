package transformer_test

import (
	"os"
	"testing"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squircy3/plugins/babel/transformer"
	"code.dopame.me/veonik/squircy3/vm"
)

func init() {
	if _, err := os.Stat("../../../testdata/node_modules"); os.IsNotExist(err) {
		panic("tests in this package require node dependencies to be installed in the testdata directory")
	}
}

func HandleRuntimeInit(vmp *vm.VM) func(*goja.Runtime) {
	return func(gr *goja.Runtime) {
		vmp.SetTransformer(nil)
		b, err := transformer.New(gr)
		if err != nil {
			logrus.Warnln("unable to run babel init script:", err)
			return
		}
		vmp.SetTransformer(b.Transform)
	}
}

var registry = vm.NewRegistry("../../../testdata")

func TestBabel_Transform(t *testing.T) {
	vmp, err := vm.New(registry)
	if err != nil {
		t.Fatalf("unexpected error creating VM: %s", err)
	}
	// vmp.SetModule(&vm.Module{Name: "events", Path: "./events.js", Main: "index"})
	vmp.OnRuntimeInit(HandleRuntimeInit(vmp))
	if err = vmp.Start(); err != nil {
		t.Fatalf("unexpected error starting VM: %s", err)
	}
	res, err := vmp.RunString(`require('regenerator-runtime');

(async () => {
	let output = null;
	
	setTimeout(() => {
		output = "HELLO!";
	}, 200);
	
	const sleep = async (d) => {
		return new Promise(resolve => {
			setTimeout(() => resolve(), d);
		});
	};
	
	await sleep(500);
	
	return output;
})();
`).Await()
	if err != nil {
		t.Fatalf("error requiring module: %s", err)
	}
	expected := "HELLO!"
	if res.String() != expected {
		t.Fatalf("expected: %s\ngot: %s", expected, res.String())
	}
}
