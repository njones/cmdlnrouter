package cmdlnrouter

import (
	"reflect"
	"testing"
)

func TestPassingOptionStruct(t *testing.T) {

	type opt struct {
		Short *bool `cmdln:"-s,--short,a short flag test"`
		Long  *bool `cmdln:"-l,--long,a long flag test"`
	}

	o := new(opt)
	r := New()
	r.Optional(o)

	var want0 *bool
	have0 := o.Short
	if want0 != have0 {
		t.Errorf("[0] Want: %v Have: %v", want0, have0)
	}

	want1 := o
	have1 := r.opt.irf

	if !reflect.DeepEqual(want1, have1) {
		t.Errorf("[1] Want: %v Have: %v", want1, have1)
	}

	want1a := map[string]reflect.Value{
		"-s":      reflect.ValueOf(o.Short),
		"--short": reflect.ValueOf(o.Short),
		"-l":      reflect.ValueOf(o.Long),
		"--long":  reflect.ValueOf(o.Long),
	}

	have1a := r.opt.val

	if !reflect.DeepEqual(want1a, have1a) {
		//t.Errorf("[1a] Want: %#v Have: %#v", want1a, have1a)
	}

	Parse([]string{"command", "-s", "--long"}, r)

	want1b := true
	have1b := o.Short

	if want0 == have1b {
		t.Errorf("[1b+] Want: %v Have: %v", want1b, have1b)
	}
	if want0 != have1b && want1b != *have1b {
		t.Errorf("[1b-] Want: %v Have: %v", want1b, have1b)
	}

	want2b := true
	have2b := want1.Long

	if want0 == have2b {
		t.Errorf("[1b+] Want: %v Have: %v", want2b, have2b)
	}

	if want0 != have2b && want2b != *have2b {
		t.Errorf("[1b-] Want: %v Have: %v", want2b, have2b)
	}

	var m int32
	want2 := ErrInvalidInterface

	r2 := New().Optional(m)
	have2 := r2.Err

	if want2 != have2 {
		t.Errorf("[2] Want: %v Have: %v", want2, have2)
	}
}

func TestMultipleOptions(t *testing.T) {

	type opt struct {
		Short []bool `cmdln:"-s,--short,a short flag test"`
	}

	var want0 []bool

	o := new(opt)
	r := New()
	r.Optional(o)

	Parse([]string{"command", "-s", "--short"}, r)

	want1b := []bool{true, true}
	have1b := o.Short

	if reflect.DeepEqual(want0, have1b) {
		t.Errorf("[1b+] Want: %v Have: %v", want1b, have1b)
	}

	if !reflect.DeepEqual(want0, have1b) && !reflect.DeepEqual(want1b, have1b) {
		t.Errorf("[1b-] Want: %v Have: %v", want1b, have1b)
	}
}

func TestBoolFalseOption(t *testing.T) {

	type opt struct {
		Short *bool `cmdln:"-s,--short,a short flag test"`
	}

	var want0 *bool

	o := new(opt)
	r := New()
	r.Optional(o)

	Parse([]string{"command", "-s=false"}, r)

	b := false
	want1b := &b
	have1b := o.Short

	if want0 == have1b {
		t.Errorf("[1b+] Want: %v Have: %v", want1b, have1b)
	}

	if want0 != have1b && *want1b != *have1b {
		t.Errorf("[1b-] Want: %v Have: %v", *want1b, *have1b)
	}
}

func TestStringOption(t *testing.T) {

	type opt struct {
		Short  *string  `cmdln:"-s,--short,a short flag test"`
		Medium *string  `cmdln:"-m,--med,a medium? flag test"`
		Long   []string `cmdln:"-l,--long,a long flag test"`
	}

	var want0 *string

	o := new(opt)
	r := New()
	r.Optional(o)

	Parse([]string{"command", "-s", "less", "--med=is", "-l", "more", `-l="or"`, "--long=moreless"}, r)

	s1 := "less"
	want1 := &s1
	have1 := o.Short

	if want0 == have1 {
		t.Errorf("[1+] Want: %v Have: %v", want1, have1)
	}

	if want0 != have1 && *want1 != *have1 {
		t.Errorf("[1-] Want: %v Have: %v", *want1, *have1)
	}

	s2 := "is"
	want2 := &s2
	have2 := o.Medium

	if want0 == have2 {
		t.Errorf("[2+] Want: %v Have: %v", want2, have2)
	}

	if want0 != have2 && *want2 != *have2 {
		t.Errorf("[2-] Want: %v Have: %v", *want2, *have2)
	}

	want3 := []string{"more", "or", "moreless"}
	have3 := o.Long

	if reflect.DeepEqual(want0, have3) {
		t.Errorf("[3+] Want: %v Have: %v", want3, have3)
	}

	if !reflect.DeepEqual(want0, have3) && !reflect.DeepEqual(want3, have3) {
		t.Errorf("[3-] Want: %v Have: %v", want3, have3)
	}
}

