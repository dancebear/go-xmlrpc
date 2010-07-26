package xmlrpc

import (
    "encoding/base64"
    "bufio"
    "fmt"
    "http"
    "io"
    "net"
    "os"
    "strconv"
    "strings"
    "xml"
)

func isSpace(c byte) bool {
        return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

type XMLRPCError struct {
    Msg string
}

func (err XMLRPCError) String() string { return err.Msg }

type ParseState int

var psStrings = []string { "Method", "MethodName", "InName", "Params",
    "Param", "Value", "EndValue", "EndParam", "EndParams", "EndMethod", "???", }

func (ps ParseState) String() string { return psStrings[ps] }

const (
    psMethod ParseState = iota
    psName
    psInName
    psParms
    psParam
    psValue
    psEndValue
    psEndParam
    psEndParms
    psEndMethod
)

func getStateVals(st ParseState, isResp bool) (string, bool) {
    isEnd := (st == psEndMethod || st == psEndParms || st == psEndParam ||
        st == psEndValue)

    var tag string
    switch st {
    case psMethod, psEndMethod:
        if isResp {
            tag = "methodResponse"
        } else {
            tag = "methodCall"
        }
    case psName, psInName:
        tag = "methodName"
    case psParms, psEndParms:
        tag = "params"
    case psParam, psEndParam:
        tag = "param"
    case psValue, psEndValue:
        tag = "value"
    default:
        tag = "???"
    }

    return tag, isEnd
}

func getValue(typeName string, b []byte) (interface{}, *XMLRPCError) {
    var unimplemented = &XMLRPCError{Msg:"Unimplemented"}

    valStr := string(b)
    if typeName == "array" {
        return nil, unimplemented
    } else if typeName == "base64" {
        return nil, unimplemented
    } else if typeName == "boolean" {
        if valStr == "1" {
            return true, nil
        } else if valStr == "0" {
            return false, nil
        } else {
            msg := fmt.Sprintf("Bad <boolean> value \"%s\"", valStr)
            return nil, &XMLRPCError{Msg:msg}
        }
    } else if typeName == "dateTime.iso8601" {
        return nil, unimplemented
    } else if typeName == "double" {
        f, err := strconv.Atof(valStr)
        if err != nil {
            return f, &XMLRPCError{Msg:err.String()}
        }

        return f, nil
    } else if typeName == "int" || typeName == "i4" {
        i, err := strconv.Atoi(valStr)
        if err != nil {
            return i, &XMLRPCError{Msg:err.String()}
        }

        return i, nil
    } else if typeName == "string" {
        return valStr, nil
    } else if typeName == "struct" {
        return nil, unimplemented
    }

    return nil, &XMLRPCError{Msg:fmt.Sprintf("Unknown type <%s> for \"%s\"",
            typeName, valStr)}
}

func unmarshalValue(p *xml.Parser) (interface{}, *XMLRPCError, bool) {
    var typeName string
    var rtnVal interface{}

    const noEndValTag = false

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return rtnVal, &XMLRPCError{Msg:err.String()}, noEndValTag
        }

        const debug = false
        if debug {
            var tokStr string
            if t2, ok := tok.(xml.CharData); ok {
                tokStr = string([]byte(t2))
            } else {
                tokStr = fmt.Sprintf("%v", tok)
            }

            fmt.Printf("ty %s rtn %v tok %s<%T>\n", typeName, rtnVal, tokStr,
                tok)
        }

        switch v := tok.(type) {
        case xml.StartElement:
            if typeName != "" {
                err := &XMLRPCError{Msg:fmt.Sprintf("Found multiple types" +
                        " (%s and %s) inside <value>", typeName, v.Name.Local)}
                return nil, err, noEndValTag
            }

            typeName = v.Name.Local
        case xml.EndElement:
            if typeName == "" && v.Name.Local == "value" {
                return "", nil, true
            } else if typeName != v.Name.Local {
                err := &XMLRPCError{Msg:fmt.Sprintf("Found unexpected </%s>" +
                        " (wanted </%s>)", v.Name.Local, typeName)}
                return nil, err, noEndValTag
            }

            if typeName == "string" && rtnVal == nil {
                rtnVal = ""
            }
            return rtnVal, nil, noEndValTag
        case xml.CharData:
            if typeName != "" && rtnVal == nil {
                var valErr *XMLRPCError
                rtnVal, valErr = getValue(typeName, v)
                if valErr != nil {
                    return rtnVal, valErr, noEndValTag
                }
            } else {
                for _, c := range v {
                    if !isSpace(c) {
                        if rtnVal == nil {
                            return string([]byte(v)), nil, noEndValTag
                        }

                        err := &XMLRPCError{Msg:fmt.Sprintf("Found" +
                                " non-whitespace chars \"%s\" inside <value>",
                                string([]byte(v)))}
                        return nil, err, noEndValTag
                    }
                }
            }
        default:
            err := &XMLRPCError{Msg:fmt.Sprintf("Not handling <value> %v" +
                    " (type %T)", v, v)}
            return nil, err, noEndValTag
        }
    }

    if typeName == "" {
        return rtnVal, &XMLRPCError{Msg:"No type found inside <value>"},
        noEndValTag
    }

    return rtnVal, &XMLRPCError{Msg:fmt.Sprintf("Closing tag not found for" +
        " <%s>", typeName)}, noEndValTag
}

