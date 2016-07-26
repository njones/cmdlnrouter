package cmdlnrouter

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
