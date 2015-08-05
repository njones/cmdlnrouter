package cmdlnrouter

import (
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Bitwise parsing mode idenifiers.
const (
	ParseOptSingleDashAsOpt = iota << 1
	ParseOptOnlyBeforeFirstCmd

	ParseOptGoFlagStyle = ParseOptSingleDashAsOpt | ParseOptOnlyBeforeFirstCmd
)

type Handle func(*Context)

type Handler interface {
	ServeCmdln(*Context)
}

type HandlerFunc func(*Context)

func (f HandlerFunc) ServeCmdln(c *Context) { f(c) }

type SubRouter struct {
	subcmd string
	*Router
}

func (sr *SubRouter) SubCmd(s string) *SubRouter {
	return sr.Router.SubCmd(sr.subcmd + " " + s)
}

func (sr *SubRouter) Handle(cmdln string, handle Handle) {
	sr.Router.Handle(sr.subcmd+" "+cmdln, handle)
}

func (sr *SubRouter) Handler(cmdln string, handler Handler) {
	sr.Router.Handler(sr.subcmd+" "+cmdln, handler)
}

func (sr *SubRouter) HandlerFunc(cmdln string, handler HandlerFunc) {
	sr.Router.Handler(sr.subcmd+" "+cmdln, handler)
}

type Router struct {
	subs map[string]*SubRouter

	opts  interface{}
	cmds  interface{}
	trees map[*regexp.Regexp]Handle

	mode         int
	NotFound     Handler
	PanicHandler func(*Context, interface{})
}

func (r *Router) recovery(c *Context) {
	if rcvr := recover(); rcvr != nil {
		r.PanicHandler(c, rcvr)
	}
}

func (r *Router) Handle(cmdln string, handle Handle) {

	if r.trees == nil {
		r.trees = make(map[*regexp.Regexp]Handle)
	}

	cmdSpace := regexp.QuoteMeta(strings.Join(strings.Fields(cmdln), `\s+`))

	reCmd := regexp.MustCompile(`:(\w+)`)
	cmdRe := `^` + string(reCmd.ReplaceAll([]byte(cmdSpace), []byte(`(?P<$1>\w+)`))) + `$`

	// Loop through all of the subcommands and add those handlers here
	// log.Println("registering: ", cmdRe)

	r.trees[regexp.MustCompile(cmdRe)] = handle
}

func (r *Router) Handler(cmdln string, handler Handler) {
	r.Handle(cmdln,
		func(c *Context) {
			handler.ServeCmdln(c)
		},
	)
}

func (r *Router) HandlerFunc(cmdln string, handler HandlerFunc) {
	r.Handler(cmdln, handler)
}

func (r *Router) ServeCmdln(c *Context) {
	if r.PanicHandler != nil {
		defer r.recovery(c)
	}

	for rx, handle := range r.trees {
		if rx.Match(c.cmdlnParse) {

			parseCmds(rx, string(c.cmdlnParse), &r.cmds)
			c.Command = r.cmds
			handle(c)
			return
		}
	}

	if r.NotFound != nil {
		r.NotFound.ServeCmdln(c)
		return
	}
}

func (r *Router) Mode(i int) {
	r.mode = i
}

func (r *Router) SubCmd(s string) *SubRouter {
	if r.subs == nil {
		r.subs = make(map[string]*SubRouter)
	}

	r.subs[s] = &SubRouter{subcmd: s, Router: new(Router)}
	return r.subs[s]
}

func (r *Router) Options(opts interface{}) {
	r.opts = opts
}

func (r *Router) Commands(cmds interface{}) {
	r.cmds = cmds
}

func Parse(args []string, handler Handler) {
	parse, opts, extra := parseArgs(args, handler)

	c := NewContext()
	c.Options = opts
	c.Unhandled = extra
	c.cmdlnAsRaw = []byte(strings.Join(args, " "))
	c.cmdlnParse = []byte(Join(parse, " "))
	handler.ServeCmdln(c)
	switch handler.(type) {
	case *Router:
		for _, v := range handler.(*Router).subs {
			Parse(args, v)
		}
	case *SubRouter:
		for _, v := range handler.(*SubRouter).subs {
			Parse(args, v)
		}
	}
}

func parseCmds(rx *regexp.Regexp, cmdtxt string, cmd interface{}) {
	if cmd == nil {
		return
	}

	names := rx.SubexpNames()
	values := rx.FindStringSubmatch(cmdtxt)

	if r, ok := cmd.(map[string]interface{}); ok {
		for i, v := range names {
			if len(v) > 0 {
				r[v] = values[i]
			}
		}
		return
	}

	val := reflect.ValueOf(reflect.ValueOf(cmd).Interface())
	if val.Elem().Type().Kind() != reflect.Struct {
		log.Fatal("Only stucts can be passed in. Please check the type of the interface{}. Found: ", val.Elem().Type().Kind())
	}

	// For now we can only accept flat command. No structs or slices.
	// Just the following type:
	//    *string

	elm := val.Elem()
	for i := 0; i < elm.NumField(); i++ {
		vField := elm.Field(i)
		tField := elm.Type().Field(i)

		for n, v := range names {
			if strings.ToLower(tField.Name) == strings.ToLower(v) {
				if vField.CanSet() {
					vField.Set(reflect.ValueOf(&values[n]))
				}
				break
			}
		}
	}

	return
}

func parseArgs(args []string, handler Handler) ([]Argument, interface{}, map[string]string) {
	// create a map of the options we will be looking for
	var pags []Argument
	var opts interface{}
	var xtra map[string]string

	switch handler.(type) {
	case *Router:
		r := handler.(*Router)
		if r.opts != nil {
			pags, opts, xtra = parseArgsToStruct(r.mode, args, r.opts)
		} else {
			pags, opts = parseArgsToMap(r.mode, args)
		}
	case *SubRouter:
		r := handler.(*SubRouter)
		if r.opts != nil {
			pags, opts, xtra = parseArgsToStruct(r.mode, args, r.opts)
		} else {
			pags, opts = parseArgsToMap(r.mode, args)
		}
	}
	return pags, opts, xtra
}

func parseArgsToMap(mode int, args []string) (pags []Argument, opts map[string]*string) {

	opts = make(map[string]*string)
	for i, field := range args {
		if field[0] == '-' {
			opts[field] = func(a []string, i int) (r *string) {
				if len(args) > i {
					r = &a[i]
				}
				return
			}(args, i+1)
			continue
		}
		if i-1 >= 0 && args[i-1][0] == '-' {
			continue // skip becuase it should have already be processed
		}
		pags = append(pags, Argument{arg: field})
	}
	return
}

func parseArgsToStruct(mode int, args []string, optsIn interface{}) (pags []Argument, opts interface{}, unhandled map[string]string) {
	val := reflect.ValueOf(reflect.ValueOf(optsIn).Interface())
	if val.Elem().Type().Kind() != reflect.Struct {
		log.Fatal("Only stucts can be passed in. Please check the type of the interface{}. Found: ", val.Elem().Type().Kind())
	}

	// For now we can only accept flat options. No structs or slices.
	// Just the following types:
	//    *int
	//    *float
	//    *string
	//    *bool

	// This will hold the field names of the struct to compare the comandline
	// against.
	flatMap := make(map[string]struct {
		k int  // key index
		v int  // value index
		f bool // found in struct
	})

	// This is a 3 step process
	// STEP 1
	// 1. Create a copy of the Arguments list
	// 2. Create an options map with the items from the Args list that
	//    are flags

	// STEP 2
	// 3. Map the items from the options map to the options object
	// 4. Replace the items with empty arguements in the copy list of the
	//    options that were used to map

	// STEP 3
	// 4. Update the original arguments list, removing the empty items.

	var tmpPags []Argument
	for i, arg := range args {
		tmpPags = append(tmpPags, Argument{arg: arg})

		if arg[0] == '-' {
			v := i + 1
			if v > len(args) {
				v = -1
			}
			flatMap[arg] = struct {
				k int
				v int
				f bool
			}{i, i + 1, false} // always return index of the value to take.
			continue
		}
		if i-1 >= 0 && args[i-1][0] == '-' {
			continue // skip becuase it should have already be processed
		}
	}

	// Add the data to the struct, using type hints to apply the data
	var removeArg Argument
	elm := val.Elem()
	for i := 0; i < elm.NumField(); i++ {
		vField := elm.Field(i)
		tField := elm.Type().Field(i)

		tags := strings.Split(tField.Tag.Get("cmdln"), ",")
		for i, v := range tags {
			switch i {
			case 0, 1:
				if v != "" && v != "-" && v != "--" {

					if argv, ok := flatMap[v]; ok {
						argv.f = true
						if vField.CanSet() == false {
							log.Println("Please make sure the ", tField.Name, " field is exportable.")
							continue
						}
						switch vField.Interface().(type) {
						case *int:
							if argv.v == -1 {
								log.Println("No value for the int found.")
								tmpPags[argv.k] = removeArg
								continue
							}
							vPtr, err := strconv.Atoi(args[argv.v])
							if err != nil {
								log.Println("Error converting to int. Err: ", err)
							}
							tmpPags[argv.k] = removeArg
							tmpPags[argv.v] = removeArg
							vField.Set(reflect.ValueOf(&vPtr))
						case *float64:
							if argv.v == -1 {
								log.Println("No value for the float flag found.")
								tmpPags[argv.k] = removeArg
								continue
							}
							vPtr, err := strconv.ParseFloat(args[argv.v], 10)
							if err != nil {
								log.Println("Error converting to float. Err: ", err)
							}
							vField.Set(reflect.ValueOf(&vPtr))
							tmpPags[argv.k] = removeArg
							tmpPags[argv.v] = removeArg
						case *string:
							if argv.v == -1 {
								log.Println("No value for the string found.")
								pags[argv.k] = removeArg
								continue
							}
							vPtr := args[argv.v]
							vField.Set(reflect.ValueOf(&vPtr))
							tmpPags[argv.k] = removeArg
							tmpPags[argv.v] = removeArg
						case *bool:
							vPtr := true
							vField.Set(reflect.ValueOf(&vPtr))
							tmpPags[argv.k] = removeArg
						default:
							log.Println("Not parseable. Found Kind: ", vField.Type())
						}
					}
				}
			}
		}
	}

	unhandled = make(map[string]string)

	for _, v := range tmpPags {
		if val, ok := flatMap[v.arg]; ok {
			unhandled[v.arg] = args[val.v]
			tmpPags[val.k] = removeArg
			if val.v != -1 {
				tmpPags[val.v] = removeArg
			}
			continue
		}
		if v != removeArg {
			pags = append(pags, v)
		}
	}

	return pags, optsIn, unhandled
}

func Join(args []Argument, s string) string {
	var str []string
	for _, a := range args {
		str = append(str, a.String())
	}
	return strings.Join(str, s)
}
