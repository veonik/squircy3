package squircy2_compat

import (
	"crypto/sha256"
	"fmt"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/plugins/squircy2_compat/data"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"
)


// must logs the given error as a warning
func must(what string, err error) {
	if err != nil {
		logrus.Warnf("squiryc2_compat: error %s: %s", what, err)
	}
}

func (p *HelperSet) Enable(gr *goja.Runtime) {
	p.setDispatcher(gr)
	p.setAsyncFunc(gr)
	p.setDataHelper(gr)

	gr.Set("Http", p.httpHelper(gr))
	gr.Set("Config", &p.conf)
	gr.Set("Irc", p.ircHelper(gr))
	gr.Set("File", p.fileHelper(gr))
}

func (p *HelperSet) setDispatcher(gr *goja.Runtime) {
	getFnName := func(fn goja.Value) (name string) {
		s := sha256.Sum256([]byte(fmt.Sprintf("%v", fn)))
		return fmt.Sprintf("__Handler%x", s)
	}
	if p.funcs != nil {
		for _, f := range p.funcs {
			p.events.Unbind(f.eventType, f.handler)
		}
	}
	p.funcs = map[string]callback{}
	gr.Set("bind", func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		arg1 := call.Argument(1)
		if fn, ok := goja.AssertFunction(arg1); ok {
			fnName := getFnName(arg1)
			h := func(ev *event.Event) {
				dat := make(map[string]interface{}, len(ev.Data))
				for k, v := range ev.Data {
					dat[k] = v
				}
				p.vm.Do(func(r *goja.Runtime) {
					d := r.ToValue(dat)
					_, err := fn(nil, d)
					if err != nil {
						logrus.Warnln("error running", eventType, "callback:", err)
					}
				})
			}
			p.funcs[fnName] = callback{eventType: eventType, callable: fn, handler: h}
			p.events.Bind(eventType, h)
			return gr.ToValue(fnName)
		}
		panic(gr.NewTypeError("expected arg1 to be Function"))
	})

	gr.Set("unbind", func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		arg1 := call.Argument(1)
		fnName := getFnName(arg1)
		if in, ok := p.funcs[fnName]; ok {
			p.events.Unbind(eventType, in.handler)
			delete(p.funcs, fnName)
		} else {
			logrus.Debugln("unbind called with unknown (or unbound) handler", eventType, arg1, fnName)
		}
		return goja.Undefined()
	})
	emit := func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		var dat map[string]interface{}
		if err := gr.ExportTo(call.Argument(1), &dat); err != nil {
			panic(gr.NewTypeError("expected arg1 to be an object", call.Argument(1)))
		}
		p.events.Emit(eventType, dat)
		return goja.Undefined()
	}
	gr.Set("trigger", emit)
	gr.Set("emit", emit)
}

func (p *HelperSet) setAsyncFunc(gr *goja.Runtime) {
	gr.Set("async", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 2 {
			panic(gr.NewTypeError("expected 2 arguments"))
		}
		async, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(gr.NewTypeError("expected argument 1 to be a Function"))
		}
		await, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			panic(gr.NewTypeError("expected argument 2 to be a Function"))
		}
		v, err := async(nil)
		if err != nil {
			panic(gr.NewGoError(err))
		}
		_, err = await(nil, v)
		if err != nil {
			panic(gr.NewGoError(err))
		}
		return goja.Undefined()
	})
}

