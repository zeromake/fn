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
	"net/http"
	"reflect"
)

var (
	globalContainer = &Container{
		supportTypes:        supportTypes,
	}
)

// Fn handler interface
type Fn interface {
	http.Handler
	Plugin(before ...PluginFunc) *fn
}

func wrapCheckType(t reflect.Type) (int, bool) {
	if t.Kind() != reflect.Func {
		panic("fn only support wrap a function to http.Handler")
	}

	numOut := t.NumOut()

	// Supported signatures
	// func(...) (Response, error)
	if numOut != 2 {
		panic("unsupported function type, function return values should contain response data & error")
	}

	var (
		numIn     = t.NumIn()
		inContext = false
	)

	if numIn > 0 {
		for i := 0; i < numIn; i++ {
			// Legal: func(ctx context.Context, ...) ...
			if t.In(i) == contextType {
				// Illegal: func(..., ctx context.Context, ...) ...
				if i != 0 {
					panic("the `context.Context` must be the first parameter if the signature contains `context.Context`")
				}
				inContext = true
			}
		}
	}
	return numIn, inContext
}

// Wrap wrap handler
func Wrap(f interface{}) Fn {
	return globalContainer.Wrap(f)
}

// SetErrorEncoder set error respone encoder
func SetErrorEncoder(c ErrorEncoder) {
	if c == nil {
		panic("nil pointer to error encoder")
	}
	errorEncoder = c
}

// SetResponseEncoder set respone encoder
func SetResponseEncoder(c ResponseEncoder) {
	if c == nil {
		panic("nil pointer to error encoder")
	}
	responseEncoder = c
}

// SetMultipartFormMaxMemory set multipart max memory
func SetMultipartFormMaxMemory(m int64) {
	maxMemory = m
}

// RequestPlugin set request plugin func (ctx context.Context, r *http.Request) ({{data}}, error)
func RequestPlugin(p interface{}) *Container {
	return globalContainer.RequestPlugin(p)
}