func Unmarshal(r io.Reader) (string, interface{}, *XMLRPCError) {
    p := xml.NewParser(r)

    state := psMethod
    isResp := true
    stateTag := "???"
    wantEnd := false
    methodName := ""

    var rtnVal interface{}

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return methodName, rtnVal, &XMLRPCError{Msg:err.String()}
        }

        const debug = false
        if debug {
            var tokStr string
            if t2, ok := tok.(xml.CharData); ok {
                tokStr = string([]byte(t2))
            } else {
                tokStr = fmt.Sprintf("%v", tok)
            }

            fmt.Printf("ps %s tag %s wantEnd %v tok %s<%T>\n", state, stateTag,
                wantEnd, tokStr, tok)
        }

        switch v := tok.(type) {
        case xml.StartElement:
            if state == psMethod {
                if v.Name.Local == "methodResponse" {
                    state = psParms
                    isResp = true
                    stateTag, wantEnd = getStateVals(state, isResp)
                } else if v.Name.Local == "methodCall" {
                    state = psName
                    isResp = false
                    stateTag, wantEnd = getStateVals(state, isResp)
                } else {
                    err := &XMLRPCError{Msg:fmt.Sprintf("Unexpected initial" +
                            " tag <%s>", v.Name.Local)}
                    return methodName, rtnVal, err
                }
            } else if v.Name.Local == stateTag && ! wantEnd {
                if state != psValue {
                    state += 1
                    stateTag, wantEnd = getStateVals(state, isResp)
                } else {
                    var rtnErr *XMLRPCError
                    var sawEndValTag bool
                    rtnVal, rtnErr, sawEndValTag = unmarshalValue(p)
                    if rtnErr != nil {
                        return methodName, rtnVal, rtnErr
                    }
                    if ! sawEndValTag {
                        state += 1
                    } else {
                        state = psEndParam
                    }
                    stateTag, wantEnd = getStateVals(state, isResp)
                }
            } else {
                err := &XMLRPCError{Msg:fmt.Sprintf("Unexpected <%s> token" +
                        " for state %s", v.Name.Local, state)}
                return methodName, rtnVal, err
            }
        case xml.EndElement:
            if state == psEndMethod {
                if methodName == "" {
                    stateTag = "methodResponse"
                } else {
                    stateTag = "methodCall"
                }
                if v.Name.Local == stateTag {
                    state += 1
                    stateTag = "???"
                    wantEnd = false
                }
            } else if v.Name.Local == stateTag && wantEnd {
                state += 1
                stateTag, wantEnd = getStateVals(state, isResp)
            } else if state == psParam && ! wantEnd && v.Name.Local == "params" {
                state = psEndMethod
                stateTag, wantEnd = getStateVals(state, isResp)
            } else {
                err := &XMLRPCError{Msg:fmt.Sprintf("Unexpected </%s> token" +
                        " for state %s", v.Name.Local, state)}
                return methodName, rtnVal, err
            }
        case xml.CharData:
            if state == psInName {
                methodName = string([]byte(v))
                wantEnd = true
            } else {
                for _, c := range v {
                    if !isSpace(c) {
                        err := fmt.Sprintf("Found non-whitespace" +
                            " chars \"%s\" for state %s", string([]byte(v)),
                            state)
                        return methodName, rtnVal, &XMLRPCError{Msg:err}
                    }
                }
            }
        case xml.ProcInst:
            // ignored
        default:
            err := &XMLRPCError{Msg:fmt.Sprintf("Not handling %v (type %T)" +
                    " for state %s", v, v, state)}
            return methodName, rtnVal, err
        }
    }

    return methodName, rtnVal, nil
}

