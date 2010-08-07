package xmlrpc

import (
    //"encoding/base64"
    "bufio"
    "bytes"
    "container/vector"
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

// An Error represents an internal failure in parsing or communication
type Error struct {
    Msg string
}

func (err *Error) String() string {
    if err == nil {
        return "NilError"
    }
    return err.Msg
}

// A Fault represents an error or exception in the procedure call
// being run on the remote machine
type Fault struct {
    Code int
    Msg string
}

func (f *Fault) String() string {
    if f == nil {
        return "NilFault"
    }
    return fmt.Sprintf("%s (code#%d)", f.Msg, f.Code)
}

type parseState int

var psStrings = []string { "Method", "MethodName", "InName", "Params",
    "Param", "Value", "EndValue", "EndParam", "EndParams", "EndMethod", "???", }

func (ps parseState) String() string { return psStrings[ps] }

const (
    psMethod parseState = iota
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

type structState int

var ssStrings = []string { "Initial", "Member", "Name", "InName", "EndName",
    "Value", "InValue", "EndValue", "EndMember", "EndStruct", "???", }

func (ss structState) String() string { return ssStrings[ss] }

const (
    stInitial = iota
    stMember
    stName
    stInName
    stEndName
    stValue
    stInValue
    stEndValue
    stEndMember
    stEndStruct
)

func getStateVals(st parseState, isResp bool) (string, bool) {
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

func getNextStructState(state structState) (structState, string, bool) {
    state += 1
    var stateTag string
    isEnd := state == stEndName || state == stEndValue ||
        state == stEndMember || state == stEndStruct

    switch state {
    case stMember, stEndMember:
        stateTag = "member"
    case stName, stEndName:
        stateTag = "name"
    case stValue, stEndValue:
        stateTag = "value"
    case stEndStruct:
        stateTag = "struct"
    default:
        stateTag = ""
    }

    return state, stateTag, isEnd
}

func parseStruct(p *xml.Parser) (interface{}, *Error, bool) {
    state, stateTag, wantEnd := getNextStructState(stInitial)

    key := ""

    valMap := make(map[string]interface{})

    finished := false
    for ! finished {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return nil, &Error{Msg:err.String()}, false
        }

        const debug = false
        if debug {
            var tokStr string
            if t2, ok := tok.(xml.CharData); ok {
                tokStr = string([]byte(t2))
            } else {
                tokStr = fmt.Sprintf("%v", tok)
            }

            fmt.Printf("st %v tag %s wantEnd %v tok %s<%T>\n", state, stateTag,
                wantEnd, tokStr, tok)
        }

        switch v := tok.(type) {
        case xml.StartElement:
            if wantEnd && state == stEndStruct && v.Name.Local == "member" {
                state = stMember
            } else if wantEnd || v.Name.Local != stateTag {
                err := fmt.Sprintf("Expected struct tag <%s>, not <%s>",
                    stateTag, v.Name.Local)
                return nil, &Error{Msg:err}, false
            }

            if state == stValue {
                val, err, sawEndValTag := unmarshalValue(p)
                if err != nil {
                    return nil, err, false
                }

                valMap[key] = val

                if sawEndValTag {
                    state = stEndValue
                } else {
                    state = stEndValue - 1
                }
            }

            state, stateTag, wantEnd = getNextStructState(state)

        case xml.EndElement:
            if wantEnd && state == stEndStruct && v.Name.Local == "member" {
                state = stMember
            } else if ! wantEnd || v.Name.Local != stateTag {
                err := fmt.Sprintf("Expected struct tag </%s>, not </%s>",
                    stateTag, v.Name.Local)
                return nil, &Error{Msg:err}, false
            }

            if state == stEndStruct {
                finished = true
            } else {
                state, stateTag, wantEnd = getNextStructState(state)
            }
        case xml.CharData:
            if state == stInName {
                key = string([]byte(v))
                state, stateTag, wantEnd = getNextStructState(state)
            } else {
                ignore := true
                for _, c := range v {
                    if !isSpace(c) {
                        ignore = false
                        break
                    }
                }

                if ! ignore {
                    err := &Error{Msg:fmt.Sprintf("Found" +
                            " non-whitespace chars \"%s\" inside <struct>",
                            string([]byte(v)))}
                    return nil, err, false
                }
            }
        }
    }

    return valMap, nil, true
}

func getValue(p *xml.Parser, typeName string, b []byte) (interface{},
    *Error, bool) {
    var unimplemented = &Error{Msg:"Unimplemented"}

    valStr := string(b)
    if typeName == "array" {
        return nil, unimplemented, false
    } else if typeName == "base64" {
        return nil, unimplemented, false
    } else if typeName == "boolean" {
        if valStr == "1" {
            return true, nil, false
        } else if valStr == "0" {
            return false, nil, false
        } else {
            msg := fmt.Sprintf("Bad <boolean> value \"%s\"", valStr)
            return nil, &Error{Msg:msg}, false
        }
    } else if typeName == "dateTime.iso8601" {
        return nil, unimplemented, false
    } else if typeName == "double" {
        f, err := strconv.Atof(valStr)
        if err != nil {
            return f, &Error{Msg:err.String()}, false
        }

        return f, nil, false
    } else if typeName == "int" || typeName == "i4" {
        i, err := strconv.Atoi(valStr)
        if err != nil {
            return i, &Error{Msg:err.String()}, false
        }

        return i, nil, false
    } else if typeName == "string" {
        return valStr, nil, false
    } else if typeName == "struct" {
        return parseStruct(p)
    }

    return nil, &Error{Msg:fmt.Sprintf("Unknown type <%s> for \"%s\"",
            typeName, valStr)}, false
}

func unmarshalValue(p *xml.Parser) (interface{}, *Error, bool) {
    var typeName string
    var rtnVal interface{}

    const noEndValTag = false

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return rtnVal, &Error{Msg:err.String()}, noEndValTag
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
                err := &Error{Msg:fmt.Sprintf("Found multiple types" +
                        " (%s and %s) inside <value>", typeName, v.Name.Local)}
                return nil, err, noEndValTag
            }

            typeName = v.Name.Local
        case xml.EndElement:
            if typeName == "" && v.Name.Local == "value" {
                return "", nil, true
            } else if typeName != v.Name.Local {
                err := &Error{Msg:fmt.Sprintf("Found unexpected </%s>" +
                        " (wanted </%s>)", v.Name.Local, typeName)}
                return nil, err, noEndValTag
            }

            if typeName == "string" && rtnVal == nil {
                rtnVal = ""
            }
            return rtnVal, nil, noEndValTag
        case xml.CharData:
            if typeName != "" && rtnVal == nil {
                var valErr *Error
                var sawEndTypeTag bool
                rtnVal, valErr, sawEndTypeTag = getValue(p, typeName, v)
                if valErr != nil {
                    return rtnVal, valErr, noEndValTag
                }

                if sawEndTypeTag {
                    return rtnVal, nil, noEndValTag
                }
            } else {
                for _, c := range v {
                    if !isSpace(c) {
                        if rtnVal == nil {
                            return string([]byte(v)), nil, noEndValTag
                        }

                        err := &Error{Msg:fmt.Sprintf("Found" +
                                " non-whitespace chars \"%s\" inside <value>",
                                string([]byte(v)))}
                        return nil, err, noEndValTag
                    }
                }
            }
        default:
            err := &Error{Msg:fmt.Sprintf("Not handling <value> %v" +
                    " (type %T)", v, v)}
            return nil, err, noEndValTag
        }
    }

    if typeName == "" {
        return rtnVal, &Error{Msg:"No type found inside <value>"},
        noEndValTag
    }

    return rtnVal, &Error{Msg:fmt.Sprintf("Closing tag not found for" +
        " <%s>", typeName)}, noEndValTag
}

