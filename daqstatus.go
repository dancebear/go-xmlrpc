package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
    "http"
	"io"
	"net"
	"os"
	"strings"
    "xml"
)

/********** From request.go ************/

type badStringError struct {
	what string
	str  string
}

func (e *badStringError) String() string { return fmt.Sprintf("%s %q", e.what, e.str) }

/********** From client.go ************/

// Given a string of the form "host", "host:port", or "[ipv6::address]:port",
// return true if the string includes a port.
func hasPort(s string) bool { return strings.LastIndex(s, ":") > strings.LastIndex(s, "]") }

// Used in Send to implement io.ReadCloser by bundling together the
// io.BufReader through which we read the response, and the underlying
// network connection.
type readClose struct {
	io.Reader
	io.Closer
}

// Send issues an HTTP request.  Caller should close resp.Body when done reading it.
//
// TODO: support persistent connections (multiple requests on a single connection).
// send() method is nonpublic because, when we refactor the code for persistent
// connections, it may no longer make sense to have a method with this signature.
func send(req *http.Request) (resp *http.Response, err os.Error) {
	if req.URL.Scheme != "http" {
		return nil, &badStringError{"unsupported protocol scheme", req.URL.Scheme}
	}

	addr := req.URL.Host
	if !hasPort(addr) {
		addr += ":http"
	}
	info := req.URL.Userinfo
	if len(info) > 0 {
		enc := base64.URLEncoding
		encoded := make([]byte, enc.EncodedLen(len(info)))
		enc.Encode(encoded, []byte(info))
		if req.Header == nil {
			req.Header = make(map[string]string)
		}
		req.Header["Authorization"] = "Basic " + string(encoded)
	}
	conn, err := net.Dial("tcp", "", addr)
	if err != nil {
		return nil, err
	}

	err = req.Write(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	resp, err = http.ReadResponse(reader, req.Method)
	if err != nil {
		conn.Close()
		return nil, err
	}

	resp.Body = readClose{resp.Body, conn}

	return
}

// Post issues a POST to the specified URL.
//
// Caller should close r.Body when done reading it.
// XXX - copied from http.Post, but 'body' is a string instead of io.Reader,
// XXX   req.TransferEncoding is not set and re.ContentLength is set to len(body)
func PostString(url string, bodyType string, body string) (r *http.Response, err os.Error) {
	var req http.Request
	req.Method = "POST"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Close = true
	req.Body = nopCloser{strings.NewReader(body)}
	req.Header = map[string]string{
		"Content-Type": bodyType,
	}

    req.RawURL = "/RPC2"
    req.ContentLength = int64(len(body))

	req.URL, err = http.ParseURL(url)
	if err != nil {
		return nil, err
	}

	return send(&req)
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() os.Error { return nil }

/************** My code *****************/

/*
import (
    "fmt"
    "strings"
)
*/

const tokenMethodResponse = "methodResponse"
const tokenParams = "params"
const tokenParam = "param"

func parseValue(p *xml.Parser) (interface{}, string) {
    var inVal bool
    var typeName string
    var rtnVal interface{}

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            fmt.Fprintf(os.Stderr, "Token returned %v", err)
            os.Exit(1)
        }

        switch v := tok.(type) {
        case xml.EndElement:
            if v.Name.Local == "param" {
                return rtnVal, ""
            } else if v.Name.Local == "value" {
                if ! inVal {
                    return nil, fmt.Sprintf("Multiple </%s> tokens",
                        v.Name.Local)
                }
                inVal = false
            } else if typeName == "" {
                return nil, fmt.Sprintf("Type was not found", v.Name.Local)
            } else if v.Name.Local == typeName {
                typeName = ""
            }
        case xml.StartElement:
            if v.Name.Local == "value" {
                if inVal {
                    return nil, fmt.Sprintf("Multiple <%s> tokens",
                        v.Name.Local)
                }
                inVal = true
            } else if typeName != "" {
                return nil, fmt.Sprintf("Found multiple types (<%s> and <%s>",
                    v.Name.Local, typeName)
            } else {
                typeName = v.Name.Local
            }
        case xml.CharData:
            if len(v) == 1 && (v[0] =='\n' || v[0] == '\r') {
                // ignore
            } else {
                fmt.Printf("====> Need to grab %v\n", v)
            }
        default:
            return nil, fmt.Sprintf("Not handling %v <%T>\n", v, v)
        }
    }

    return rtnVal, ""
}

