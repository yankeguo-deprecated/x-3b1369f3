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
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"landzero.net/x/com"
	"landzero.net/x/net/web/inject"
)

// Locale reprents a localization interface.
type Locale interface {
	Language() string
	Tr(string, ...interface{}) string
}

// RequestBody represents a request body.
type RequestBody struct {
	reader io.ReadCloser
}

// Bytes reads and returns content of request body in bytes.
func (rb *RequestBody) Bytes() ([]byte, error) {
	return ioutil.ReadAll(rb.reader)
}

// String reads and returns content of request body in string.
func (rb *RequestBody) String() (string, error) {
	data, err := rb.Bytes()
	return string(data), err
}

// ReadCloser returns a ReadCloser for request body.
func (rb *RequestBody) ReadCloser() io.ReadCloser {
	return rb.reader
}

// Request represents an HTTP request received by a server or to be sent by a client.
type Request struct {
	*http.Request
}

func (r *Request) Body() *RequestBody {
	return &RequestBody{r.Request.Body}
}

// ContextInvoker is an inject.FastInvoker wrapper of func(c *Context).
type ContextInvoker func(c *Context)

func (invoke ContextInvoker) Invoke(params []interface{}) ([]reflect.Value, error) {
	invoke(params[0].(*Context))
	return nil, nil
}

// Context represents the runtime context of current request of Web instance.
// It is the integration of most frequently used middlewares and helper methods.
type Context struct {
	env string
	inject.Injector
	handlers []Handler
	action   Handler
	index    int

	*Router
	Req    Request
	Resp   ResponseWriter
	params Params
	Render
	Locale
	Data   map[string]interface{}
	crid   string
	logger *log.Logger
}

func (c *Context) handler() Handler {
	if c.index < len(c.handlers) {
		return c.handlers[c.index]
	}
	if c.index == len(c.handlers) {
		return c.action
	}
	panic("invalid index for context handler")
}

func (c *Context) Crid() string {
	return c.crid
}

func (c *Context) CridMark() string {
	return "CRID[" + c.Crid() + "]"
}

func (c *Context) Next() {
	c.index++
	c.run()
}

// Written Resp.Written
func (c *Context) Written() bool {
	return c.Resp.Written()
}

func (c *Context) run() {
	for c.index <= len(c.handlers) {
		vals, err := c.Invoke(c.handler())
		if err != nil {
			panic(err)
		}
		c.index++

		// if the handler returned something, write it to the http response
		if len(vals) > 0 {
			ev := c.GetVal(reflect.TypeOf(ReturnHandler(nil)))
			handleReturn := ev.Interface().(ReturnHandler)
			handleReturn(c, vals)
		}

		if c.Written() {
			return
		}
	}
}

// RemoteAddr returns more real IP address.
func (c *Context) RemoteAddr() string {
	addr := c.Req.Header.Get("X-Real-IP")
	if len(addr) == 0 {
		addr = c.Req.Header.Get("X-Forwarded-For")
		if addr == "" {
			addr = c.Req.RemoteAddr
			if i := strings.LastIndex(addr, ":"); i > -1 {
				addr = addr[:i]
			}
		}
	}
	return addr
}

func (c *Context) renderHTML(status int, setName, tplName string, data ...interface{}) {
	if len(data) <= 0 {
		c.Render.HTMLSet(status, setName, tplName, c.Data)
	} else if len(data) == 1 {
		c.Render.HTMLSet(status, setName, tplName, data[0])
	} else {
		c.Render.HTMLSet(status, setName, tplName, data[0], data[1].(HTMLOptions))
	}
}

// HTML calls Render.HTML but allows less arguments.
func (c *Context) HTML(status int, name string, data ...interface{}) {
	c.renderHTML(status, DEFAULT_TPL_SET_NAME, name, data...)
}

// HTML calls Render.HTMLSet but allows less arguments.
func (c *Context) HTMLSet(status int, setName, tplName string, data ...interface{}) {
	c.renderHTML(status, setName, tplName, data...)
}

func (c *Context) Redirect(location string, status ...int) {
	code := http.StatusFound
	if len(status) == 1 {
		code = status[0]
	}

	http.Redirect(c.Resp, c.Req.Request, location, code)
}

// Maximum amount of memory to use when parsing a multipart form.
// Set this to whatever value you prefer; default is 10 MB.
var MaxMemory = int64(1024 * 1024 * 10)