func extractParams(v *vector.Vector) interface{} {
    if v.Len() == 0 {
        return nil
    } else if v.Len() == 1 {
        return v.At(0)
    }

    pList := make([]interface{}, v.Len(), v.Len())
    for i := 0; i < v.Len(); i++ {
        pList[i] = v.At(i)
    }

    return pList
}

// Translate an XML string into a local data object
func Unmarshal(r io.Reader) (string, interface{}, *Error, *Fault) {
    p := xml.NewParser(r)

    state := psMethod
    isResp := true
    stateTag := "???"
    wantEnd := false
    methodName := ""

    params := new(vector.Vector)

    isFault := false
    var faultVal *Fault

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return methodName, extractParams(params),
            &Error{Msg:err.String()}, faultVal
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
                    err := &Error{Msg:fmt.Sprintf("Unexpected initial" +
                            " tag <%s>", v.Name.Local)}
                    return methodName, extractParams(params), err, faultVal
                }
            } else if v.Name.Local == stateTag && ! wantEnd {
                if state != psValue {
                    state += 1
                    stateTag, wantEnd = getStateVals(state, isResp)
                } else {
                    var uVal interface{}
                    var uErr *Error
                    var sawEndValTag bool
                    uVal, uErr, sawEndValTag = unmarshalValue(p)
                    if uErr != nil {
                        return methodName, extractParams(params), uErr, faultVal
                    }
                    if isFault {
                        if uVal == nil {
                            err := &Error{Msg:"No fault value returned"}
                            return methodName, extractParams(params), err, nil
                        }

                        if fmap, ok := uVal.(map[string]interface{}); ! ok {
                            err := fmt.Sprintf("Bad type %T for fault", uVal)
                            return methodName, extractParams(params),
                            &Error{Msg:err},
                            nil
                        } else {
                            if code, ok := fmap["faultCode"].(int); ! ok {
                                err := fmt.Sprintf("Fault code should be an" +
                                    " int, not %T", code)
                                return methodName, extractParams(params),
                                &Error{Msg:err}, nil
                            } else if msg, ok := fmap["faultString"].(string);
                            ! ok {
                                err := fmt.Sprintf("Fault string should be a" +
                                    " string, not %T", msg)
                                return methodName, extractParams(params),
                                &Error{Msg:err}, nil
                            } else {
                                faultVal = &Fault{Code:code, Msg:msg}
                            }
                        }

                        if ! sawEndValTag {
                            state += 1
                            stateTag, wantEnd = getStateVals(state, isResp)
                        } else {
                            state = psEndParms
                            stateTag = "fault"
                            wantEnd = true
                        }
                    } else {
                        params.Push(uVal)
                        if ! sawEndValTag {
                            state += 1
                        } else {
                            state = psEndParam
                        }
                        stateTag, wantEnd = getStateVals(state, isResp)
                    }
                }
            } else if state == psParms && v.Name.Local == "fault" {
                isFault = true
                state = psValue
                stateTag, wantEnd = getStateVals(state, isResp)
            } else if wantEnd && state == psEndParms &&
                v.Name.Local == "param" {
                state = psValue
                stateTag, wantEnd = getStateVals(state, isResp)
            } else {
                err := &Error{Msg:fmt.Sprintf("Unexpected <%s> token" +
                        " for state %s", v.Name.Local, state)}
                return methodName, extractParams(params), err, faultVal
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
                if isFault && state == psEndValue {
                    state = psEndParms
                    stateTag = "fault"
                    wantEnd = true
                } else {
                    state += 1
                    stateTag, wantEnd = getStateVals(state, isResp)
                }
            } else if state == psParam && ! wantEnd &&
                v.Name.Local == "params" {
                state = psEndMethod
                stateTag, wantEnd = getStateVals(state, isResp)
            } else {
                err := &Error{Msg:fmt.Sprintf("Unexpected </%s> token" +
                        " for state %s", v.Name.Local, state)}
                return methodName, extractParams(params), err, faultVal
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
                        return methodName, extractParams(params),
                        &Error{Msg:err},
                            faultVal
                    }
                }
            }
        case xml.ProcInst:
            // ignored
        default:
            err := &Error{Msg:fmt.Sprintf("Not handling %v (type %T)" +
                    " for state %s", v, v, state)}
            return methodName, extractParams(params), err, faultVal
        }
    }

    return methodName, extractParams(params), nil, faultVal
}

