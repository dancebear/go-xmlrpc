package xmlrpc

import (
    "bufio"
    "bytes"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/url"
    "reflect"
    "strconv"
    "strings"
    "encoding/xml"
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

func parseStruct(p *xml.Decoder) (interface{}, *Error, bool) {
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
            return nil, &Error{Msg:err.Error()}, false
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

func getValue(p *xml.Decoder, typeName string, b []byte) (interface{},
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
        f, err := strconv.ParseFloat(valStr, 64)
        if err != nil {
            return f, &Error{Msg:err.Error()}, false
        }

        return f, nil, false
    } else if typeName == "int" || typeName == "i4" {
        i, err := strconv.Atoi(valStr)
        if err != nil {
            return i, &Error{Msg:err.Error()}, false
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

func unmarshalValue(p *xml.Decoder) (interface{}, *Error, bool) {
    var typeName string
    var rtnVal interface{}

    const noEndValTag = false

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return rtnVal, &Error{Msg:err.Error()}, noEndValTag
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

func extractParams(v []interface{}) interface{} {
    if len(v) == 0 {
        return nil
    } else if len(v) == 1 {
        return v[0]
    }

    return v
}

// Translate an XML string into a local data object
func Unmarshal(r io.Reader) (string, interface{}, *Error, *Fault) {
    p := xml.NewDecoder(r)

    state := psMethod
    isResp := true
    stateTag := "???"
    wantEnd := false
    methodName := ""

    params := make([]interface{}, 0)

    isFault := false
    var faultVal *Fault

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return methodName, extractParams(params),
            &Error{Msg:err.Error()}, faultVal
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
                            &Error{Msg:err}, nil
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
                        params = append(params, uVal)
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
                        &Error{Msg:err}, faultVal
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
        case float32:
            valStr = fmt.Sprintf("<double>%f</double>", val)
        case float64:
            valStr = fmt.Sprintf("<double>%f</double>", val)
        case int:
            valStr = fmt.Sprintf("<int>%d</int>", val)
        case int16:
            valStr = fmt.Sprintf("<int>%d</int>", val)
        case int32:
            valStr = fmt.Sprintf("<int>%d</int>", val)
        case int64:
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

// Write a local data object as an XML-RPC request
func Marshal(w io.Writer, methodName string, args ... interface{}) *Error {
    return marshalArray(w, methodName, args)
}

func marshalArray(w io.Writer, methodName string, args []interface{}) *Error {
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

        fmt.Fprintf(w, "%s", valStr)
    }

    fmt.Fprintf(w, "  </params>\n</method%s>\n", name)

    return nil
}

type Client struct {
    http.Client
    urlStr string
}

func NewClient(host string, port int) (c *Client, err *Error) {
    address := fmt.Sprintf("http://%s:%d/RPC2", host, port)

    uurl, uerr := url.Parse(address)
    if uerr != nil {
        return nil, &Error{Msg:uerr.Error()}
    }

    return &Client{urlStr:uurl.String()}, nil
}

func (c *Client) RPCCall(methodName string,
    args ... interface{}) (interface{}, *Error, *Fault) {

    buf := bytes.NewBufferString("")
    berr := marshalArray(buf, methodName, args)
    if berr != nil {
        return nil, berr, nil
    }

    req, err := http.NewRequest("POST", c.urlStr,
        strings.NewReader(buf.String()))
    if err != nil {
        return nil, &Error{Msg:err.Error()}, nil
    }

    req.Header.Add("Content-Type", "text/xml")

    r, err := c.Do(req)
    if err != nil {
        return nil, &Error{Msg:err.Error()}, nil
    } else if r == nil {
        err := fmt.Sprintf("PostString for %s returned nil response\n",
            methodName)
        return nil, &Error{Msg:err}, nil
    }

    _, pval, perr, pfault := Unmarshal(r.Body)

    if r.Close {
        r.Body.Close()
    }

    return pval, perr, pfault
}

func DumpResponse(r io.Reader) (interface{}, string) {
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        fmt.Println(scanner.Text())
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

    return nil, ""
}

/* ----------------------- */

type methodData struct {
    obj interface{}
    method reflect.Method
}

type Handler struct {
    methods map[string]*methodData
}

func NewHandler() *Handler {
    h := new(Handler)
    h.methods = make(map[string]*methodData)
    return h
}

func (h *Handler) Register(prefix string, obj interface{}) error {
    ot := reflect.TypeOf(obj)

    for i := 0; i < ot.NumMethod(); i++ {
        m := ot.Method(i)
        if m.PkgPath != "" {
            continue
        }

        var name string
        if prefix == "" {
            name = m.Name
        } else {
            name = prefix + "." + m.Name
        }

        md := &methodData{obj: obj, method: m}
        h.methods[name] = md
        h.methods[strings.ToLower(name)] = md
    }

    return nil
}

func writeFault(out io.Writer, code int, msg string) {
    fmt.Fprintf(out, `<?xml version="1.0"?>
<methodResponse>
  <fault>
    <value>
        <struct>
          <member>
            <name>faultCode</name>
            <value><int>%d</int></value>
          </member>
          <member>
            <name>faultString</name>
            <value>%s</value>
          </member>
        </struct>
    </value>
  </fault>
</methodResponse>`, code, msg)
}

const errNotWellFormed = -32700
const errUnknownMethod = -32601
const errInvalidParams = -32602
const errInternal = -32603

func (h *Handler) handleRequest(resp http.ResponseWriter, req *http.Request) {
    methodName, params, err, fault := Unmarshal(req.Body)

    if err != nil {
        writeFault(resp, errNotWellFormed,
            fmt.Sprintf("Unmarshal error: %v", err))
        return
    } else if fault != nil {
        writeFault(resp, fault.Code, fault.Msg)
        return
    }

    var args []interface{}
    var ok bool

    if args, ok = params.([]interface{}); !ok {
        args := make([]interface{}, 1, 1)
        args[0] = params
    }

    var mData *methodData

    if mData, ok = h.methods[methodName]; !ok {
        writeFault(resp, errUnknownMethod,
            fmt.Sprintf("Unknown method \"%s\"", methodName))
        return
    }

    if len(args) + 1 != mData.method.Type.NumIn() {
        writeFault(resp, errInvalidParams,
            fmt.Sprintf("Bad number of parameters for method \"%s\"," +
                " (%d != %d)", methodName, len(args),
                mData.method.Type.NumIn()))
        return
    }

    vals := make([]reflect.Value, len(args) + 1, len(args) + 1)

    vals[0] = reflect.ValueOf(mData.obj)
    for i := 1; i < mData.method.Type.NumIn(); i++ {
        if mData.method.Type.In(i) != reflect.TypeOf(args[i - 1]) {
            writeFault(resp, errInvalidParams,
                fmt.Sprintf("Bad %s argument #%d (%v should be %v)",
                    methodName, i - 1, reflect.TypeOf(args[i - 1]),
                    mData.method.Type.In(i)))
            return
        }

        vals[i] = reflect.ValueOf(args[i - 1])
    }

    rtnVals := mData.method.Func.Call(vals)

    mArray := make([]interface{}, len(rtnVals), len(rtnVals))
    for i := 0; i < len(rtnVals); i++ {
        mArray[i] = rtnVals[i].Interface()
    }

    buf := bytes.NewBufferString("")
    err = marshalArray(buf, "", mArray)
    if err != nil {
        writeFault(resp, errInternal, fmt.Sprintf("Faied to marshal %s: %v",
            methodName, err))
        return
    }

    buf.WriteTo(resp)
}

func StartServer(port int) *Handler {
    h := NewHandler()
    http.HandleFunc("/", h.handleRequest)
    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
    return h
}
