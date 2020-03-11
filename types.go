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

	defaultErrorEncoder = func(ctx context.Context, err error) interface{} {
		return err.Error()
	}

	defaultResponseEncoder = func(ctx context.Context, payload interface{}) interface{} {
		return payload
	}
)

type contextValuer func(ctx context.Context, r *http.Request) (reflect.Value, error)

// BenchmarkIsBuiltinType-8   	100000000	        23.1 ns/op	       0 B/op	       0 allocs/op
var supportTypes = []interface{}{
	bodyValuer,        // request.Body
	headerValuer,      // request.Header
	formValuer,        // request.Form
	postFromValuer,    // request.PostFrom
	formPtrValuer,     // request.Form
	postFromPtrValuer, // request.PostFrom
	urlValuer,         // request.URL
	multipartValuer,   // request.MultipartForm
	requestValuer,     // raw request
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

func bodyValuer(_ context.Context, r *http.Request) (io.ReadCloser, error) {
	return r.Body, nil
}

func urlValuer(_ context.Context, r *http.Request) (*url.URL, error) {
	return r.URL, nil
}

func headerValuer(_ context.Context, r *http.Request) (http.Header, error) {
	return r.Header, nil
}

func multipartValuer(_ context.Context, r *http.Request) (*multipart.Form, error) {
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		return nil, err
	}
	return r.MultipartForm, nil
}

func formValuer(_ context.Context, r *http.Request) (Form, error) {
	err := r.ParseForm()
	if err != nil {
		return Form{}, nil
	}
	return Form{uniform{r.Form}}, nil
}

func formPtrValuer(_ context.Context, r *http.Request) (*Form, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, nil
	}
	return &Form{uniform{r.Form}}, nil
}

func postFromValuer(_ context.Context, r *http.Request) (PostForm, error) {
	err := r.ParseForm()
	if err != nil {
		return PostForm{}, nil
	}
	return PostForm{uniform{r.PostForm}}, nil
}

func postFromPtrValuer(_ context.Context, r *http.Request) (*PostForm, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, nil
	}
	return &PostForm{uniform{r.PostForm}}, nil
}

func requestValuer(_ context.Context, r *http.Request) (*http.Request, error) {
	return r, nil
}
