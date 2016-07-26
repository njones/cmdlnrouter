package cmdlnrouter

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"
)

var helpFieldSortOrder = func() map[byte]int {
	o := make(map[byte]int)
	for i, v := range []byte("AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz ") {
		o[v] = i
	}
	return o
}()

type helpFields struct {
	ShortFld string
	LongFld  string
	VarFld   string
	DescFld  string
	PadSpace string
}

type ByHelpFields []helpFields

func (s ByHelpFields) Len() int {
	return len(s)
}

func (s ByHelpFields) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ByHelpFields) Less(i, j int) bool {
	si := (strings.Trim(s[i].ShortFld, "-") + " ")[0]
	sj := (strings.Trim(s[j].ShortFld, "-") + " ")[0]
	if s[i].ShortFld == "" && s[j].ShortFld == "" {
		// Case doesn't matter for long strings
		si = (strings.Trim(strings.ToLower(s[i].LongFld), "-") + " ")[0]
		sj = (strings.Trim(strings.ToLower(s[j].LongFld), "-") + " ")[0]
	}
	return helpFieldSortOrder[si] < helpFieldSortOrder[sj]
}

func parseCmdlnTag(tag string) (optShort, optLong, optDesc string) {
	tags := strings.Split(tag, ",")
	// There is a short tag
	if len(tags) > 0 && tags[0] != "-" {
		optShort = tags[0]
	}
	// There is a long tag
	if len(tags) > 1 && tags[1] != "--" {
		optLong = tags[1]
	}
	if len(tags) > 2 {
		optDesc = tags[2]
	}

	return
}

func parseCmdlnVTag(tag, fieldName string) (optVar string) {
	if len(tag) == 0 {
		return fieldName
	}
	if len(tag) > 0 && tag != "-" {
		return strings.Split(tag, ",")[0]
	}

	return
}

func maxOptLen(oldOptLen, newOptLen int) (optLen int) {
	optLen = newOptLen
	if newOptLen < oldOptLen {
		return oldOptLen
	}
	return
}

func (r *Router) helpMap() (helpFlags, helpOptions []helpFields) {

	// Loop through all and convert to the first part of the line
	// Find the longsest first part then 3 extra spaces
	// then add all of the paddings (part2) for each line with the 3rd part.
	if r.opts != nil {
		var optLen int
		helpFlags, helpOptions = make([]helpFields, 0), make([]helpFields, 0)

		val := reflect.ValueOf(reflect.ValueOf(r.opts).Interface())
		elm := val.Elem()

		// Loop through the fields of the options struct
		// to get the short & long names, descriptions and
		// the variable field name if there.
		for i := 0; i < elm.NumField(); i++ {
			vField := elm.Field(i)
			tField := elm.Type().Field(i)

			var optVar string
			optShort, optLong, optDesc := parseCmdlnTag(tField.Tag.Get("cmdln"))

			if fmt.Sprintf("%s", vField.Type()) == "*bool" {
				helpFlags = append(helpFlags, helpFields{
					ShortFld: optShort,
					LongFld:  optLong,
					DescFld:  optDesc,
				})
			} else {
				optVar = parseCmdlnVTag(tField.Tag.Get("cmdvar"), tField.Name)
				helpOptions = append(helpOptions, helpFields{
					ShortFld: optShort,
					LongFld:  optLong,
					VarFld:   optVar,
					DescFld:  optDesc,
				})
			}

			optLen = maxOptLen(optLen, len(optShort)+len(optLong)+len(optVar))
		}

		for _, fields := range [][]helpFields{helpFlags, helpOptions} {
			for i, v := range fields {
				padLen := optLen - (len(v.ShortFld) + len(v.LongFld) + len(v.VarFld))
				v.PadSpace = strings.Repeat(" ", padLen)
				fields[i] = v
			}
		}
	}

	sort.Sort(ByHelpFields(helpFlags))
	sort.Sort(ByHelpFields(helpOptions))

	return helpFlags, helpOptions
}

