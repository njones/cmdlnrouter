package cmdlnrouter

import "reflect"

func defaultParseOptArgFunc(args <-chan string, vals valmap, c *Context) (err error, out chan string) {
	out = make(chan string, len(args))

	if v, ok := vals["*"]; ok {
		if val, ok := v.(map[string]string); ok {
			setMapValues(val, args, out)
		}
		close(out)
		return
	}

	for s := range args {
		// check if there is a =<value>
		var e string
		s, e = optArgKeyVal(s)

		if v, ok := vals[s]; ok {
			if val, ok := v.(reflect.Value); ok {
				setStructValue(val, e, args)
			}
		} else {
			if isOptArg(s) {
				c.unhandled = append(c.unhandled, s)
			} else {
				out <- s
			}
		}
	}

	close(out)
	return
}

func isOptArg(s string) bool {
	if len(s) > 0 {
		return s[0] == '-'
	}
	return false
}

func optArgKeyVal(s string) (k, v string) {
	k = s
	for i, r := range s {
		if r == '=' {
			if len(s) >= i+1 {
				k, v = string(s[:i]), string(s[i+1:])
				break
			}
		}
	}
	return k, v
}

func setMapValues(val map[string]string, args <-chan string, out chan string) {

	var optArgToStr = func(s string) string {
		if !isOptArg(s) {
			return s
		}
		return "true"

	}

	for s := range args {
		// check if there is a =<value>
		var e string
		s, e = optArgKeyVal(s)

		// if we can pull more arguments, then start to lookahead
		if len(args) > 0 {

			if !isOptArg(s) {
				// since this is the first one, it's the command.
				out <- s
				s, e = optArgKeyVal(<-args)
			}

			// this is a "look-ahead" next, as we still use the s, and e variables filled in from before.
			for next := range args {
				if e == "" && isOptArg(s) {
					val[s] = optArgToStr(next)
					if !isOptArg(next) {
						next = <-args
					}
				} else if e != "" {
					val[s] = e
				} else {
					out <- s
				}
				s, e = optArgKeyVal(next)
			}
		}

		if isOptArg(s) {
			val[s] = optArgToStr(e)
		} else if s != "" {
			out <- s
		}
	}
}