// Translate an XML string into a local data object
func UnmarshalString(s string) (string, interface{}, *Error,
    *Fault) {
    return Unmarshal(strings.NewReader(s))
}

func wrapParam(xval interface{}) (string, *Error) {
    var valStr string

    if xval == nil {
        valStr = "<nil/>"
    } else {
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
        case string:
            valStr = fmt.Sprintf("<string>%s</string>", val)
        default:
            err := fmt.Sprintf("Not wrapping type %T (%v)", val, val)
            return "", &Error{Msg:err}
        }
    }

    return fmt.Sprintf(`    <param>
      <value>
        %s
      </value>
    </param>
`, valStr), nil
}

func Marshal(w io.Writer, methodName string, args ... interface{}) *Error {
    var name string
    var addExtra bool
    if methodName == "" {
        name = "Response"
        addExtra = false
    } else {
        name = "Call"
        addExtra = true
    }

    fmt.Fprintf(w, "<?xml version=\"1.0\"?>\n<method%s>\n", name)
    if addExtra {
        fmt.Fprintf(w, "  <methodName>%s</methodName>\n", methodName)
    }

    fmt.Fprintf(w, "  <params>\n")

    for _, a := range args {
        valStr, err := wrapParam(a)
        if err != nil {
            return err
        }

        fmt.Fprintf(w, valStr)
    }

    fmt.Fprintf(w, "  </params>\n</method%s>\n", name)

    return nil
}