func genFlgTxt(helpFlags []helpFields) string {
	flagTxt := "-"
	for _, v := range helpFlags {
		flagTxt += strings.Trim(v.ShortFld, "-")
	}
	return flagTxt
}

func genCmdTxt(helpTree []map[string][]int) (cmdLnTxt string) {
	for _, v := range helpTree {
		var cmdTxts []string
		for cmdTxt := range v {

			if strings.Contains(cmdTxt, ":") {
				cmdTxt = "[ " + strings.ToUpper(cmdTxt) + " ]"
			}

			if strings.Contains(cmdTxt, `\|`) {
				cmdTxt = "[ " + strings.Join(strings.Split(cmdTxt, `\|`), " | ") + " ]"
			}

			cmdTxts = append(cmdTxts, cmdTxt)
		}

		if len(cmdTxts) == 1 {
			cmdLnTxt += " " + cmdTxts[0]
		} else {
			sort.Strings(cmdTxts)
			cmdLnTxt += " [ " + strings.Join(cmdTxts, " ") + " ]"
		}
	}
	return strings.TrimSpace(cmdLnTxt)
}

func (r *Router) Help() string {
	out := new(bytes.Buffer)

	helpFlags, helpOptions := r.helpMap()

	appTxt := filepath.Base(os.Args[0])
	flgTxt := genFlgTxt(helpFlags)
	cmdTxt := genCmdTxt(r.helpTree)

	hlpTplData := struct {
		Application  string
		LongFlags    string
		ShortFlags   string
		Command      string
		FlagsRange   []helpFields
		OptionsRange []helpFields
	}{
		Application:  appTxt,
		ShortFlags:   flgTxt,
		Command:      cmdTxt,
		FlagsRange:   helpFlags,
		OptionsRange: helpOptions,
	}

	t := template.Must(template.New("help").Parse(helpBasic))
	err := t.Execute(out, hlpTplData)
	if err != nil {
		log.Fatalf("execution failed: %s", err)
	}

	return out.String()
}

func (r *Router) CmdList() []string {
	return r.cmdlst
}

var (
	helpShort = `{{.Application}}, version {{.Version}}

usage: {{.Application}} {{.ShortFlags}} {{.Command}}

{{range $k, $v := .OptionsRange}}
{{if $k.ShortWithVar == ""}}
	{{$k.Short}} : {{$v.ShortDesc}}
{{end}}
{{end}}

{{range $k, $v := .OptionsRange}}
{{if $k.ShortWithVar != ""}}
	{{$k.ShortWithVar}} : {{$v.ShortDesc}}
{{end}}
{{end}}
`

	helpMinimum = `usage: {{.Application}} {{.ShortFlags}} {{.LongFlags}} {{.Options}} {{.Command}}
`

	helpBasic = `Usage: {{.Application}} [options...] {{.Command}}

Options:{{range $i, $v := .OptionsRange}}
  {{with $v.ShortFld}}{{.}}{{end}}{{with $v.LongFld}}{{if $v.ShortFld}}, {{end}}{{.}}{{end}}{{with $v.VarFld}} {{.}}{{end}}{{$v.PadSpace}}{{if $v.ShortFld | not}}  {{end}}{{if $v.LongFld | not}}  {{end}}{{if $v.VarFld | not}} {{end}}   {{$v.DescFld}}{{end}}

Flags:{{range $k, $v := .FlagsRange}}
  {{with $v.ShortFld}}{{.}}{{end}}{{with $v.LongFld}}{{if $v.ShortFld}}, {{end}}{{.}}{{end}}{{with $v.VarFld}} {{.}}{{end}}{{$v.PadSpace}}{{if $v.ShortFld | not}}  {{end}}{{if $v.LongFld | not}}  {{end}}{{if $v.VarFld | not}} {{end}}   {{$v.DescFld}}{{end}}
`
)
