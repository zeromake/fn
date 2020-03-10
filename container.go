package fn

import (
	"context"
	"net/http"
	"reflect"
)

type (
	supportType map[reflect.Type]contextValuer
	Container   struct {
		plugins         []PluginFunc
		supportTypes    supportType
		errorEncoder    ErrorEncoder
		responseEncoder ResponseEncoder
	}
)

func (s supportType) clone() supportType {
	n := make(supportType, len(s))
	for k, v := range s {
		n[k] = v
	}
	return n
}

func (c *Container) New() *Container {
	return New()
}

func (c *Container) Clone() *Container {
	return &Container{
		plugins:         c.plugins[:],
		supportTypes:    c.supportTypes.clone(),
		responseEncoder: c.responseEncoder,
		errorEncoder:    c.errorEncoder,
	}
}

func (c *Container) Wrap(f interface{}) Fn {
	t := reflect.TypeOf(f)
	var (
		adapter          adapter
		numIn, inContext = wrapCheckType(t)
	)

	if numIn == 0 {
		// func() (Response, error)
		adapter = &simplePlainAdapter{
			inContext: false,
			method:    reflect.ValueOf(f),
			cacheArgs: []reflect.Value{},
		}
	} else if numIn == 1 && inContext {
		// func(ctx context.Context) (Response, error)
		adapter = &simplePlainAdapter{
			inContext: true,
			method:    reflect.ValueOf(f),
			cacheArgs: make([]reflect.Value, 1),
		}
	} else if numIn == 1 && !c.isBuiltinType(t.In(0)) && t.In(0).Kind() == reflect.Ptr {
		// func(request *Customized) (Response, error)
		adapter = &simpleUnaryAdapter{
			argType:   t.In(0),
			method:    reflect.ValueOf(f),
			cacheArgs: make([]reflect.Value, 1),
		}
	} else {
		// Complicated signatures
		//
		// e.g:
		// type LoginResponse {...}
		// type LoginRequest {...}
		//
		// func (header http.Header) (*LoginResponse, error) {}
		// func (form fn.Form) (*LoginResponse, error) {}
		// func (header http.Header, form fn.Form, body io.ReadCloser) (*LoginResponse, error) {}
		// func (header http.Header, r *LoginRequest, url *url.URL) (*LoginResponse, error) { }
		adapter = makeGenericAdapter(c, reflect.ValueOf(f), inContext)
	}

	return &fn{container: c, adapter: adapter}
}

func (c *Container) Plugin(before ...PluginFunc) *Container {
	for _, b := range before {
		if b != nil {
			c.plugins = append(c.plugins, b)
		}
	}
	return c
}

// buildSupportTypesFunc 生成对应
func buildSupportTypesFunc(vv reflect.Value) contextValuer {
	return func(ctx context.Context, r *http.Request) (value reflect.Value, err error) {
		v := vv.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(r)})
		if v[1].IsNil() {
			return v[0], nil
		}
		return v[0], v[1].Interface().(error)
	}
}

func (c *Container) requestPlugin(p interface{}) *Container {
	if p == nil {
		return c
	}
	vv := reflect.ValueOf(p)
	t := vv.Type()
	if t.Kind() != reflect.Func || t.NumOut() != 2 || t.NumIn() != 2 {
		panic("request plugin is func (ctx context.Context, r *http.Request) ({{data}}, error)")
	}
	switch {
	case t.In(0) != contextType, t.In(1) != requestType:
		panic("param must is context.Context, *http.Request")
	case t.Out(1) != errorType:
		panic("return must is {{data}}, error")
	}
	out := t.Out(0)
	f := buildSupportTypesFunc(vv)
	c.supportTypes[out] = f
	return c
}

func (c *Container) RequestPlugin(plugins ...interface{}) *Container {
	for _, p := range plugins {
		c.requestPlugin(p)
	}
	return c
}

func (c *Container) builtinType(t reflect.Type) (contextValuer, bool) {
	v, ok := c.supportTypes[t]
	return v, ok
}

func (c *Container) isBuiltinType(t reflect.Type) bool {
	_, ok := c.supportTypes[t]
	return ok
}

func (c *Container) SetErrorEncoder(e ErrorEncoder) {
	if e == nil {
		panic("nil pointer to error encoder")
	}
	c.errorEncoder = e
}

// SetResponseEncoder set respone encoder
func (c *Container) SetResponseEncoder(r ResponseEncoder) {
	if r == nil {
		panic("nil pointer to error encoder")
	}
	c.responseEncoder = r
}

// NewGroup 以继承模式新建容器
func NewGroup() *Container {
	return globalContainer.Clone()
}

// New 新建一个空白容器
func New() *Container {
	return &Container{
		supportTypes:    supportType{},
		responseEncoder: defaultResponseEncoder,
		errorEncoder:    defaultErrorEncoder,
	}
}