func UnmarshalString(s string) (string, interface{}, *XMLRPCError) {
    return Unmarshal(strings.NewReader(s))
}

func wrapParam(xval interface{}) (string, *XMLRPCError) {
    var valStr string

    switch val := xval.(type) {
    case bool:
        var bval int
        if val {
            bval = 1
        } else {
            bval = 0
        }
        valStr = fmt.Sprintf("<boolean>%d</boolean>", bval)
    case float:
        valStr = fmt.Sprintf("<double>%f</double>", val)
    case int:
        valStr = fmt.Sprintf("<int>%d</int>", val)
    default:
        err := fmt.Sprintf("Not wrapping type %T (%v)", val, val)
        return "", &XMLRPCError{Msg:err}
    }

    return fmt.Sprintf(`    <param>
      <value>
        %s
      </value>
    </param>
`, valStr), nil
}

func Marshal(methodName string, args ... interface{}) (string, *XMLRPCError) {
    var name, extra string
    if methodName == "" {
        name = "Response"
        extra = ""
    } else {
        name = "Call"
        extra = fmt.Sprintf("  <methodName>%s</methodName>\n", methodName)
    }

    xmlStr := fmt.Sprintf(`<?xml version="1.0"?>
<method%s>
%s  <params>
`, name, extra)

    for _, a := range args {
        valStr, err := wrapParam(a)
        if err != nil {
            return "", err
        }

        xmlStr += valStr
    }

    xmlStr += fmt.Sprintf(`  </params>
</method%s>
`, name)
    return xmlStr, nil
}

/********** From request.go ************/

type badStringError struct {
	what string
	str  string
}

func (e *badStringError) String() string { return fmt.Sprintf("%s %q", e.what, e.str) }

/********** From client.go ************/

// Given a string of the form "host", "host:port", or "[ipv6::address]:port",
// return true if the string includes a port.
func hasPort(s string) bool { return strings.LastIndex(s, ":") > strings.LastIndex(s, "]") }

// Used in Send to implement io.ReadCloser by bundling together the
// io.BufReader through which we read the response, and the underlying
// network connection.
type readClose struct {
	io.Reader
	io.Closer
}

// Send issues an HTTP request.  Caller should close resp.Body when done reading it.
//
// TODO: support persistent connections (multiple requests on a single connection).
// send() method is nonpublic because, when we refactor the code for persistent
// connections, it may no longer make sense to have a method with this signature.
func send(req *http.Request) (resp *http.Response, err os.Error) {
	if req.URL.Scheme != "http" {
		return nil, &badStringError{"unsupported protocol scheme", req.URL.Scheme}
	}

	addr := req.URL.Host
	if !hasPort(addr) {
		addr += ":http"
	}
	info := req.URL.Userinfo
	if len(info) > 0 {
		enc := base64.URLEncoding
		encoded := make([]byte, enc.EncodedLen(len(info)))
		enc.Encode(encoded, []byte(info))
		if req.Header == nil {
			req.Header = make(map[string]string)
		}
		req.Header["Authorization"] = "Basic " + string(encoded)
	}
	conn, err := net.Dial("tcp", "", addr)
	if err != nil {
		return nil, err
	}

	err = req.Write(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	resp, err = http.ReadResponse(reader, req.Method)
	if err != nil {
		conn.Close()
		return nil, err
	}

	resp.Body = readClose{resp.Body, conn}

	return
}

// Post issues a POST to the specified URL.
//
// Caller should close r.Body when done reading it.
// XXX - copied from http.Post, but 'body' is a string instead of io.Reader,
// XXX   req.TransferEncoding is not set and re.ContentLength is set to len(body)
func PostString(url string, bodyType string, body string) (r *http.Response, err os.Error) {
	var req http.Request
	req.Method = "POST"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Close = true
	req.Body = nopCloser{strings.NewReader(body)}
	req.Header = map[string]string{
		"Content-Type": bodyType,
	}

    req.RawURL = "/RPC2"
    req.ContentLength = int64(len(body))

	req.URL, err = http.ParseURL(url)
	if err != nil {
		return nil, err
	}

	return send(&req)
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() os.Error { return nil }