func (c *Context) parseForm() {
	if c.Req.Form != nil {
		return
	}

	contentType := c.Req.Header.Get(_CONTENT_TYPE)
	if (c.Req.Method == "POST" || c.Req.Method == "PUT") &&
		len(contentType) > 0 && strings.Contains(contentType, "multipart/form-data") {
		c.Req.ParseMultipartForm(MaxMemory)
	} else {
		c.Req.ParseForm()
	}
}

// Query querys form parameter.
func (c *Context) Query(name string) string {
	c.parseForm()
	return c.Req.Form.Get(name)
}

// QueryTrim querys and trims spaces form parameter.
func (c *Context) QueryTrim(name string) string {
	return strings.TrimSpace(c.Query(name))
}

// QueryStrings returns a list of results by given query name.
func (c *Context) QueryStrings(name string) []string {
	c.parseForm()

	vals, ok := c.Req.Form[name]
	if !ok {
		return []string{}
	}
	return vals
}

// QueryEscape returns escapred query result.
func (c *Context) QueryEscape(name string) string {
	return template.HTMLEscapeString(c.Query(name))
}

// QueryBool returns query result in bool type.
func (c *Context) QueryBool(name string) bool {
	v, _ := strconv.ParseBool(c.Query(name))
	return v
}

// QueryInt returns query result in int type.
func (c *Context) QueryInt(name string) int {
	return com.StrTo(c.Query(name)).MustInt()
}

// QueryInt64 returns query result in int64 type.
func (c *Context) QueryInt64(name string) int64 {
	return com.StrTo(c.Query(name)).MustInt64()
}

// QueryFloat64 returns query result in float64 type.
func (c *Context) QueryFloat64(name string) float64 {
	v, _ := strconv.ParseFloat(c.Query(name), 64)
	return v
}

// Params returns value of given param name.
// e.g. c.Params(":uid") or c.Params("uid")
func (c *Context) Params(name string) string {
	if len(name) == 0 {
		return ""
	}
	if len(name) > 1 && name[0] != ':' {
		name = ":" + name
	}
	return c.params[name]
}

// SetParams sets value of param with given name.
func (c *Context) SetParams(name, val string) {
	if !strings.HasPrefix(name, ":") {
		name = ":" + name
	}
	c.params[name] = val
}

// ReplaceAllParams replace all current params with given params
func (c *Context) ReplaceAllParams(params Params) {
	c.params = params
}

// ParamsEscape returns escapred params result.
// e.g. c.ParamsEscape(":uname")
func (c *Context) ParamsEscape(name string) string {
	return template.HTMLEscapeString(c.Params(name))
}

// ParamsInt returns params result in int type.
// e.g. c.ParamsInt(":uid")
func (c *Context) ParamsInt(name string) int {
	return com.StrTo(c.Params(name)).MustInt()
}

// ParamsInt64 returns params result in int64 type.
// e.g. c.ParamsInt64(":uid")
func (c *Context) ParamsInt64(name string) int64 {
	return com.StrTo(c.Params(name)).MustInt64()
}

// ParamsFloat64 returns params result in int64 type.
// e.g. c.ParamsFloat64(":uid")
func (c *Context) ParamsFloat64(name string) float64 {
	v, _ := strconv.ParseFloat(c.Params(name), 64)
	return v
}

// GetFile returns information about user upload file by given form field name.
func (c *Context) GetFile(name string) (multipart.File, *multipart.FileHeader, error) {
	return c.Req.FormFile(name)
}

