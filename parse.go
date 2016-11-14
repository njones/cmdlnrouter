package cmdlnrouter

import (
	"reflect"
	"strconv"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

func parseArgsHandler(argsHand argsHandled, argsChan chan string) (argList argsInputed, err error) {
	cmp := argsHand.Slice()
	cnt := 0
	for arg := range argsChan {
		switch {
		case len(cmp) < cnt+1:
			return nil, errors.New("Some Unhandled")
		case cmp[cnt] != arg && cmp[cnt][0] != ':':
			return nil, errors.New("Not a valid var")
		case cmp[cnt][0] != ':':
			argList = append(argList, argCmp{argVal: arg})
		case cmp[cnt][0] == ':':
			argList = append(argList, argCmp{argVar: cmp[cnt], argVal: arg})
		}
		cnt++
	}

	if cnt != len(argList) || argsHand.String() != argList.String() {
		err = errors.New("There is a problem with the arglist.")
	}
	return argList, err
}

func parseArgs(args <-chan string, argm argmap, valm valmap, c *Context) (err error) {

	var (
		ok        bool
		ssvErr    error
		handChs   []chan string
		doneChs   []chan struct{}
		getHandle []handleArgFunc
	)

	cmd := <-args
	if getHandle, ok = argm[cmd]; !ok {
		return
	}

	rtnCh := make(chan inputHandle, len(getHandle))
	for _, fn := range getHandle {
		handle, done := fn(rtnCh)
		handChs, doneChs = append(handChs, handle), append(doneChs, done)
	}

	// pass each arg along to each handler for matching
	for arg := range args {
		for _, ch := range handChs {
			ch <- arg
		}
	}

	// close all of the handles that are checking arguments
	for _, ch := range handChs {
		close(ch)
	}

	// drain the done channels
	for _, d := range doneChs {
		// simply block until this handler is done processing, or timed out
		<-d
	}

	// now that all of the handler have been processed, we can close the return channel
	close(rtnCh)

	if len(rtnCh) > 1 {
		ssvErr = errors.Wrap(ssvErr, "There is a conflict for the handlers")
	}

	if len(rtnCh) == 1 {
		s := <-rtnCh
		if s.er != nil {
			ssvErr = errors.Wrap(multierror.Append(ssvErr, s.er), "from handler")
		}
		for _, a := range s.in {
			if v, ok := valm[strings.ToLower(a.argVar)]; ok {
				if val, ok := v.(reflect.Value); ok {
					if ssvErr = setStructValue(val, a.argVal, nil); ssvErr != nil {
						ssvErr = errors.Wrap(ssvErr, "from setting a value")
					}
				}
			}
		}

		// error handling
		if ssvErr != nil {
			c.err = errors.Wrap(multierror.Append(c.err, ssvErr), "from setting values")
		}

		close(c.done)
		s.hn(c)
	}

	return
}

func setStructValue(rVal reflect.Value, sVal string, args <-chan string) (err error) {

	var argCheckBool = func(s string, a <-chan string) (rtn *bool) {
		var ptr bool
		switch s {
		case "1", "t", "T", "true", "TRUE", "True":
			ptr = true
		case "0", "f", "F", "false", "FALSE", "False":
			ptr = false
		default:
			ptr = true
		}
		return &ptr
	}

	var argCheckInt64 = func(s string, a <-chan string) (rtn *int64) {
		var i int64
		if i, err = strconv.ParseInt(s, 10, 64); err != nil {
			err = errors.Wrap(err, "during argCheckInt64")
			return nil
		}
		return &i
	}

	var argCheckInt32 = func(s string, a <-chan string) (rtn *int32) {
		i := int32(*argCheckInt64(s, a))
		return &i
	}

	var argCheckInt16 = func(s string, a <-chan string) (rtn *int16) {
		i := int16(*argCheckInt64(s, a))
		return &i
	}

	var argCheckInt = func(s string, a <-chan string) (rtn *int) {
		i := int(*argCheckInt64(s, a))
		return &i
	}

	var argCheckFloat = func(s string, a <-chan string) (rtn *float64) {
		var f float64
		if f, err = strconv.ParseFloat(s, 64); err != nil {
			err = errors.Wrap(err, "during argCheckFloat")
			return nil
		}
		return &f
	}

	var argCheckString = func(s string, a <-chan string) (rtn *string) {
		var ptr string
		switch s {
		case "":
			if a != nil && len(a) > 0 {
				ptr = <-a
			} else {
				err = errors.Wrap(err, "there should be a rValue, using empty")
				ptr = ""
			}
		default:
			if (s[0] == '"' && s[len(s)-1] == '"') ||
				(s[0] == '\'' && s[len(s)-1] == '\'') {
				if us, sErr := strconv.Unquote(s); err == nil {
					s = us
				} else {
					err = errors.Wrap(sErr, "during default unquote")
				}
			}
			ptr = s
		}
		return &ptr
	}

	var argCheckBytes = func(s string, a <-chan string) (rtn []byte) {
		return []byte(*argCheckString(s, a))
	}

	var argCheckByte = func(s string, a <-chan string) (rtn byte) {
		if len(s) == 0 {
			err = errors.Wrap(err, "The incoming string needs 1 byte")
		}
		if len(s) > 1 {
			err = errors.Wrap(err, "The incoming string is too long, more than 1 byte")
		}
		return s[0]
	}

	var valSet = func(r reflect.Value, v interface{}) {
		r.Set(reflect.ValueOf(v))
	}

	var valAppend = func(r reflect.Value, v interface{}) {
		switch r.Interface().(type) {
		case []byte:
			r.Set(reflect.AppendSlice(r, reflect.Indirect(reflect.ValueOf(v))))
		default:
			r.Set(reflect.Append(r, reflect.Indirect(reflect.ValueOf(v))))
		}
	}

	switch rVal.Interface().(type) {
	case *bool:
		valSet(rVal, argCheckBool(sVal, args))
	case []bool:
		valAppend(rVal, argCheckBool(sVal, args))
	case *byte:
		valSet(rVal, argCheckByte(sVal, args))
	case []byte:
		valAppend(rVal, argCheckBytes(sVal, args))
	case *float64:
		valSet(rVal, argCheckFloat(sVal, args))
	case []float64:
		valAppend(rVal, argCheckFloat(sVal, args))
	case *int:
		valSet(rVal, argCheckInt(sVal, args))
	case []int:
		valAppend(rVal, argCheckInt(sVal, args))
	case *int16:
		valSet(rVal, argCheckInt16(sVal, args))
	case []int16:
		valAppend(rVal, argCheckInt16(sVal, args))
	case *int32:
		valSet(rVal, argCheckInt32(sVal, args))
	case []int32:
		valAppend(rVal, argCheckInt32(sVal, args))
	case *int64:
		valSet(rVal, argCheckInt64(sVal, args))
	case []int64:
		valAppend(rVal, argCheckInt64(sVal, args))
	case *string:
		valSet(rVal, argCheckString(sVal, args))
	case []string:
		valAppend(rVal, argCheckString(sVal, args))
	default:
		err = errors.Wrap(err, "The value could not be converted.")
	}

	return err
}
