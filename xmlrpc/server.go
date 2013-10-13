package xmlrpc

import (
    "bytes"
    "fmt"
    "net/http"
    "io"
    "reflect"
    "strings"
)

type methodData struct {
    obj interface{}
    method reflect.Method
}

// Map from XML-RPC procedure names to Go methods
type Handler struct {
    methods map[string]*methodData
}

// create a new handler mapping XML-RPC procedure names to Go methods
func NewHandler() *Handler {
    h := new(Handler)
    h.methods = make(map[string]*methodData)
    return h
}

// register all methods associated with the Go object, passing them
// through the name mapper if one is supplied
//
// The name mapper can return "" to ignore a method or transform the
// name as desired
func (h *Handler) Register(obj interface{}, mapper func(string)string) error {
    ot := reflect.TypeOf(obj)

    for i := 0; i < ot.NumMethod(); i++ {
        m := ot.Method(i)
        if m.PkgPath != "" {
            continue
        }

        var name string
        if mapper == nil {
            name = m.Name
        } else {
            name = mapper(m.Name)
            if name == "" {
                continue
            }
        }

        md := &methodData{obj: obj, method: m}
        h.methods[name] = md
        h.methods[strings.ToLower(name)] = md
    }

    return nil
}

// Return an XML-RPC fault
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

// semi-standard XML-RPC response codes
const (
    errNotWellFormed = -32700
    errUnknownMethod = -32601
    errInvalidParams = -32602
    errInternal = -32603
)

// handle an XML-RPC request
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

// start an XML-RPC server
func StartServer(port int) *Handler {
    h := NewHandler()
    http.HandleFunc("/", h.handleRequest)
    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
    return h
}
