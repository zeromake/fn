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
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
)

//type valuer func(r *http.Request) (reflect.Value, error)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	requestType = reflect.TypeOf((*http.Request)(nil))
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

type contextValuer func(ctx context.Context, r *http.Request) (reflect.Value, error)

// BenchmarkIsBuiltinType-8   	100000000	        23.1 ns/op	       0 B/op	       0 allocs/op
var supportTypes = map[reflect.Type]contextValuer{
	reflect.TypeOf((*io.ReadCloser)(nil)).Elem(): bodyValuer,        // request.Body
	reflect.TypeOf((http.Header)(nil)):           headerValuer,      // request.Header
	reflect.TypeOf(Form{}):                       formValuer,        // request.Form
	reflect.TypeOf(PostForm{}):                   postFromValuer,    // request.PostFrom
	reflect.TypeOf((*Form)(nil)):                 formPtrValuer,     // request.Form
	reflect.TypeOf((*PostForm)(nil)):             postFromPtrValuer, // request.PostFrom
	reflect.TypeOf((*url.URL)(nil)):              urlValuer,         // request.URL
	reflect.TypeOf((*multipart.Form)(nil)):       multipartValuer,   // request.MultipartForm
	reflect.TypeOf((*http.Request)(nil)):         requestValuer,     // raw request
}

//var supportRequestTypes = map[reflect.Type]contextValuer{}

var maxMemory = int64(2 * 1024 * 1024)

type uniform struct {
	url.Values
}

// Form parse `request.Form`
type Form struct {
	uniform
}

// PostForm parse `request.PostForm`
type PostForm struct {
	uniform
}

func bodyValuer(_ context.Context, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.Body), nil
}

func urlValuer(_ context.Context, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.URL), nil
}

func headerValuer(_ context.Context, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.Header), nil
}

func multipartValuer(_ context.Context, r *http.Request) (reflect.Value, error) {
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(r.MultipartForm), nil
}

func formValuer(_ context.Context, r *http.Request) (reflect.Value, error) {
	err := r.ParseForm()
	if err != nil {
		return reflect.Value{}, nil
	}
	return reflect.ValueOf(Form{uniform{r.Form}}), nil
}

func postFromValuer(_ context.Context, r *http.Request) (reflect.Value, error) {
	err := r.ParseForm()
	if err != nil {
		return reflect.Value{}, nil
	}
	return reflect.ValueOf(PostForm{uniform{r.PostForm}}), nil
}

func formPtrValuer(_ context.Context, r *http.Request) (reflect.Value, error) {
	err := r.ParseForm()
	if err != nil {
		return reflect.Value{}, nil
	}
	return reflect.ValueOf(&Form{uniform{r.Form}}), nil
}

func postFromPtrValuer(_ context.Context, r *http.Request) (reflect.Value, error) {
	err := r.ParseForm()
	if err != nil {
		return reflect.Value{}, nil
	}
	return reflect.ValueOf(&PostForm{uniform{r.PostForm}}), nil
}

func requestValuer(_ context.Context, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r), nil
}

//func isBuiltinType(t reflect.Type) bool {
//	_, ok := supportTypes[t]
//	return ok
//}
//
//func isRequestType(t reflect.Type) bool {
//	_, ok := supportRequestTypes[t]
//	return ok
//}
