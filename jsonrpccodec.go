package xmlrpc

import (
    "bytes"
    "fmt"
    "http"
    "io"
    "json"
    "os"
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

        *replPtr = pmap["result"]
    }

    return nil
}

// NewJSONRPCClientCodec returns a new rpc.ClientCodec using XML-RPC on conn.
func NewJSONRPCClientCodec(conn io.ReadWriteCloser, url *http.URL) rpc.ClientCodec {
    return &httpClient{codec: new(jsonrpcCodec), conn: &conn, url: url, ready: make(chan uint64)}
}

// NewJSONRPCClient returns a new rpc.Client to handle requests to the
// set of services at the other end of the connection.
func NewJSONRPCClient(conn io.ReadWriteCloser, url *http.URL) *rpc.Client {
    return rpc.NewClientWithCodec(NewJSONRPCClientCodec(conn, url))
}
