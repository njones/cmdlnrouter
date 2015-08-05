package cmdlnrouter

import (
	"bufio"
	"fmt"
	"io"
	"os"
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

	bag        map[string]interface{} // holds items to pass along with the context
	cmdlnAsRaw []byte                 // The full raw commandline as bytes.
	cmdlnParse []byte                 // The full parsed commandline as bytes.
}

func (c *Context) Set(key string, value interface{}) {
	if c.bag == nil {
		c.bag = make(map[string]interface{})
	}
	c.bag[key] = value
}

func (c *Context) Get(key string) interface{} {
	return c.bag[key]
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
