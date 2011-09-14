package xmlrpc

import (
    "fmt"
    "http"
    "net"
    "os"
    "reflect"
    "rpc2"
)

type methodData struct {
    obj interface{}
    method reflect.Method
}

type XMLRPCHandler struct {
    codec rpc2.RPCCodec
    methods map[string]*methodData
}

func NewHandler(codec rpc2.RPCCodec) *XMLRPCHandler {
    return &XMLRPCHandler{codec: codec, methods: make(map[string]*methodData)}
}

func (h *XMLRPCHandler) Register(prefix string, obj interface{}) os.Error {
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

        h.methods[name] = &methodData{obj: obj, method: m}
    }

    return nil
}

func (h *XMLRPCHandler) ServeHTTP(conn http.ResponseWriter, req *http.Request) {
    methodName, params, err, ok := h.codec.UnserializeRequest(req.Body, conn)
    if !ok {
        return
    }

    if err != nil {
        h.codec.HandleError(conn, 1, fmt.Sprintf("Unmarshal error: %v", err))
        return
    }

    var args []interface{}

    if args, ok = params.([]interface{}); !ok {
        args := make([]interface{}, 1, 1)
        args[0] = params
    }

    var mData *methodData

    if mData, ok = h.methods[methodName]; !ok {
        h.codec.HandleError(conn, 2,
            fmt.Sprintf("Unknown method \"%s\"", methodName))
        return
    }

    if len(args) + 1 != mData.method.Type.NumIn() {
        h.codec.HandleError(conn, 3,
            fmt.Sprintf("Bad number of parameters for method \"%s\"," +
            "(%d != %d)", methodName, len(args), mData.method.Type.NumIn()))
        return
    }

    vals := make([]reflect.Value, len(args) + 1, len(args) + 1)

    vals[0] = reflect.ValueOf(mData.obj)
    for i := 1; i < mData.method.Type.NumIn(); i++ {
        expType := mData.method.Type.In(i)
        argType := reflect.TypeOf(args[i - 1])

        tmpVal := reflect.ValueOf(args[i - 1])
        if expType == argType {
            vals[i] = tmpVal
        } else {
            val, ok := h.codec.HandleTypeMismatch(tmpVal.Interface(), expType)
            if !ok {
                h.codec.HandleError(conn, 4,
                    fmt.Sprintf("Bad %s argument #%d (%v should be %v)",
                    methodName, i - 1, argType, expType))
                return
            }

            vals[i] = reflect.ValueOf(val)
        }
    }

    rtnVals := mData.method.Func.Call(vals)

    mArray := make([]interface{}, len(rtnVals), len(rtnVals))
    for i := 0; i < len(rtnVals); i++ {
        mArray[i] = rtnVals[i].Interface()
    }

    mBytes, merr := h.codec.SerializeResponse(mArray)
    if merr != nil {
        h.codec.HandleError(conn, 5, fmt.Sprintf("Marshal error: %v", merr))
        return
    }

    conn.Write(mBytes)
}

func StartServer(port int, h *XMLRPCHandler) net.Listener {
    l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        fmt.Printf("Listen failed: %v\n", err)
        return nil
    }

    go http.Serve(l, h)
    return l
}