func parse(r io.Reader) (interface{}, string) {
    p := xml.NewParser(r)

    var rtnVal interface{}

    var inResp, inParams, inParam bool
    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            fmt.Fprintf(os.Stderr, "Token returned %v", err)
            os.Exit(1)
        }

        switch v := tok.(type) {
        case xml.EndElement:
            if v.Name.Local == tokenMethodResponse {
                if ! inResp {
                    return nil, fmt.Sprintf("Multiple </%s> tokens", v.Name.Local)
                }
                inResp = false
            } else if v.Name.Local == tokenParams {
                if ! inParams {
                    return nil, fmt.Sprintf("Multiple </%s> tokens", v.Name.Local)
                }
                inParams = false
            } else if v.Name.Local == tokenParam {
                if ! inParam {
                    return nil, fmt.Sprintf("Multiple </%s> tokens", v.Name.Local)
                }
                inParam = false
            } else {
                return nil, fmt.Sprintf("Unknown </%s> token", v.Name.Local)
            }
        case xml.StartElement:
            if v.Name.Local == tokenMethodResponse {
                if inResp {
                    return nil, fmt.Sprintf("Multiple <%s> tokens", v.Name.Local)
                }
                inResp = true
            } else if v.Name.Local == tokenParams {
                if ! inResp {
                    return nil, fmt.Sprintf("Out-of-order <%s> token", v.Name.Local)
                } else if inParams {
                    return nil, fmt.Sprintf("Multiple <%s> tokens", v.Name.Local)
                }
                inParams = true
            } else if v.Name.Local == tokenParam {
                if ! inResp || ! inParams {
                    return nil, fmt.Sprintf("Out-of-order <%s> token", v.Name.Local)
                } else if inParam {
                    return nil, fmt.Sprintf("Multiple <%s> tokens", v.Name.Local)
                }
                inParam = true
                var rtnErr string
                rtnVal, rtnErr = parseValue(p)
                if rtnErr != "" {
                    return nil, rtnErr
                }
            } else {
                return nil, fmt.Sprintf("Unknown <%s> token", v.Name.Local)
            }
        case xml.CharData:
            if len(v) == 1 && (v[0] =='\n' || v[0] == '\r') {
                // ignore
            } else {
                fmt.Printf("??? %v<%T>\n", v, v)
            }
        }
    }

    return rtnVal, ""
}

func main() {
    body := "<?xml version=\"1.0\"?>\n<methodCall>\n<methodName>rpc_ping</methodName>\n<params>\n</params>\n</methodCall>\n"
    //t := "<?xml version='1.0'?>\n<methodCall>\n<methodName>rpc_ping</methodName>\n<params>\n</params>\n</methodCall>\n"
    r, err := PostString("http://localhost:8080", "text/xml", body)
    if err != nil {
        fmt.Fprintf(os.Stderr, "PostString failed: %v", err)
        os.Exit(1)
    } else if r == nil {
        fmt.Fprintf(os.Stderr, "PostString returned nil response")
        os.Exit(1)
    }

    //io.Copy(os.Stdout, r.Body)
    pval, perr := parse(r.Body)
    if len(perr) != 0 {
        fmt.Fprintf(os.Stderr, "%s\n", perr)
        os.Exit(1)
    }

    fmt.Printf("XML-RPC returned %v\n", pval)

    if r.Close {
        fmt.Printf("Closing ...")
        r.Body.Close()
        fmt.Printf(" done\n")
    }
}
