package cmdlnrouter

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// NewContext returns the new context for the router
func NewContext(args []string) *Context {
	c := new(Context)
	c.raw = args
	c.done = make(chan struct{}, 0)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c
}

var (
	// ErrCxtValueNotFound is the error
	ErrCxtValueNotFound = errors.New("The context value was not found")

	// ErrCxtValueNotString is an error
	ErrCxtValueNotString = errors.New("The interface could nto be represented as a string")
)

// CxtArg is the interface from context that is returned from a context
type CxtArg interface {
	String() string
	Interface() interface{}
}

// DefaultCxtArg is how context arguments are stored
type DefaultCxtArg struct{ Itf interface{} }

// String returns the value as a string
func (dcx *DefaultCxtArg) String() string {
	switch dcx.Itf.(type) {
	case string:
		return dcx.Itf.(string)
	case fmt.Stringer:
		return dcx.Itf.(fmt.Stringer).String()

	}
	return "" // Returns empty even if interface is filled but cannot be represented as a string
}

// Interface returns the value as a raw interface
func (dcx *DefaultCxtArg) Interface() interface{} { return dcx.Itf }

// Context holds the context for the Commandline Handler
type Context struct {
	opt       interface{}
	arg       interface{}
	unhandled []string

	// Directly overwrite these if needed
	Stdin          io.Reader
	Stderr, Stdout io.Writer

	bag map[string]CxtArg

	err  error
	done chan struct{}

	raw []string
}

// Err returns a non-nil error value after Done is closed. Err returns
// Canceled if the context was canceled or DeadlineExceeded if the
// context's deadline passed. No other values for Err are defined.
// After Done is closed, successive calls to Err return the same value.
func (cx *Context) Err() error {
	return cx.err
}

// Err returns a non-nil error value after Done is closed. Err returns
// Canceled if the context was canceled or DeadlineExceeded if the
// context's deadline passed. No other values for Err are defined.
// After Done is closed, successive calls to Err return the same value.
func (cx *Context) Done() <-chan struct{} {
	return cx.done
}

// Optional is like a getter that returns the optional agruments struct
func (cx *Context) Optional() interface{} {
	return cx.opt
}

// Arguments is like a getter that returns the arguments struct
func (cx *Context) Arguments() interface{} {
	return cx.arg
}

// Set saves the key as the interface value
func (cx *Context) Set(key string, value interface{}) {
	if cx.bag == nil {
		cx.bag = make(map[string]CxtArg)
	}
	switch value.(type) {
	case CxtArg:
		cx.bag[key] = value.(CxtArg)
	default:
		cx.bag[key] = &DefaultCxtArg{Itf: value}
	}
}

// Get returns the full ContextArg value
func (cx *Context) Get(s string) (CxtArg, error) {
	if val, ok := cx.bag[s]; ok {
		return val, nil
	}
	return nil, ErrCxtValueNotFound
}

// GetStr returns the ContextArg key as a string
func (cx *Context) GetStr(s string) (string, error) {
	if val, ok := cx.bag[s]; ok {
		switch val.Interface().(type) {
		case string, fmt.Stringer:
			return val.String(), nil
		}
		return "", ErrCxtValueNotString
	}
	return "", ErrCxtValueNotFound
}

// GetStr is a convienece function to return an string
func GetStr(ca CxtArg) string {
	return ca.Interface().(string)
}

// GetInt is a convienece function to return an int
func GetInt(ca CxtArg) int {
	return ca.Interface().(int)
}

// GetInt16 is a convienece function to return an int16
func GetInt16(ca CxtArg) int16 {
	return ca.Interface().(int16)
}

// GetInt32 is a convienece function to return an int32
func GetInt32(ca CxtArg) int32 {
	return ca.Interface().(int32)
}

// GetInt64 is a convienece function to return an int64
func GetInt64(ca CxtArg) int64 {
	return ca.Interface().(int64)
}

// GetFloat64 is a convienece function to return an float64
func GetFloat64(ca CxtArg) float64 {
	return ca.Interface().(float64)
}

// GetBool is a convienece function to return an bool
func GetBool(ca CxtArg) bool {
	return ca.Interface().(bool)
}