// SaveToFile reads a file from request by field name and saves to given path.
func (c *Context) SaveToFile(name, savePath string) error {
	fr, _, err := c.GetFile(name)
	if err != nil {
		return err
	}
	defer fr.Close()

	fw, err := os.OpenFile(savePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer fw.Close()

	_, err = io.Copy(fw, fr)
	return err
}

// SetCookie sets given cookie value to response header.
// FIXME: IE support? http://golanghome.com/post/620#reply2
func (c *Context) SetCookie(name string, value string, others ...interface{}) {
	cookie := http.Cookie{}
	cookie.Name = name
	cookie.Value = url.QueryEscape(value)

	if len(others) > 0 {
		switch v := others[0].(type) {
		case int:
			cookie.MaxAge = v
		case int64:
			cookie.MaxAge = int(v)
		case int32:
			cookie.MaxAge = int(v)
		}
	}

	cookie.Path = "/"
	if len(others) > 1 {
		if v, ok := others[1].(string); ok && len(v) > 0 {
			cookie.Path = v
		}
	}

	if len(others) > 2 {
		if v, ok := others[2].(string); ok && len(v) > 0 {
			cookie.Domain = v
		}
	}

	if len(others) > 3 {
		switch v := others[3].(type) {
		case bool:
			cookie.Secure = v
		default:
			if others[3] != nil {
				cookie.Secure = true
			}
		}
	}

	if len(others) > 4 {
		if v, ok := others[4].(bool); ok && v {
			cookie.HttpOnly = true
		}
	}

	if len(others) > 5 {
		if v, ok := others[5].(time.Time); ok {
			cookie.Expires = v
			cookie.RawExpires = v.Format(time.UnixDate)
		}
	}

	c.Resp.Header().Add("Set-Cookie", cookie.String())
}

// GetCookie returns given cookie value from request header.
func (c *Context) GetCookie(name string) string {
	cookie, err := c.Req.Cookie(name)
	if err != nil {
		return ""
	}
	val, _ := url.QueryUnescape(cookie.Value)
	return val
}

// GetCookieInt returns cookie result in int type.
func (c *Context) GetCookieInt(name string) int {
	return com.StrTo(c.GetCookie(name)).MustInt()
}

// GetCookieInt64 returns cookie result in int64 type.
func (c *Context) GetCookieInt64(name string) int64 {
	return com.StrTo(c.GetCookie(name)).MustInt64()
}

// GetCookieFloat64 returns cookie result in float64 type.
func (c *Context) GetCookieFloat64(name string) float64 {
	v, _ := strconv.ParseFloat(c.GetCookie(name), 64)
	return v
}

func (c *Context) setRawContentHeader() {
	c.Resp.Header().Set("Content-Description", "Raw content")
	c.Resp.Header().Set("Content-Type", "text/plain")
	c.Resp.Header().Set("Expires", "0")
	c.Resp.Header().Set("Cache-Control", "must-revalidate")
	c.Resp.Header().Set("Pragma", "public")
}

// ServeContent serves given content to response.
func (c *Context) ServeContent(name string, r io.ReadSeeker, params ...interface{}) {
	modtime := time.Now()
	for _, p := range params {
		switch v := p.(type) {
		case time.Time:
			modtime = v
		}
	}

	c.setRawContentHeader()
	http.ServeContent(c.Resp, c.Req.Request, name, modtime, r)
}

// ServeFileContent serves given file as content to response.
func (c *Context) ServeFileContent(file string, names ...string) {
	var name string
	if len(names) > 0 {
		name = names[0]
	} else {
		name = path.Base(file)
	}

	f, err := os.Open(file)
	if err != nil {
		if c.env == PROD {
			http.Error(c.Resp, "Internal Server Error", 500)
		} else {
			http.Error(c.Resp, err.Error(), 500)
		}
		return
	}
	defer f.Close()

	c.setRawContentHeader()
	http.ServeContent(c.Resp, c.Req.Request, name, time.Now(), f)
}

// ServeFile serves given file to response.
func (c *Context) ServeFile(file string, names ...string) {
	var name string
	if len(names) > 0 {
		name = names[0]
	} else {
		name = path.Base(file)
	}
	c.Resp.Header().Set("Content-Description", "File Transfer")
	c.Resp.Header().Set("Content-Type", "application/octet-stream")
	c.Resp.Header().Set("Content-Disposition", "attachment; filename="+name)
	c.Resp.Header().Set("Content-Transfer-Encoding", "binary")
	c.Resp.Header().Set("Expires", "0")
	c.Resp.Header().Set("Cache-Control", "must-revalidate")
	c.Resp.Header().Set("Pragma", "public")
	http.ServeFile(c.Resp, c.Req.Request, file)
}

// ChangeStaticPath changes static path from old to new one.
func (c *Context) ChangeStaticPath(oldPath, newPath string) {
	if !filepath.IsAbs(oldPath) {
		oldPath = filepath.Join(Root, oldPath)
	}
	dir := statics.Get(oldPath)
	if dir != nil {
		statics.Delete(oldPath)

		if !filepath.IsAbs(newPath) {
			newPath = filepath.Join(Root, newPath)
		}
		*dir = http.Dir(newPath)
		statics.Set(dir)
	}
}

// Printf log.Printf with crid prefixed
func (c *Context) Printf(format string, v ...interface{}) {
	c.logger.Output(2, c.CridMark()+" "+fmt.Sprintf(format, v...))
}

// Println log.Printf with crid prefixed
func (c *Context) Println(v ...interface{}) {
	c.logger.Output(2, c.CridMark()+" "+fmt.Sprintln(v...))
}

// Print log.print with crid prefixed
func (c *Context) Print(v ...interface{}) {
	c.logger.Output(2, c.CridMark()+" "+fmt.Sprint(v...))
}
