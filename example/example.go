package main

import (
	"fmt"
	"os"

	cmdln "./cmdlnrouter"
)

type Options struct {
	Config  *string `cmdln:"-c,--config"`
	NoShort *int    `cmdln:"-,--noshort"`
}

type SubOptions struct {
	NoLong *string `cmdln:"-n,--,This is a simple description of no long"`
}

func Version() string {
	return "version 1.0"
}

func doExample(c *cmdln.Context) {
	fmt.Fprint(c.Stdout, "Yes this is good")
	fmt.Println()
	fmt.Println(c.Options)
	fmt.Println(c.Unhandled)
}

func main() {
	var opt Options
	r := new(cmdln.Router)
	r.Options(&opt)

	sub := r.SubCmd("example sub")
	subSub := sub.SubCmd("more")

	subopt := SubOptions{}
	sub.Options(&subopt)
	subSub.Handle("", doExample)
	subSub.Handle("place", doExample)
	sub.Handle("kicks", doExample)

	r.Handle("example", doExample)
	r.HandlerFunc("example :func", cmdln.CmdlnHandlerFunc(func(c *cmdln.Context) {
		fmt.Println(Version())
		os.Exit(0)
	}))
	r.HandlerFunc("example ask something", cmdln.CmdlnHandlerFunc(func(c *cmdln.Context) {
		resp := c.Ask("What is your name?")
		fmt.Println("<<<", resp)
	}))

	cmdln.Parse(os.Args[1:], r)
}