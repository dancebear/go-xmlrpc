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

type ParseState int

var stateName = []string { "Method", "Params", "Param", "Value",
    "EndValue", "EndParam", "EndParams", "EndMethod" }

func (ps ParseState) String() string { return stateName[ps] }

const (
    psMethod ParseState = iota
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

    var key string
    switch st {
    case psMethod, psEndMethod:
        if isResp {
            key = "methodResponse"
        } else {
            key = "methodRequest"
        }
    case psParms, psEndParms:
        key = "params"
    case psParam, psEndParam:
        key = "param"
    case psValue, psEndValue:
        key = "value"
    default:
        key = "???"
    }

    return key, isEnd
}

func getValue(typeName string, b []byte) (interface{}, string) {
    valStr := string(b)
    if typeName == "boolean" {
        if valStr == "1" {
            return true, ""
        } else if valStr == "0" {
            return false, ""
        } else {
            return nil, fmt.Sprintf("Bad <boolean> value \"%s\"", valStr)
        }
    } else if typeName == "double" {
        f, err := strconv.Atof(valStr)
        if err != nil {
            return f, err.String()
        }

        return f, ""
    } else if typeName == "int" || typeName == "i4" {
        i, err := strconv.Atoi(valStr)
        if err != nil {
            return i, err.String()
        }

        return i, ""
    } else if typeName == "string" {
        return valStr, ""
    }

    return nil, fmt.Sprintf("Unknown type <%s> for \"%s\"", typeName, valStr)
}

func parseValue(p *xml.Parser) (interface{}, string, bool) {
    var typeName string
    var rtnVal interface{}

    const noEndValTag = false

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return rtnVal, err.String(), noEndValTag
        }

        switch v := tok.(type) {
        case xml.StartElement:
            if typeName != "" {
                return nil, fmt.Sprintf("Found multiple types (%s and %s)" +
                    " inside <value>", typeName, v.Name.Local), noEndValTag
            }

            typeName = v.Name.Local
        case xml.EndElement:
            if typeName == "" && v.Name.Local == "value" {
                return "", "", true
            } else if typeName != v.Name.Local {
                return nil, fmt.Sprintf("Found unexpected </%s> (wanted </%s>)",
                    v.Name.Local, typeName), noEndValTag
            }

            if typeName == "string" && rtnVal == nil {
                rtnVal = ""
            }
            return rtnVal, "", noEndValTag
        case xml.CharData:
            if typeName != "" && rtnVal == nil {
                var valErr string
                rtnVal, valErr = getValue(typeName, v)
                if valErr != "" {
                    return rtnVal, valErr, noEndValTag
                }
            } else {
                for _, c := range v {
                    if !isSpace(c) {
                        if rtnVal == nil {
                            var valErr string
                            rtnVal, valErr = getValue("string", v)
                            return rtnVal, valErr, noEndValTag
                        }

                        return nil, "Found non-whitespace chars inside <value>",
                        noEndValTag
                    }
                }
            }
        default:
            return nil, fmt.Sprintf("Not handling <value> %v (type %T)", v, v),
            noEndValTag
        }
    }

    if typeName == "" {
        return rtnVal, "No type found inside <value>", noEndValTag
    }

    return rtnVal, fmt.Sprintf("Closing tag not found for <%s>", typeName),
    noEndValTag
}

func Parse(r io.Reader, isResp bool) (interface{}, string) {
    p := xml.NewParser(r)

    state := psMethod
    stateKey, wantEnd := getStateVals(state, isResp)

    var rtnVal interface{}

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return rtnVal, err.String()
        }

        //fmt.Printf("ps %s key %s wantEnd %v tok %v<%T>\n", state, stateKey,
        //    wantEnd, tok, tok)

        switch v := tok.(type) {
        case xml.StartElement:
            if v.Name.Local == stateKey && ! wantEnd {
                if state != psValue {
                    state += 1
                    stateKey, wantEnd = getStateVals(state, isResp)
                } else {
                    var rtnErr string
                    var sawEndValTag bool
                    rtnVal, rtnErr, sawEndValTag = parseValue(p)
                    if rtnErr != "" {
                        return nil, rtnErr
                    }
                    if ! sawEndValTag {
                        state += 1
                    } else {
                        state = psEndParam
                    }
                    stateKey, wantEnd = getStateVals(state, isResp)
                }
            } else {
                return nil, fmt.Sprintf("Unexpected <%s> token for state %s",
                    v.Name.Local, state)
            }
        case xml.EndElement:
            if v.Name.Local == stateKey && wantEnd {
                state += 1
                stateKey, wantEnd = getStateVals(state, isResp)
            } else if state == psParam && ! wantEnd && v.Name.Local == "params" {
                state = psEndMethod
                stateKey, wantEnd = getStateVals(state, isResp)
            } else {
                return nil, fmt.Sprintf("Unexpected </%s> token for state %s",
                    v.Name.Local, state)
            }
        case xml.CharData:
            for _, c := range v {
                if !isSpace(c) {
                    return nil,
                    fmt.Sprintf("Found non-whitespace chars for state %s", state)
                }
            }
        case xml.ProcInst:
            // ignored
        default:
            return nil, fmt.Sprintf("Not handling %v (type %T) for state %s",
                v, v, state)
        }
    }

    return rtnVal, ""
}

func ParseString(s string, isResp bool) (interface{}, string) {
    return Parse(strings.NewReader(s), isResp)
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
