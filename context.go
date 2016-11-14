package cmdlnrouter

import (
	"bufio"
	"errors"
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

// NewContext returns a new context that can be used by the commandline
// options
func NewContext() *Context {
	c := new(Context)
	c.bag = make(map[string]interface{})
	
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.StdErr = os.Stderr

	return c
}

// Ask is a convenience method for getting commandline input.
func (c *Context) Ask(s string) (r string) {
	fmt.Fprint(c.Stdout, s, " ")
	scnln := bufio.NewScanner(c.Stdin)
	scnln.Scan()
	return scnln.Text()
}

// Ask is a convenience method for getting commandline input.
func (c *Context) Confirm(s string) (r bool, e error) {
	fmt.Fprint(c.Stdout, s, " [y/n] ")
	scnln := bufio.NewScanner(c.Stdin)
	scnln.Scan()
	switch scnln.Text() {
	case "y", "Y", "yes":
		return true, nil
	case "n", "N", "no":
		return false, nil
	default:
		return false, errors.New("Invalid Response: " + scnln.Text())
	}

	return false, errors.New("Should not get here.")
}
