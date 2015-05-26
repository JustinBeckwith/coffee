// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import (
	"html/template"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/gin-gonic/gin/render"
)

const Version = "v1.0rc1"

var default404Body = []byte("404 page not found")
var default405Body = []byte("405 method not allowed")

type (
	HandlerFunc   func(*Context)
	HandlersChain []HandlerFunc

	// Represents the web framework, it wraps the blazing fast httprouter multiplexer and a list of global middlewares.
	Engine struct {
		RouterGroup
		HTMLRender  render.HTMLRender
		pool        sync.Pool
		allNoRoute  HandlersChain
		allNoMethod HandlersChain
		noRoute     HandlersChain
		noMethod    HandlersChain
		trees       map[string]*node

		// Enables automatic redirection if the current route can't be matched but a
		// handler for the path with (without) the trailing slash exists.
		// For example if /foo/ is requested but a route only exists for /foo, the
		// client is redirected to /foo with http status code 301 for GET requests
		// and 307 for all other request methods.
		RedirectTrailingSlash bool

		// If enabled, the router tries to fix the current request path, if no
		// handle is registered for it.
		// First superfluous path elements like ../ or // are removed.
		// Afterwards the router does a case-insensitive lookup of the cleaned path.
		// If a handle can be found for this route, the router makes a redirection
		// to the corrected path with status code 301 for GET requests and 307 for
		// all other request methods.
		// For example /FOO and /..//Foo could be redirected to /foo.
		// RedirectTrailingSlash is independent of this option.
		RedirectFixedPath bool

		// If enabled, the router checks if another method is allowed for the
		// current route, if the current request can not be routed.
		// If this is the case, the request is answered with 'Method Not Allowed'
		// and HTTP status code 405.
		// If no other Method is allowed, the request is delegated to the NotFound
		// handler.
		HandleMethodNotAllowed bool
	}
)

// Returns a new blank Engine instance without any middleware attached.
// The most basic configuration
func New() *Engine {
	debugPrintWARNING()
	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			BasePath: "/",
		},
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
		trees: make(map[string]*node),
	}
	engine.RouterGroup.engine = engine
	engine.pool.New = func() interface{} {
		return engine.allocateContext()
	}
	return engine
}

// Returns a Engine instance with the Logger and Recovery already attached.
func Default() *Engine {
	engine := New()
	engine.Use(Recovery(), Logger())
	return engine
}

func (engine *Engine) allocateContext() (context *Context) {
	return &Context{engine: engine}
}

func (engine *Engine) LoadHTMLGlob(pattern string) {
	if IsDebugging() {
		engine.HTMLRender = render.HTMLDebug{Glob: pattern}
	} else {
		templ := template.Must(template.ParseGlob(pattern))
		engine.SetHTMLTemplate(templ)
	}
}

func (engine *Engine) LoadHTMLFiles(files ...string) {
	if IsDebugging() {
		engine.HTMLRender = render.HTMLDebug{Files: files}
	} else {
		templ := template.Must(template.ParseFiles(files...))
		engine.SetHTMLTemplate(templ)
	}
}

func (engine *Engine) SetHTMLTemplate(templ *template.Template) {
	engine.HTMLRender = render.HTMLProduction{Template: templ}
}

// Adds handlers for NoRoute. It return a 404 code by default.
func (engine *Engine) NoRoute(handlers ...HandlerFunc) {
	engine.noRoute = handlers
	engine.rebuild404Handlers()
}

func (engine *Engine) NoMethod(handlers ...HandlerFunc) {
	engine.noMethod = handlers
	engine.rebuild405Handlers()
}

func (engine *Engine) Use(middlewares ...HandlerFunc) {
	engine.RouterGroup.Use(middlewares...)
	engine.rebuild404Handlers()
	engine.rebuild405Handlers()
}

func (engine *Engine) rebuild404Handlers() {
	engine.allNoRoute = engine.combineHandlers(engine.noRoute)
}

func (engine *Engine) rebuild405Handlers() {
	engine.allNoMethod = engine.combineHandlers(engine.noMethod)
}

