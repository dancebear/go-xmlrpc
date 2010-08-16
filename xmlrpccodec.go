package xmlrpc

import (
    "bytes"
    "fmt"
    "http"
    "io"
    "os"
    "reflect"
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

func (cli *xmlrpcCodec) HandleError(conn *http.Conn, code int, msg string) {
    fStr := fmt.Sprintf(`<?xml version="1.0"?>
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
    conn.Write(bytes.NewBufferString(fStr).Bytes())
    // XXX - figure out how we should really get bytes from a string
}

func (cli *xmlrpcCodec) UnserializeRequest(r io.Reader,
    conn *http.Conn) (string, interface{}, os.Error, bool) {
    methodName, params, err, fault := Unmarshal(r)

    if err != nil {
        return "", nil, os.NewError(err.String()), true
    } else if fault != nil {
        cli.HandleError(conn, fault.Code, fault.Msg)
        return "", nil, nil, false
    }

    return methodName, params, nil, true
}

func (cli *xmlrpcCodec) SerializeResponse(mArray []interface{}) ([]byte, os.Error) {
    buf := bytes.NewBufferString("")
    err := marshalArray(buf, "", mArray)
    if err != nil {
        return nil, os.NewError(err.String())
    }

    return buf.Bytes(), nil
}

func (cli *xmlrpcCodec) HandleTypeMismatch(origVal interface{}, expType reflect.Type) (interface{}, bool) {
    return nil, false
}

func NewXMLRPCCodec() RPCCodec { return new(xmlrpcCodec) }