/********** From http/client.go ************/

// Given a string of the form "host", "host:port", or "[ipv6::address]:port",
// return true if the string includes a port.
func hasPort(s string) bool {
    return strings.LastIndex(s, ":") > strings.LastIndex(s, "]")
}


type nopCloser struct {
    io.Reader
}

func (nopCloser) Close() os.Error { return nil }

/********** From http/client.go ************/

type Client struct {
    conn net.Conn
    url *http.URL
}

func open(url *http.URL) (net.Conn, *Error) {
    if url.Scheme != "http" {
        return nil, &Error{Msg:fmt.Sprintf("Only supporting \"http\"," +
                " not \"%s\"", url.Scheme)}
    }

    addr := url.Host
    if !hasPort(addr) {
        addr += ":http"
    }

    conn, cerr := net.Dial("tcp", "", addr)
    if cerr != nil {
        return nil, &Error{Msg:cerr.String()}
    }

    return conn, nil
}

func NewClient(host string, port int) (c *Client, err *Error) {
    address := fmt.Sprintf("http://%s:%d", host, port)

    url, uerr := http.ParseURL(address)
    if uerr != nil {
        return nil, &Error{Msg:err.String()}
    }

    var client Client

    var cerr *Error
    if client.conn, cerr = open(url); cerr != nil {
        return nil, cerr
   }

    client.url = url

    return &client, nil
}

func (client *Client) RPCCall(methodName string,
    args ... interface{}) (interface{}, *Fault, *Error) {
    buf := bytes.NewBufferString("")
    berr := Marshal(buf, methodName, args)
    if berr != nil {
        return nil, nil, berr
    }

    var req http.Request
    req.URL = client.url
    req.Method = "POST"
    req.ProtoMajor = 1
    req.ProtoMinor = 1
    req.Close = false
    req.Body = nopCloser{buf}
    req.Header = map[string]string{
        "Content-Type": "text/xml",
    }
    req.RawURL = "/RPC2"
    req.ContentLength = int64(buf.Len())

    if client.conn == nil {
        var cerr *Error
        if client.conn, cerr = open(client.url); cerr != nil {
            return nil, nil, cerr
        }
    }

    if werr := req.Write(client.conn); werr != nil {
        client.conn.Close()
        return nil, nil, &Error{Msg:werr.String()}
    }

    reader := bufio.NewReader(client.conn)
    resp, rerr := http.ReadResponse(reader, req.Method)
    if rerr != nil {
        client.conn.Close()
        return nil, nil, &Error{Msg:rerr.String()}
    } else if resp == nil {
        rrerr := fmt.Sprintf("ReadResponse for %s returned nil response\n",
            methodName)
        return nil, nil, &Error{Msg:rrerr}
    }

    _, pval, perr, pfault := Unmarshal(resp.Body)

    if resp.Close {
        resp.Body.Close()
        client.conn = nil
    }

    return pval, pfault, perr
}

func (client *Client) Close() {
    client.conn.Close()
}
