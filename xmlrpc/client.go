package xmlrpc

import (
    "bufio"
    "bytes"
    "fmt"
    "http"
    "io"
    "net"
    "os"
    "rpc2"
    "url"
)

type nopCloser struct {
    io.Reader
}

func (nopCloser) Close() os.Error { return nil }

type RPCClient struct {
    conn net.Conn
    url *url.URL
}

func (client *RPCClient) RPCCall(methodName string,
    args ... interface{}) (interface{}, *rpc2.Fault, *rpc2.Error) {
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
    req.Header = map[string][]string{
        "Content-Type": {"text/xml"},
    }
    req.RawURL = "/RPC2"
    req.ContentLength = int64(buf.Len())

    if client.conn == nil {
        var cerr *rpc2.Error
        if client.conn, cerr = rpc2.Open(client.url); cerr != nil {
            return nil, nil, cerr
        }
    }

    if werr := req.Write(client.conn); werr != nil {
        client.conn.Close()
        return nil, nil, &rpc2.Error{Msg:werr.String()}
    }

    reader := bufio.NewReader(client.conn)
    resp, rerr := http.ReadResponse(reader, &req)
    if rerr != nil {
        client.conn.Close()
        return nil, nil, &rpc2.Error{Msg:rerr.String()}
    } else if resp == nil {
        rrerr := fmt.Sprintf("ReadResponse for %s returned nil response\n",
            methodName)
        return nil, nil, &rpc2.Error{Msg:rrerr}
    }

    _, pval, perr, pfault := Unmarshal(resp.Body)

    if resp.Close {
        resp.Body.Close()
        client.conn = nil
    }

    return pval, pfault, perr
}

func (client *RPCClient) Close() {
    client.conn.Close()
}

func NewClient(host string, port int) (c *RPCClient, err *rpc2.Error) {
    conn, url, cerr := rpc2.OpenConnURL(host, port)
    if cerr != nil {
        return nil, &rpc2.Error{Msg:cerr.String()}
    }

    return &RPCClient{conn: conn, url: url}, nil
}
