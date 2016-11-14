package cmdlnrouter

import (
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	// ErrInvalidInterface - interface should be struct{}
	ErrInvalidInterface = errors.New("Invalid interface, it should be a struct or map[string]string")
)

// TimeoutHandlerDuration specifies how long a timeout will happen for a cmdln request
var TimeoutHandlerDuration = 2 * time.Second

type (
	valmap map[string]interface{}
	argmap map[string][]handleArgFunc

	handleArgFunc   func(out chan<- inputHandle) (args chan string, done chan struct{})
	parseArgFunc    func(<-chan string, argmap, valmap, *Context) error
	parseOptArgFunc func(<-chan string, valmap, *Context) (error, chan string)
)

// New returns a fully baked Router object
func New(p ...parseOptArgFunc) *Router {
	r := new(Router)
	r.arg.pfn = parseArgs
	if len(p) > 0 {
		r.opt.pfn = p
	}
	return r
}

type (
	argCmp struct {
		argVar, argVal string
	}
	argsInputed []argCmp
	argsHandled []string

	inputHandle struct {
		in argsInputed
		hn Handle

		er error
	}
)

func (a *argsInputed) String() string {
	ss := make([]string, len(*a))
	for i, v := range *a {
		if v.argVar != "" {
			ss[i] = ":"
			continue
		}
		ss[i] = v.argVal
	}
	return strings.Join(ss, "/")
}

func (a *argsHandled) String() string {
	ss := make([]string, len(*a))
	for i, v := range *a {
		if v[0] == ':' {
			ss[i] = ":"
			continue
		}
		ss[i] = v
	}
	// skip the first one...
	return strings.Join(ss[1:], "/")
}

func (a *argsHandled) Slice() []string {
	return (*a)[1:]
}

func (a *argsHandled) Root() string {
	return (*a)[0]
}

func getArgsHandled(s string) argsHandled {
	return strings.Fields(s)
}

func doneTimeout(done chan struct{}) {
	select {
	case <-done:
	// nothing we're just done...
	case <-time.After(TimeoutHandlerDuration):
		if _, ok := <-done; !ok {
			close(done)
		}
	}
}

// Handle is the function signature for the context f(c)
type Handle func(*Context)

