package cmdlnrouter

import "testing"
import "strings"
import "reflect"
import "encoding/json"
import "regexp"

func TestParseArgsToMap(t *testing.T) {

	ptrs := func(x string) *string { return &x }
	ptrn := func() *string { return nil }

	tests := []struct {
		test []string
		pags []string
		opts map[string]*string
	}{
		{
			[]string{"example"},
			[]string{"example"},
			map[string]*string{},
		},
		{
			strings.Split("example with more than 1", " "),
			[]string{"example", "with", "more", "than", "1"},
			map[string]*string{},
		},
		{
			strings.Split("--example with --some flags", " "),
			[]string{},
			map[string]*string{"--example": ptrs("with"), "--some": ptrs("flags")},
		},
		{
			strings.Split("--example --with --extra-space-and --only --flags", " "),
			[]string{},
			map[string]*string{
				"--example":         ptrs("--with"),
				"--with":            ptrs("--extra-space-and"),
				"--extra-space-and": ptrs("--only"),
				"--only":            ptrs("--flags"),
				"--flags":           ptrn(),
			},
		},
		{
			[]string{"example", "--simple", "flag"},
			[]string{"example"},
			map[string]*string{"--simple": ptrs("flag")},
		},
		{
			strings.Split("example -s flag", " "),
			[]string{"example"},
			map[string]*string{"-s": ptrs("flag")},
		},
		{
			strings.Split("example -s flag example", " "),
			[]string{"example", "example"},
			map[string]*string{"-s": ptrs("flag")},
		},
		{
			strings.Split("example -swma flag example", " "),
			[]string{"example", "example"},
			map[string]*string{"-swma": ptrs("flag")},
		},
		{
			strings.Split("example -swma", " "),
			[]string{"example"},
			map[string]*string{"-swma": ptrn()},
		},
		{
			strings.Split("--simple flag example", " "),
			[]string{"example"},
			map[string]*string{"--simple": ptrs("flag")},
		},
		{
			strings.Split("ex•mple -a 1 -b 2 --cdef 3 four", " "),
			[]string{"ex•mple", "four"},
			map[string]*string{
				"-a":     ptrs("1"),
				"-b":     ptrs("2"),
				"--cdef": ptrs("3"),
			},
		},
		{
			[]string{"bonjour", "⛳", "-a", "hello world"},
			[]string{"bonjour", "⛳"},
			map[string]*string{"-a": ptrs("hello world")},
		},
	}

	for _, tst := range tests {
		a, b := parseArgsToMap(0, tst.test)
		a1 := strings.Join(tst.pags, " ")
		a2 := Join(a, " ")
		if a1 != a2 {
			t.Error("Expected:", a1, "Found:", a2)
		}
		if !reflect.DeepEqual(tst.opts, b) {
			t.Error("Expected:", tst.opts, "Found:", b)
		}
	}
}

type TestContext struct {
	opts interface{}
}

type TestOptionsStruct1 struct {
	A *int     `cmdln:"--aye,-a,A short description"`
	B *float64 `cmdln:"--bee,-b,A short description"`
	C *string  `cmdln:"--sea,-c,A short description"`
	D *bool    `cmdln:"--dei,-d,A short description"`
}

type TestCommandStruct1 struct {
	Ample *string
}

func TestParseArgsToStruct(t *testing.T) {

	tests := []struct {
		test []string
		pags []string
		strt TestOptionsStruct1
		opts string
	}{
		{
			[]string{"example"},
			[]string{"example"},
			TestOptionsStruct1{},
			`{"A":null,"B":null,"C":null,"D":null}`,
		},
		{
			[]string{"example", "--aye", "1"},
			[]string{"example"},
			TestOptionsStruct1{},
			`{"A":1,"B":null,"C":null,"D":null}`,
		},
		{
			[]string{"example", "--aye", "1", "-b", "2.5", "--sea", "This is a test?"},
			[]string{"example"},
			TestOptionsStruct1{},
			`{"A":1,"B":2.5,"C":"This is a test?","D":null}`,
		},
	}

	for _, tst := range tests {

		c := new(TestContext)
		c.opts = &tst.strt

		a, b, _ := parseArgsToStruct(0, tst.test, c.opts)
		a1 := strings.Join(tst.pags, " ")
		a2 := Join(a, " ")

		b1, _ := json.Marshal(b)
		b2 := string(b1)

		if a1 != a2 {
			// t.Error("Expected:", a1, "Found:", a2)
		}
		if tst.opts != b2 {
			t.Error("Expected:", tst.opts, "Found:", b2)
		}
	}
}

func TestParseCommand(t *testing.T) {

	tests := []struct {
		rgex *regexp.Regexp
		cmln string
		cmds string
		strt TestCommandStruct1
	}{
		{
			regexp.MustCompile(`^example$`),
			"example",
			`{"Ample":null}`,
			TestCommandStruct1{},
		},
		{
			regexp.MustCompile(`^(?P<ample>\w+)$`),
			"example",
			`{"Ample":"example"}`,
			TestCommandStruct1{},
		},
	}

	for _, tst := range tests {

		parseCmds(tst.rgex, tst.cmln, &tst.strt)

		a1, _ := json.Marshal(tst.strt)
		a2 := string(a1)

		if tst.cmds != a2 {
			t.Error("Expected:", tst.cmds, "Found:", a2)
		}
	}

	b1 := map[string]interface{}{"ample": "example"}
	b2 := make(map[string]interface{})
	parseCmds(regexp.MustCompile(`^(?P<ample>\w+)$`), "example", b2)

	if !reflect.DeepEqual(b1, b2) {
		t.Error("Expected:", b1, "Found:", b2)
	}
}
