package rpc2

import (
    //"encoding/base64"
    "bufio"
    "fmt"
    "http"
    "io"
    "net"
    "os"
    "reflect"
    "rpc"
    "strings"
    "url"
)

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
    url *url.URL
}

func Open(url *url.URL) (net.Conn, *Error) {
    if url.Scheme != "http" {
        return nil, &Error{Msg:fmt.Sprintf("Only supporting \"http\"," +
                " not \"%s\"", url.Scheme)}
    }

    addr := url.Host
    if !hasPort(addr) {
        addr += ":http"
    }

    conn, cerr := net.Dial("tcp", addr)
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
    HandleError(conn http.ResponseWriter, code int, msg string)
    UnserializeRequest(r io.Reader, conn http.ResponseWriter) (string, interface{},
        os.Error, bool)
    SerializeResponse(mArray []interface{}) ([]byte, os.Error)
    HandleTypeMismatch(origVal interface{},
        expType reflect.Type) (interface{}, bool)
}

type httpClient struct {
    codec RPCCodec
    url *url.URL
    conn *io.ReadWriteCloser
    req *http.Request
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
        conn, err := Open(cli.url)
        if err != nil {
            return os.NewError(err.String())
        }

        if ioConn, ok := conn.(io.ReadWriteCloser); !ok {
            errMsg := fmt.Sprintf("Open() returned bad connection type %T",
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
    req.Header = map[string][]string{
        "Content-Type": {cli.codec.ContentType()},
    }
    req.RawURL = cli.codec.RawURL()
    req.ContentLength = int64(len)

    if werr := req.Write(*cli.conn); werr != nil {
        return werr
    }

    cli.req = &req

    cli.ready <- r.Seq

    return nil
}

func (cli *httpClient) ReadResponseHeader(r *rpc.Response) os.Error {
    <-cli.ready

    if cli.conn == nil {
        return os.NewError("Client connection is nil")
    }

    reader := bufio.NewReader(*cli.conn)

    resp, rerr := http.ReadResponse(reader, cli.req)
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
     fmt.Printf("Not closing %v <%T>\n", cli.conn, cli.conn)
     //return cli.conn.Close()
     return nil
}

/* ----------------------------------------- */

// NewRPCClientCodec returns a new rpc.ClientCodec for the RPCCodec
func NewRPCClientCodec(codec RPCCodec, conn io.ReadWriteCloser, url *url.URL) rpc.ClientCodec {
    return &httpClient{codec: codec, conn: &conn, url: url, ready: make(chan uint64)}
}

// NewNRPCClient returns a new rpc.Client to handle requests to the
// set of services at the other end of the connection.
func NewRPCClient(codec RPCCodec, conn io.ReadWriteCloser,
    url *url.URL) *rpc.Client {
    return rpc.NewClientWithCodec(NewRPCClientCodec(codec, conn, url))
}

/* ----------------------------------------- */

func OpenConnURL(host string, port int) (net.Conn, *url.URL, os.Error) {
    address := fmt.Sprintf("%s:%d", host, port)

    conn, cerr := net.Dial("tcp", address)
    if cerr != nil {
        return nil, nil, cerr
    }

    url, uerr := url.Parse("http://" + address)
    if uerr != nil {
        return nil, nil, uerr
    }

    return conn, url, nil
}

// Dial connects to an XML-RPC server at the specified network address.
func Dial(host string, port int, codec RPCCodec) (*rpc.Client, os.Error) {
    conn, url, cerr := OpenConnURL(host, port)
    if cerr != nil {
        return nil, &Error{Msg:cerr.String()}
    }

    return NewRPCClient(codec, conn, url), nil
}
