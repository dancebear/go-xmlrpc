package xmlrpc

import (
    "bytes"
    "fmt"
    "http"
    "io"
    "json"
    "os"
    "reflect"
    "rpc"
)

type jsonrpcCodec int

func (cli *jsonrpcCodec) ContentType() string { return "application/json" }

func (cli *jsonrpcCodec) RawURL() string { return "/" }

func (cli *jsonrpcCodec) SerializeRequest(r *rpc.Request,
    params interface{}) (io.ReadWriter, int, os.Error) {
    var args []interface{}
    var ok bool
    if args, ok = params.([]interface{}); !ok {
        msg := fmt.Sprintf("Expected []interface{}, not %T", params)
        return nil, -1, os.NewError(msg)
    }

    jmap := make(map[string]interface{})
    jmap["method"] = r.ServiceMethod
    jmap["params"] = args
    jmap["id"] = r.Seq + 1
    byteArray, berr := json.MarshalForHTML(jmap)
    if berr != nil {
        return nil, -1, berr
    }

    buf := bytes.NewBufferString("")
    buf.Write(byteArray)

    return buf, buf.Len(), nil
}

func (cli *jsonrpcCodec) UnserializeResponse(r io.Reader,
    x interface{}) os.Error {
    buf := bytes.NewBufferString("")
    buf.ReadFrom(r)

    var pval interface{}
    perr := json.Unmarshal(buf.Bytes(), &pval)
    if perr != nil {
        return perr
    }

    if replPtr, ok := x.(*interface{}); !ok {
        return os.NewError(fmt.Sprintf("Reply type is %T, not *interface{}",
            x))
    } else if pmap, ok := pval.(map[string]interface{}); !ok {
        msg := fmt.Sprintf("Returned value is %T, not map[string]interface{}",
            pval)
        return os.NewError(msg)
    } else {
        if pmap["err"] != nil {
            return os.NewError(fmt.Sprintf("%v", pmap["err"]))
        }

        if val, ok := pmap["result"].([]interface{}); !ok {
            *replPtr = pmap["result"]
        } else if len(val) == 1 {
            *replPtr = val[0]
        } else {
            *replPtr = val
        }
    }

    return nil
}

func (cli *jsonrpcCodec) HandleError(conn *http.Conn, code int, msg string) {
    http.Error(conn, msg, http.StatusBadRequest)
}

func (cli *jsonrpcCodec) UnserializeRequest(r io.Reader,
    conn *http.Conn) (string, interface{}, os.Error, bool) {
    buf := bytes.NewBufferString("")
    buf.ReadFrom(r)

    var pval interface{}
    perr := json.Unmarshal(buf.Bytes(), &pval)
    if perr != nil {
        http.Error(conn, perr.String(), http.StatusBadRequest)
        return "", nil, nil, false
    }

    var pmap map[string]interface{}
    var ok bool

    if pmap, ok = pval.(map[string]interface{}); !ok {
        msg := fmt.Sprintf("Returned value is %T, not map[string]interface{}",
            pval)
        http.Error(conn, msg, http.StatusBadRequest)
        return "", nil, nil, false
    }

    var methodName string

    if methodName, ok = pmap["method"].(string); !ok {
        msg := fmt.Sprintf("Returned method name is %T, string",
            pmap["method"])
        http.Error(conn, msg, http.StatusBadRequest)
        return "", nil, nil, false
    }

    return methodName, pmap["params"], nil, true
}

func (cli *jsonrpcCodec) SerializeResponse(mArray []interface{}) ([]byte, os.Error) {
    jmap := make(map[string]interface{})
    jmap["result"] = mArray
    jmap["err"] = nil
    return json.MarshalForHTML(jmap)
}

func (cli *jsonrpcCodec) HandleTypeMismatch(origVal interface{}, expType reflect.Type) (interface{}, bool) {
    // work around brain-damaged json package

    var fval float64
    var ok bool
    if fval, ok = origVal.(float64); !ok {
        return nil, false
    }

    var cvtVal interface{}
    switch expType.Kind() {
    case reflect.Int:
        cvtVal = int(fval)
    case reflect.Int8:
        cvtVal = int8(fval)
    case reflect.Int16:
        cvtVal = int16(fval)
    case reflect.Int32:
        cvtVal = int32(fval)
    case reflect.Int64:
        cvtVal = int64(fval)
    case reflect.Uint:
        cvtVal = uint(fval)
    case reflect.Uint8:
        cvtVal = uint8(fval)
    case reflect.Uint16:
        cvtVal = uint16(fval)
    case reflect.Uint32:
        cvtVal = uint32(fval)
    case reflect.Uint64:
        cvtVal = uint64(fval)
    case reflect.Float:
        cvtVal = float(fval)
    case reflect.Float32:
        cvtVal = float32(fval)
    default:
        return nil, false
    }

    return cvtVal, true
}

func NewJSONRPCCodec() RPCCodec { return new(jsonrpcCodec) }