func (engine *Engine) addRoute(method, path string, handlers HandlersChain) {
	debugPrintRoute(method, path, handlers)

	if path[0] != '/' {
		panic("path must begin with '/'")
	}
	if method == "" {
		panic("HTTP method can not be empty")
	}
	if len(handlers) == 0 {
		panic("there must be at least one handler")
	}

	root := engine.trees[method]
	if root == nil {
		root = new(node)
		engine.trees[method] = root
	}
	root.addRoute(path, handlers)
}

func (engine *Engine) Run(addr string) (err error) {
	debugPrint("Listening and serving HTTP on %s\n", addr)
	defer func() { debugPrintError(err) }()

	err = http.ListenAndServe(addr, engine)
	return
}

func (engine *Engine) RunTLS(addr string, cert string, key string) (err error) {
	debugPrint("Listening and serving HTTPS on %s\n", addr)
	defer func() { debugPrintError(err) }()

	err = http.ListenAndServe(addr, engine)
	return
}

func (engine *Engine) RunUnix(file string) (err error) {
	debugPrint("Listening and serving HTTP on unix:/%s", file)
	defer func() { debugPrintError(err) }()

	os.Remove(file)
	listener, err := net.Listen("unix", file)
	if err != nil {
		return
	}
	defer listener.Close()
	err = http.Serve(listener, engine)
	return
}

// ServeHTTP makes the router implement the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := engine.getcontext(w, req)
	engine.handleHTTPRequest(c)
	engine.putcontext(c)
}

func (engine *Engine) getcontext(w http.ResponseWriter, req *http.Request) *Context {
	c := engine.pool.Get().(*Context)
	c.writermem.reset(w)
	c.Request = req
	c.reset()
	return c
}

func (engine *Engine) putcontext(c *Context) {
	engine.pool.Put(c)
}

func (engine *Engine) handleHTTPRequest(context *Context) {
	httpMethod := context.Request.Method
	path := context.Request.URL.Path

	// Find root of the tree for the given HTTP method
	if root := engine.trees[httpMethod]; root != nil {
		// Find route in tree
		handlers, params, tsr := root.getValue(path, context.Params)
		if handlers != nil {
			context.handlers = handlers
			context.Params = params
			context.Next()
			context.writermem.WriteHeaderNow()
			return

		} else if httpMethod != "CONNECT" && path != "/" {
			if engine.serveAutoRedirect(context, root, tsr) {
				return
			}
		}
	}

	if engine.HandleMethodNotAllowed {
		for method, root := range engine.trees {
			if method != httpMethod {
				if handlers, _, _ := root.getValue(path, nil); handlers != nil {
					context.handlers = engine.allNoMethod
					serveError(context, 405, default405Body)
					return
				}
			}
		}
	}
	context.handlers = engine.allNoRoute
	serveError(context, 404, default404Body)
}

func (engine *Engine) serveAutoRedirect(c *Context, root *node, tsr bool) bool {
	req := c.Request
	path := req.URL.Path
	code := 301 // Permanent redirect, request with GET method
	if req.Method != "GET" {
		code = 307
	}

	if tsr && engine.RedirectTrailingSlash {
		if len(path) > 1 && path[len(path)-1] == '/' {
			req.URL.Path = path[:len(path)-1]
		} else {
			req.URL.Path = path + "/"
		}
		debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
		http.Redirect(c.Writer, req, req.URL.String(), code)
		c.writermem.WriteHeaderNow()
		return true
	}

	// Try to fix the request path
	if engine.RedirectFixedPath {
		fixedPath, found := root.findCaseInsensitivePath(
			CleanPath(path),
			engine.RedirectTrailingSlash,
		)
		if found {
			req.URL.Path = string(fixedPath)
			debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
			http.Redirect(c.Writer, req, req.URL.String(), code)
			c.writermem.WriteHeaderNow()
			return true
		}
	}
	return false
}

func serveError(c *Context, code int, defaultMessage []byte) {
	c.writermem.status = code
	c.Next()
	if !c.Writer.Written() {
		if c.Writer.Status() == code {
			c.Data(-1, binding.MIMEPlain, defaultMessage)
		} else {
			c.Writer.WriteHeaderNow()
		}
	}
}
