// Copyright 2013 Martini Authors
// Copyright 2014 The Web Authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package web

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"
)

var (
	ColorLog      = false
	LogTimeFormat = "2006-01-02 15:04:05"
)

// LoggerInvoker is an inject.FastInvoker wrapper of func(ctx *Context, log *log.Logger).
type LoggerInvoker func(ctx *Context, log *log.Logger)

func (invoke LoggerInvoker) Invoke(params []interface{}) ([]reflect.Value, error) {
	invoke(params[0].(*Context), params[1].(*log.Logger))
	return nil, nil
}

// Logger returns a middleware handler that logs the request as it goes in and the response as it goes out.
func Logger() Handler {
	return func(ctx *Context, log *log.Logger) {
		start := time.Now()
		rw := ctx.Resp.(ResponseWriter)
		ctx.Next()
		content := fmt.Sprintf("%s %s %v %s %v", ctx.Req.Method, ctx.Req.RequestURI, rw.Status(), http.StatusText(rw.Status()), time.Since(start))
		log.Println(content)
	}
}