// A Handler responds to an parsed commandline request.
type Handler interface {
	ServeCmdln(*Context)
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as CmdlnRouter handlers. If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc Handle

// ServeCmdln calls f(c).
func (f HandlerFunc) ServeCmdln(c *Context) { f(c) }

// Router is the main struct that is used to parse commandline options
type Router struct {

	// opt holds the Options struct. There must be exported fields for them to be catured
	opt struct {
		irf interface{}
		val valmap
		pfn []parseOptArgFunc
	}

	// arg holds the Argument struct. There must be exported fields for them to be captured
	arg struct {
		irf interface{}
		val valmap
		pfn parseArgFunc
	}

	tree struct {
		arg argmap
		hdl map[string]struct{}
	}

	Err error
}

// Arguments add the arugments struct to the router
func (r *Router) Arguments(t interface{}) *Router {
	val := reflect.ValueOf(reflect.ValueOf(t).Interface())
	switch {
	// error if the passed in interface{} not a pointer,
	// or if it is a pointer but not a struct.
	case val.Kind() != reflect.Ptr,
		val.Kind() == reflect.Ptr && val.Elem().Type().Kind() != reflect.Struct:
		r.arg.irf = nil
		r.Err = ErrInvalidInterface
		return r
	}
	// the rest happens if the passed in interface{} is a struct

	r.arg.val = make(valmap)
	r.arg.irf = t
	elm := val.Elem()

	// loop through all of the fields of the passed in struct
	for i := 0; i < elm.NumField(); i++ {
		vField := elm.Field(i)
		tField := elm.Type().Field(i)

		if !vField.CanSet() {
			// warning...
			log.Println("Unexported field:", tField.Name)
			continue
		}

		argName := strings.Split(tField.Tag.Get("cmdln"), ",")[0]
		switch argName {
		case "", "-":
			argName = tField.Name
		}
		r.arg.val[strings.ToLower(":"+argName)] = vField
	}

	return r
}

// Optional add the option struct or map[string]string to the router
func (r *Router) Optional(t interface{}) *Router {
	r.opt.val = make(valmap)

	val := reflect.ValueOf(reflect.ValueOf(t).Interface())
	switch {
	// if the passed in interface{} is a map, make sure it's
	// the correct kind of map
	case val.Type().Kind() == reflect.Map:
		if _, ok := val.Interface().(map[string]string); !ok {
			r.opt.irf = nil
			r.Err = ErrInvalidInterface
			return r
		}
		r.opt.val["*"] = t
		return r
	// error if the passed in interface{} not a pointer,
	// or if it is a pointer but not a struct.
	case val.Kind() != reflect.Ptr,
		val.Kind() == reflect.Ptr && val.Elem().Type().Kind() != reflect.Struct:
		r.opt.irf = nil
		r.Err = ErrInvalidInterface
		return r
	}
	// the rest happens if the passed in interface{} is a struct

	r.opt.irf = t
	elm := val.Elem()

	// loop through all of the fields of the passed in struct
	for i := 0; i < elm.NumField(); i++ {
		vField := elm.Field(i)
		tField := elm.Type().Field(i)

		if !vField.CanSet() {
			// warning...
			log.Println("Unexported field:", tField.Name)
			continue
		}

		tags := strings.Split(tField.Tag.Get("cmdln"), ",")
		for _, v := range tags {
			// adds the tags to a map with a pointer to the value on the struct
			switch {
			case len(v) > 1 && v[0] == '-',
				len(v) > 2 && string(v[0:2]) == "--":
				r.opt.val[v] = vField
			}
		}
	}

	return r
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (r *Router) Handle(cmdln string, handle Handle) {

	// init all of the variables needed to handle...
	ah := getArgsHandled(cmdln)
	ahs := ah.String()

	// init the maps needed to hold and check for collsions of handlers
	if r.tree.arg == nil {
		r.tree.arg = make(argmap)
	}

	if r.tree.hdl == nil {
		r.tree.hdl = make(map[string]struct{})
	}

	// check for collsions in of cmdlnroute paths
	if _, isAlreadyHandled := r.tree.hdl[ahs]; isAlreadyHandled {
		log.Fatal("There is a collsion for the handlers, they must be unique.")
	} else {
		r.tree.hdl[ahs] = struct{}{}
	}

	// set up the function that will check the cmdln arguements and handle them
	var treeFn = func(out chan<- inputHandle) (args chan string, done chan struct{}) {
		args, done = make(chan string, 1), make(chan struct{})

		go doneTimeout(done)
		go func(done chan struct{}) {
			argList, err := parseArgsHandler(ah, args)
			out <- inputHandle{in: argList, hn: handle, er: err}
			close(done)
		}(done)

		return args, done
	}

	// add the function that will match paths to the root handler
	r.tree.arg[ah.Root()] = append(r.tree.arg[ah.Root()], treeFn)
}

// Handler returns the handler to use for the given request,
// consulting r.Method, r.Host, and r.URL.Path. It always returns
// a non-nil handler. If the path is not in its canonical form, the
// handler will be an internally-generated handler that redirects
// to the canonical path.
//
// Handler also returns the registered pattern that matches the
// request or, in the case of internally-generated redirects,
// the pattern that will match after following the redirect.
//
// If there is no registered handler that applies to the request,
// Handler returns a ``page not found'' handler and an empty pattern.
func (r *Router) Handler(cmdln string, handler Handler) {
	r.Handle(cmdln,
		func(c *Context) {
			handler.ServeCmdln(c)
		},
	)
}

// HandleFunc registers the handler function for the given pattern.
func (r *Router) HandleFunc(cmdln string, handler HandlerFunc) {
	r.Handler(cmdln, handler)
}

// ServeCmdln dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (r *Router) ServeCmdln(c *Context) {
	var err error

	c.opt = r.opt.irf
	c.arg = r.arg.irf

	args := c.raw
	argsCh := make(chan string, len(args))

	// formualate the arguements as a channel...
	for _, v := range args {
		argsCh <- v
	}
	close(argsCh)

	// Set the default functionaly if needed.
	if len(r.opt.pfn) == 0 {
		r.opt.pfn = append(r.opt.pfn, defaultParseOptArgFunc)
	}

	// run through all of the functions serially to parse the commandline arguments to a struct or map
	for _, fn := range r.opt.pfn {
		// note that argsCh get's overwritten...
		if err, argsCh = fn(argsCh, r.opt.val, c); err != nil {
			c.err = errors.Wrap(err, "during function call")
		}
	}

	r.arg.pfn(argsCh, r.tree.arg, r.arg.val, c)
}

// Parse is what will kick off the cmdln router process
func Parse(args []string, handler Handler) {
	handler.ServeCmdln(NewContext(args))
}
