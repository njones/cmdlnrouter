package cmdlnrouter

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Bitwise parsing mode identifiers.
const (
	ParseOptSingleDashAsOpt = iota << 1
	ParseOptOnlyBeforeFirstCmd
	ParseFailOnUnhandled

	ParseOptGoFlagStyle = ParseOptSingleDashAsOpt | ParseOptOnlyBeforeFirstCmd | ParseFailOnUnhandled
)

type Handle func(*Context)

type Handler interface {
	ServeCmdln(*Context)
}

type HandlerFunc func(*Context)

func (f HandlerFunc) ServeCmdln(c *Context) { f(c) }

type Router struct {
	subs map[string]*SubRouter

	opts   interface{}
	cmds   interface{}
	trees  map[*regexp.Regexp]Handle
	cmdlst []string

	helpTree []map[string][]int

	mode int

	// All of the handlers for issues
	HandlerDone      Handle
	HelpHandler      Handle
	NotFoundHandler  Handle
	UnhandledHandler Handle
	PanicHandler     func(*Context, interface{})
}

// Runs the PanicHandler if there is a panic
func (r *Router) recovery(c *Context) {
	if rcvr := recover(); rcvr != nil {
		r.PanicHandler(c, rcvr)
	}
}

func JoinWithSpace(ss []string) (s string) {
	for i, x := range ss {
		if i == 0 {
			s = x
			continue
		}
		switch x[len(x)-1] {
		case '?':
			s += `\s*` + x
		default:
			s += `\s+` + x
		}

	}
	return
}

func (r *Router) Handle(cmdln string, handle Handle) {

	if r.trees == nil {
		r.trees = make(map[*regexp.Regexp]Handle)
	}

	if r.cmdlst == nil {
		r.cmdlst = make([]string, 0)
	}

	r.cmdlst = append(r.cmdlst, cmdln)

	cmdFlds := strings.Fields(regexp.QuoteMeta(cmdln))
	initHelpTreeMaps(r, cmdFlds)

	reCmd := regexp.MustCompile(`:(\w+)`)
	reOptCmd := regexp.MustCompile(`:(\w+):`)

	for i, v := range cmdFlds {
		r.helpTree[i][v] = append(r.helpTree[i][v], i)

		cmdOr := strings.Split(v, `\|`)
		if len(cmdOr) > 1 {
			cmdFlds[i] = fmt.Sprintf("(%s)", strings.Join(cmdOr, `|`))
		}

		cmdFlds[i] = string(reOptCmd.ReplaceAll([]byte(cmdFlds[i]), []byte(`(?P<$1>.[^\s]+)?`)))
		cmdFlds[i] = string(reCmd.ReplaceAll([]byte(cmdFlds[i]), []byte(`(?P<$1>.[^\s]+)`)))
	}

	cmdSpacePlus := "^" + JoinWithSpace(cmdFlds) + "$"

	// // Loop through all of the subcommands and add those handlers here
	// log.Println("registering: ", cmdSpacePlus)

	r.trees[regexp.MustCompile(cmdSpacePlus)] = handle
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

	if len(c.Unhandled) > 0 && r.UnhandledHandler != nil {
		r.UnhandledHandler(c)
		return
	}

	for rx, handle := range r.trees {
		if rx.Match(c.cmdlnParse) {

			parseCmds(rx, string(c.cmdlnParse), r.cmds)
			c.Command = r.cmds
			handle(c)
			if r.HandlerDone != nil {
				r.HandlerDone(c)
			}
			return
		}
	}

	if r.NotFoundHandler != nil {
		r.NotFoundHandler(c)
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

func (r *Router) Command(cmds interface{}) {
	r.cmds = cmds
}

func initHelpTreeMaps(r *Router, cmdFlds []string) {
	// initalize the maps that are needed to fill out the
	// helpTree when adding the fields
	extraFields := len(cmdFlds) - len(r.helpTree)
	if extraFields > 0 {
		fieldMap := make([]map[string][]int, extraFields)
		r.helpTree = append(r.helpTree, fieldMap...)
		for i := range r.helpTree {
			if r.helpTree[i] == nil {
				r.helpTree[i] = make(map[string][]int)
			}
		}
	}
}

// Parse will start the parsing process for the commandline
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
		log.Fatal("Only stucts can be passed in [1]. Please check the type of the interface{}. Found: ", val.Elem().Type().Kind())
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
			continue // skip because it should have already be processed
		}
		pags = append(pags, Argument{arg: field})
	}

	return
}

type M struct {
	k int // key index
	v int // value index
}

