// Copyright 2017 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package fn

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
)

// adapter represents a container that contain a handler function
// and convert a it to a http.Handler
type adapter interface {
	invoke(context.Context, http.ResponseWriter, *http.Request) (interface{}, error)
	clone(c *Container) adapter
}

// genericAdapter represents a common adapter
type genericAdapter struct {
	container *Container
	inContext bool
	method    reflect.Value
	numIn     int
	types     []reflect.Type
	cacheArgs []reflect.Value // cache args
}

// Accept zero parameter adapter
type simplePlainAdapter struct {
	inContext bool
	method    reflect.Value
	cacheArgs []reflect.Value
}

// Accept only one parameter adapter
type simpleUnaryAdapter struct {
	//outContext bool
	argType   reflect.Type
	method    reflect.Value
	cacheArgs []reflect.Value // cache args
}

func makeGenericAdapter(c *Container, method reflect.Value, inContext bool) *genericAdapter {
	var noSupportExists = false
	t := method.Type()
	numIn := t.NumIn()

	a := &genericAdapter{
		container: c,
		inContext: inContext,
		method:    method,
		numIn:     numIn,
		types:     make([]reflect.Type, numIn),
		cacheArgs: make([]reflect.Value, numIn),
	}

	for i := 0; i < numIn; i++ {
		in := t.In(i)
		if in != contextType && !a.container.isBuiltinType(in) {
			if noSupportExists {
				panic("function should accept only one customize type")
			}

			if in.Kind() != reflect.Ptr {
				panic("customize type should be a pointer(" + in.PkgPath() + "." + in.Name() + ")")
			}
			noSupportExists = true
		}
		a.types[i] = in
	}

	return a
}

// invokeParams params handler
func (a *genericAdapter) invokeParams(ctx context.Context, r *http.Request) ([]reflect.Value, error) {
	var (
		values = a.cacheArgs
		value  reflect.Value
		err    error
	)
	for i := 0; i < a.numIn; i++ {
		typ := a.types[i]
		v, ok := a.container.builtinType(typ)
		if ok {
			// support type param
			value, err = v(ctx, r)
		} else if typ == contextType {
			// context type param
			value = reflect.ValueOf(ctx)
		} else {
			// *struct
			d := reflect.New(typ.Elem()).Interface()
			err = json.NewDecoder(r.Body).Decode(d)
			if err == nil {
				value = reflect.ValueOf(d)
			}
		}
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	return values, nil
}

func (a *genericAdapter) clone(container *Container) adapter {
	return &genericAdapter{
		container: container,
		inContext: a.inContext,
		method:    a.method,
		numIn:     a.numIn,
		types:     a.types[:],
		cacheArgs: a.cacheArgs[:],
	}
}

func (a *genericAdapter) invoke(ctx context.Context, _ http.ResponseWriter, r *http.Request) (interface{}, error) {
	values, err := a.invokeParams(ctx, r)

	results := a.method.Call(values)
	payload := results[0].Interface()
	if e := results[1].Interface(); e != nil {
		err = e.(error)
	}
	return payload, err
}

func (a *simplePlainAdapter) invoke(ctx context.Context, _ http.ResponseWriter, _ *http.Request) (interface{}, error) {
	if a.inContext {
		a.cacheArgs[0] = reflect.ValueOf(ctx)
	}

	var err error
	results := a.method.Call(a.cacheArgs)
	payload := results[0].Interface()
	if e := results[1].Interface(); e != nil {
		err = e.(error)
	}
	return payload, err
}

func (a *simplePlainAdapter) clone(_ *Container) adapter {
	return &simplePlainAdapter{
		inContext: a.inContext,
		method:    a.method,
		cacheArgs: a.cacheArgs[:],
	}
}

func (a *simpleUnaryAdapter) invoke(_ context.Context, _ http.ResponseWriter, r *http.Request) (interface{}, error) {
	data := reflect.New(a.argType.Elem()).Interface()
	err := json.NewDecoder(r.Body).Decode(data)
	if err != nil {
		return nil, err
	}

	a.cacheArgs[0] = reflect.ValueOf(data)
	results := a.method.Call(a.cacheArgs)
	payload := results[0].Interface()
	if e := results[1].Interface(); e != nil {
		err = e.(error)
	}
	return payload, err
}

func (a *simpleUnaryAdapter) clone(_ *Container) adapter {
	return &simpleUnaryAdapter{
		argType:   a.argType,
		method:    a.method,
		cacheArgs: a.cacheArgs[:],
	}
}
