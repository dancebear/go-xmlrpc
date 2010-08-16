package xmlrpc

import (
    "bytes"
    "fmt"
    "http"
    "io"
    "os"
    "rpc"
)

type xmlrpcCodec int

func (cli *xmlrpcCodec) ContentType() string { return "text/xml" }

func (cli *xmlrpcCodec) RawURL() string { return "/RPC2" }

func (cli *xmlrpcCodec) SerializeRequest(r *rpc.Request,
    params interface{}) (io.ReadWriter, int, os.Error) {
    var args []interface{}
    var ok bool
    if args, ok = params.([]interface{}); !ok {
        msg := fmt.Sprintf("Expected []interface{}, not %T", params)
        return nil, -1, os.NewError(msg)
    }

    buf := bytes.NewBufferString("")
    if xmlErr := marshalArray(buf, r.ServiceMethod, args);  xmlErr != nil {
        msg := fmt.Sprintf("WriteRequest(%v, %v) marshal failed: %s", r,
            params, xmlErr)
        return nil, -1, os.NewError(msg)
    }

    return buf, buf.Len(), nil
}

func (cli *xmlrpcCodec) UnserializeResponse(r io.Reader,
    x interface{}) os.Error {
    _, pval, perr, pfault := Unmarshal(r)
    if perr != nil {
        return os.NewError(perr.String())
    } else if pfault != nil {
        return os.NewError(pfault.String())
    }

    if replPtr, ok := x.(*interface{}); !ok {
        return os.NewError(fmt.Sprintf("Reply type is %T, not *interface{}",
            x))
    } else {
        *replPtr = pval
    }

    return nil
}

// NewXMLRPCClientCodec returns a new rpc.ClientCodec using XML-RPC on conn.
func NewXMLRPCClientCodec(conn io.ReadWriteCloser, url *http.URL) rpc.ClientCodec {
    return &httpClient{codec: new(xmlrpcCodec), conn: &conn, url: url, ready: make(chan uint64)}
}

// NewXMLRPCClient returns a new rpc.Client to handle requests to the
// set of services at the other end of the connection.
func NewXMLRPCClient(conn io.ReadWriteCloser, url *http.URL) *rpc.Client {
    return rpc.NewClientWithCodec(NewXMLRPCClientCodec(conn, url))
}