func (p *HelperSet) setDataHelper(gr *goja.Runtime) {
	gr.Set("use", func(call goja.FunctionCall) goja.Value {
		coll := call.Argument(0).String()
		db := data.NewGenericRepository(p.db, coll)
		obj := gr.NewObject()
		what := func(do string) string {
			return fmt.Sprintf("binding %s for collection %s", do, coll)
		}
		must(what("Save"), obj.Set("Save", func(call goja.FunctionCall) goja.Value {
			exp := call.Argument(0).Export()
			var model data.GenericModel
			switch t := exp.(type) {
			case data.GenericModel:
				model = t

			case map[string]interface{}:
				model = t

			default:
				panic(fmt.Sprintf("unsupported type %T", exp))
			}
			switch t := model["ID"].(type) {
			case int64:
				model["ID"] = int(t)

			case int:
				model["ID"] = t

			case float64:
				model["ID"] = int(t)
			}
			db.Save(model)

			return gr.ToValue(model["ID"])
		}))
		must(what("Delete"), obj.Set("Delete", func(call goja.FunctionCall) goja.Value {
			i := call.Argument(0).ToInteger()
			db.Delete(int(i))

			return gr.ToValue(true)
		}))
		must(what("Fetch"), obj.Set("Fetch", func(call goja.FunctionCall) goja.Value {
			i := call.Argument(0).ToInteger()
			val := db.Fetch(int(i))
			return gr.ToValue(val)
		}))
		must(what("FetchAll"), obj.Set("FetchAll", func(call goja.FunctionCall) goja.Value {
			vals := db.FetchAll()
			return gr.ToValue(vals)
		}))
		must(what("Index"), obj.Set("Index", func(call goja.FunctionCall) goja.Value {
			exp := call.Argument(0).Export()
			cols := make([]string, 0)
			for _, val := range exp.([]interface{}) {
				cols = append(cols, val.(string))
			}
			db.Index(cols)

			return goja.Undefined()
		}))
		must(what("Query"), obj.Set("Query", func(call goja.FunctionCall) goja.Value {
			qry := call.Argument(0).Export()
			vals := db.Query(qry)
			return gr.ToValue(vals)
		}))
		return obj
	})
}

func (p *HelperSet) ircHelper(gr *goja.Runtime) *goja.Object {
	v := gr.NewObject()
	must("binding Irc.Connect", v.Set("Connect", (&p.irc).Connect))
	must("binding Irc.Disconnect", v.Set("Disconnect", (&p.irc).Disconnect))
	must("binding Irc.Privmsg", v.Set("Privmsg", (&p.irc).Privmsg))
	must("binding Irc.Nick", v.Set("Nick", (&p.irc).Nick))
	must("binding Irc.CurrentNick", v.Set("CurrentNick", (&p.irc).CurrentNick))
	must("binding Irc.Action", v.Set("Action", (&p.irc).Action))
	must("binding Irc.Join", v.Set("Join", (&p.irc).Join))
	must("binding Irc.Part", v.Set("Part", (&p.irc).Part))
	must("binding Irc.Raw", v.Set("Raw", (&p.irc).Raw))
	return v
}

func (p *HelperSet) httpHelper(gr *goja.Runtime) *goja.Object {
	v := gr.NewObject()
	must("binding Http.Send", v.Set("Send", func(call goja.FunctionCall) goja.Value {
		o := call.Argument(0).ToObject(gr)

		url := o.Get("url").String()
		if len(url) == 0 {
			panic(gr.NewTypeError("'url' is a required option"))
		}
		typVal := o.Get("type")
		typ := "json"
		if typVal != nil {
			typ = typVal.String()
		}
		successCb := o.Get("success")
		datVal := o.Get("data")
		dat := ""
		if datVal != nil {
			dat = datVal.String()
		}
		headerVal := o.Get("headers")
		headers := make([]string, 0)
		if headerVal != nil {
			v := headerVal.Export()
			if vs, ok := v.(string); ok {
				headers = append(headers, vs)
			} else if vs, ok := v.(map[string]interface{}); ok {
				for k, v := range vs {
					headers = append(headers, fmt.Sprintf("%s: %s", k, v))
				}
			}
		}
		go func() {
			var res string
			switch typ {
			case "post":
				res = p.http.Post(url, dat, headers...)
			default:
				res = p.http.Get(url, headers...)
			}
			if cb, ok := goja.AssertFunction(successCb); ok {
				p.vm.Do(func(r *goja.Runtime) {
					_, err := cb(nil, r.ToValue(res))
					must("running Http.Send callback", err)
				})
			}
		}()

		return goja.Undefined()
	}))
	must("binding Http.Get", v.Set("Get", (&p.http).Get))
	must("binding Http.Post", v.Set("Post", (&p.http).Post))
	return v
}

func (p *HelperSet) fileHelper(gr *goja.Runtime) *goja.Object {
	v := gr.NewObject()
	must("binding File.ReadAll", v.Set("ReadAll", func(call goja.FunctionCall) goja.Value {
		res, err := p.file.ReadAll(call.Argument(0).String())
		if err != nil {
			panic(gr.NewGoError(err))
		}
		return gr.ToValue(res)
	}))
	must("binding File.readAll", v.Set("readAll", v.Get("ReadAll")))
	return v
}
