package cmdlnrouter

import (
	"io"
	"os"
	"fmt"
	"bufio"
)

type Argument struct {
	arg string
}

func (a *Argument) String() string {
	return a.arg
}

type Context struct {
	Options   interface{}
	Command   interface{}
	Unhandled map[string]string

	// If you use these, you can swap them out for testing purposes.
	Stdin  io.Reader
	Stdout io.Writer
	StdErr io.Writer

	cmdlnAsRaw []byte // The full raw commandline as bytes.
	cmdlnParse []byte // The full parsed commandline as bytes.
}

// NewContext returns a new context that can be used by the cmdln
// options
func NewContext() *Context {
	c := new(Context)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.StdErr = os.Stderr

	return c
}

// Ask is a convenience metho for getting commandline input.
func (c *Context) Ask(s string) (r string) {
	fmt.Fprint(c.Stdout, s, " ")
	scnln := bufio.NewScanner(c.Stdin)
	scnln.Scan()
	return scnln.Text()
}