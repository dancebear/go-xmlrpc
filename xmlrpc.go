package xmlrpc

import (
    //"encoding/base64"
    "bufio"
    "bytes"
    "fmt"
    "http"
    "io"
    "net"
    "os"
    "reflect"
    "rpc"
    "strings"
)

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

/* ========================================= */

type RPCCodec interface {
    ContentType() string
    RawURL() string
    SerializeRequest(r *rpc.Request, params interface{}) (io.ReadWriter,
        int, os.Error)
    UnserializeResponse(r io.Reader, x interface{}) os.Error
    HandleError(conn *http.Conn, code int, msg string)
    UnserializeRequest(r io.Reader, conn *http.Conn) (string, interface{},
        os.Error, bool)
    SerializeResponse(mArray []interface{}) ([]byte, os.Error)
    HandleTypeMismatch(origVal interface{},
        expType reflect.Type) (interface{}, bool)
}

type httpClient struct {
    codec RPCCodec
    url *http.URL
    conn *io.ReadWriteCloser
    resp *http.Response
    seq uint64
	ready chan uint64
}

func (cli *httpClient) WriteRequest(r *rpc.Request, params interface{}) os.Error {
    if cli.seq != 0 {
        return os.NewError("Only one XML-RPC call at a time is allowed")
    }
    cli.seq = r.Seq

    if cli.conn == nil {
        conn, err := open(cli.url)
        if err != nil {
            return os.NewError(err.String())
        }

        if ioConn, ok := conn.(io.ReadWriteCloser); !ok {
            errMsg := fmt.Sprintf("open() returned bad connection type %T",
                conn)
            return os.NewError(errMsg)
        } else {
            cli.conn = &ioConn
        }
    }

    rw, len, rerr := cli.codec.SerializeRequest(r, params)
    if rerr != nil {
        return rerr
    }

    var req http.Request
    req.URL = cli.url
    req.Method = "POST"
    req.ProtoMajor = 1
    req.ProtoMinor = 1
    req.Close = false
    req.Body = nopCloser{rw}
    req.Header = map[string]string{
        "Content-Type": cli.codec.ContentType(),
    }
    req.RawURL = cli.codec.RawURL()
    req.ContentLength = int64(len)

    if werr := req.Write(*cli.conn); werr != nil {
        return werr
    }

    cli.ready <- r.Seq

    return nil
}

func (cli *httpClient) ReadResponseHeader(r *rpc.Response) os.Error {
    <-cli.ready

    if cli.conn == nil {
        return os.NewError("Client connection is nil")
    }

    reader := bufio.NewReader(*cli.conn)

    resp, rerr := http.ReadResponse(reader, "POST")
    if rerr != nil {
        return rerr
    } else if resp == nil {
        return os.NewError("ReadResponse returned nil response")
    }

    r.Seq = cli.seq

    cli.resp = resp
    return nil
}

func (cli *httpClient) ReadResponseBody(x interface{}) os.Error {
    perr := cli.codec.UnserializeResponse(cli.resp.Body, x)
    cli.seq = 0

    if cli.resp.Close {
        cli.resp.Body.Close()
        cli.conn = nil
    }

    return perr
}

func (cli *httpClient) Close() os.Error {
    return cli.conn.Close()
}

/* ----------------------------------------- */

// NewRPCClientCodec returns a new rpc.ClientCodec for the RPCCodec
func NewRPCClientCodec(codec RPCCodec, conn io.ReadWriteCloser, url *http.URL) rpc.ClientCodec {
    return &httpClient{codec: codec, conn: &conn, url: url, ready: make(chan uint64)}
}

// NewNRPCClient returns a new rpc.Client to handle requests to the
// set of services at the other end of the connection.
func NewRPCClient(codec RPCCodec, conn io.ReadWriteCloser,
    url *http.URL) *rpc.Client {
    return rpc.NewClientWithCodec(NewRPCClientCodec(codec, conn, url))
}

/* ----------------------------------------- */

func openConnURL(host string, port int) (net.Conn, *http.URL, os.Error) {
    address := fmt.Sprintf("%s:%d", host, port)

    conn, cerr := net.Dial("tcp", "", address)
    if cerr != nil {
        return nil, nil, cerr
    }

    url, uerr := http.ParseURL("http://" + address)
    if uerr != nil {
        return nil, nil, uerr
    }

    return conn, url, nil
}

// Dial connects to an XML-RPC server at the specified network address.
func Dial(host string, port int, codec RPCCodec) (*rpc.Client, os.Error) {
    conn, url, cerr := openConnURL(host, port)
    if cerr != nil {
        return nil, &Error{Msg:cerr.String()}
    }

    return NewRPCClient(codec, conn, url), nil
}

/********** From http/client.go ************/

func openClient(url *http.URL) (net.Conn, *Error) {
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

type RPCClient struct {
    conn net.Conn
    url *http.URL
}

func (client *RPCClient) RPCCall(methodName string,
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

func (client *RPCClient) Close() {
    client.conn.Close()
}

func NewClient(host string, port int) (c *RPCClient, err *Error) {
    conn, url, cerr := openConnURL(host, port)
    if cerr != nil {
        return nil, &Error{Msg:cerr.String()}
    }

    return &RPCClient{conn: conn, url: url}, nil
}

/* ----------------------- */

type methodData struct {
    obj interface{}
    method reflect.Method
}

type XMLRPCHandler struct {
    codec RPCCodec
    methods map[string]*methodData
}

func NewHandler(codec RPCCodec) *XMLRPCHandler {
    return &XMLRPCHandler{codec: codec, methods: make(map[string]*methodData)}
}

func (h *XMLRPCHandler) Register(prefix string, obj interface{}) os.Error {
    ot := reflect.Typeof(obj)

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

func (h *XMLRPCHandler) ServeHTTP(conn *http.Conn, req *http.Request) {
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

    vals[0] = reflect.NewValue(mData.obj)
    for i := 1; i < mData.method.Type.NumIn(); i++ {
        expType := mData.method.Type.In(i)
        argType := reflect.Typeof(args[i - 1])

        tmpVal := reflect.NewValue(args[i - 1])
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

            vals[i] = reflect.NewValue(val)
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