func genArgMaps(args []string) (flatMap map[string]M, tmpPags []Argument) {
	flatMap = make(map[string]M)

	for i, arg := range args {
		tmpPags = append(tmpPags, Argument{arg: arg})

		if arg[0] == '-' {
			v := i + 1
			if v > len(args) {
				v = -1
			}
			if strings.Contains(arg, "=") { // This doesn't check for inside of quotes
				arg = strings.Split(arg, "=")[0]
				v = -2
			}
			flatMap[arg] = M{i, v} // always return index of the value to take.
			continue
		}
		if i-1 >= 0 && args[i-1][0] == '-' && flatMap[args[i-1]].v != -2 {
			continue // skip because it should have already be processed
		}
	}

	return
}

func parseCmdlnVal(data M, args []string, defaultVal string) (val string, err error) {
	switch data.v {
	case -1:
		// This is a flag that is expecting a default value
		if len(defaultVal) > 0 {
			val = defaultVal
		} else {
			err = errors.New("No default value found.")
		}
		return
	case -2:
		// This is an arguement that looks like --key=<value>
		vals := strings.Split(args[data.k], "=")
		if len(vals) > 1 {
			val = vals[1]
		} else {
			err = errors.New("No value found.")
		}
		return
	}
	// This is for if a bool is at the end
	if data.v < len(args) {
		val = args[data.v]
	}
	return
}

func parseDefaultVal(arg string) (a, b string) {
	arg = a
	if strings.Contains(arg, "=") {
		kv := strings.Split(arg, "=")
		if len(kv) == 2 {
			a = kv[0]
			b = kv[1]
		}
	}
	return
}

func parseArgsToStruct(mode int, args []string, optsIn interface{}) (pags []Argument, opts interface{}, unhandled map[string]string) {
	//	var err error
	var removeArg Argument

	argMap, argTmpMap := genArgMaps(args)

	val := reflect.ValueOf(reflect.ValueOf(optsIn).Interface())
	if val.Elem().Type().Kind() != reflect.Struct {
		log.Fatal("Only stucts can be passed in [2]. Please check the type of the interface{}. Found: ", val.Elem().Type().Kind())
	}
	elm := val.Elem()

	// For now we can only accept flat options. No structs or slices.
	// Just the following types:
	//    *int
	//    *float
	//    *string
	//    *bool

	// Add the data to the struct, using type hints to apply the data
	for i := 0; i < elm.NumField(); i++ {
		vField := elm.Field(i)
		tField := elm.Type().Field(i)

		if vField.CanSet() == false {
			log.Println("Please make sure the ", tField.Name, " field is exportable.")
			continue
		}

		tags := tField.Tag.Get("cmdln")
		optShort, optLong, _ := parseCmdlnTag(tags)

		for _, v := range []string{optShort, optLong} {
			if argv, ok := argMap[v]; ok {

				_, defaultVal := parseDefaultVal(v)

				val, err := parseCmdlnVal(argv, args, defaultVal)
				if err != nil {
					log.Println("Error getting value. Err: ", err)
					continue
				}

				switch vField.Interface().(type) {
				case *int:
					vPtr, err := strconv.Atoi(val)
					if err != nil {
						log.Println("Error converting to int. Err: ", err)
						continue
					}
					vField.Set(reflect.ValueOf(&vPtr))
					argTmpMap[argv.k] = removeArg
				case *float64:
					vPtr, err := strconv.ParseFloat(val, 10)
					if err != nil {
						log.Println("Error converting to int. Err: ", err)
						continue
					}
					vField.Set(reflect.ValueOf(&vPtr))
					argTmpMap[argv.k] = removeArg
				case *string:
					vField.Set(reflect.ValueOf(&val))
					argTmpMap[argv.k] = removeArg
				case *bool:
					vPtr := true
					vField.Set(reflect.ValueOf(&vPtr))
					argTmpMap[argv.k] = removeArg
				default:
					log.Println("Not parse-able. Found Kind: ", vField.Type())
				}
			}
		}
	}

	unhandled = make(map[string]string)

	for _, v := range argTmpMap {
		if val, ok := argMap[v.arg]; ok {
			if val.v < len(args) { // because unhandled may assume an extra param...
				unhandled[v.arg] = args[val.v]
			} else {
				unhandled[v.arg] = ""
			}
			argTmpMap[val.k] = removeArg
			if val.v != -1 && val.v < len(args) {
				argTmpMap[val.v] = removeArg
			}
			continue
		}
		if v != removeArg {
			pags = append(pags, v)
		}
	}

	// log.Println(pags, optsIn, unhandled)
	return pags, optsIn, unhandled
}

func Join(args []Argument, s string) string {
	var str []string
	for _, a := range args {
		str = append(str, a.String())
	}
	return strings.Join(str, s)
}

func DefaultStr(x string) *string     { return &x }
func DefaultInt(x int) *int           { return &x }
func DefaultFloat(x float64) *float64 { return &x }
func DefaultBool(x bool) *bool        { return &x }