func TestMap1(t *testing.T) {

	want := map[string]string{"--short": "true", "-s": "money"}
	have := make(map[string]string)
	r := New()
	r.Optional(have)

	Parse([]string{"command", "--short", "-s", "money", "after", "pit"}, r)

	if !reflect.DeepEqual(want, have) {
		t.Errorf("[3+] Want: %v Have: %v", want, have)
	}
}

func TestHandledOption(t *testing.T) {

	type opt struct {
		Short *bool `cmdln:"-s,--short,a short flag test"`
	}

	var want0 *bool

	o := new(opt)
	r := New()
	r.Optional(o)

	handle := make(chan string, 2)
	r.Handle("command fixed", func(c *Context) { handle <- "good" })
	r.Handle("command unhandled", func(c *Context) { handle <- "bad" })
	defer func() {
		close(handle)
	}()
	Parse([]string{"command", "-s=false", "fixed"}, r)

	b := false
	want1b := &b
	have1b := o.Short

	if want0 == have1b {
		t.Errorf("[1b+] Want: %v Have: %v", want1b, have1b)
	}

	if want0 != have1b && *want1b != *have1b {
		t.Errorf("[1b-] Want: %v Have: %v", *want1b, *have1b)
	}

	close(handle)
	want2 := "good"
	have2 := <-handle
	if want2 != have2 {
		t.Errorf("[2] Want: %v Have: %v", want2, have2)
	}
}

func TestHandledArgument(t *testing.T) {

	type opt struct {
		Short *bool `cmdln:"-s,--short,a short flag test"`
	}

	type arg struct {
		Filename *string
	}

	var want0 *bool

	o := new(opt)
	a := new(arg)
	r := New()
	r.Optional(o)
	r.Arguments(a)

	handle := make(chan string, 2)
	r.Handle("command fixed :filename", func(c *Context) {
		ar := c.Arguments().(*arg)
		handle <- *ar.Filename
	})
	r.Handle("command unhandled", func(c *Context) { handle <- "bad" })
	defer func() {
		close(handle)
	}()
	Parse([]string{"command", "-s=false", "fixed", "/path/to/filename.md"}, r)

	b := false
	want1b := &b
	have1b := o.Short

	if want0 == have1b {
		t.Errorf("[1b+] Want: %v Have: %v", want1b, have1b)
	}

	if want0 != have1b && *want1b != *have1b {
		t.Errorf("[1b-] Want: %v Have: %v", *want1b, *have1b)
	}

	want2 := "/path/to/filename.md"
	have2 := <-handle
	if want2 != have2 {
		t.Errorf("[2] Want: %v Have: %v", want2, have2)
	}
}

func TestHandledDoubleArgument(t *testing.T) {

	type opt struct {
		Short *bool `cmdln:"-s,--short,a short flag test"`
	}

	type arg struct {
		Filename *string
	}

	var want0 *bool

	o := new(opt)
	a := new(arg)
	r := New()
	r.Optional(o)
	r.Arguments(a)

	handle := make(chan string, 2)
	r.Handle("command fixed :filename", func(c *Context) {
		ar := c.Arguments().(*arg)
		handle <- *ar.Filename
	})
	r.Handle("command unhandled :porter", func(c *Context) { handle <- "bad" })
	defer func() {
		close(handle)
	}()
	Parse([]string{"command", "-s=false", "fixed", "/path/to/filename.md"}, r)

	b := false
	want1b := &b
	have1b := o.Short

	if want0 == have1b {
		t.Errorf("[1b+] Want: %v Have: %v", want1b, have1b)
	}

	if want0 != have1b && *want1b != *have1b {
		t.Errorf("[1b-] Want: %v Have: %v", *want1b, *have1b)
	}

	want2 := "/path/to/filename.md"
	have2 := <-handle
	if want2 != have2 {
		t.Errorf("[2] Want: %v Have: %v", want2, have2)
	}
}
