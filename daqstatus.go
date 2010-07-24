package main

import (
	"fmt"
	"os"
    "xmlrpc"
)

func main() {
    body := "<?xml version=\"1.0\"?>\n<methodCall>\n<methodName>rpc_ping</methodName>\n<params>\n</params>\n</methodCall>\n"
    //t := "<?xml version='1.0'?>\n<methodCall>\n<methodName>rpc_ping</methodName>\n<params>\n</params>\n</methodCall>\n"
    r, err := xmlrpc.PostString("http://localhost:8080", "text/xml", body)
    if err != nil {
        fmt.Fprintf(os.Stderr, "PostString failed: %v", err)
        os.Exit(1)
    } else if r == nil {
        fmt.Fprintf(os.Stderr, "PostString returned nil response")
        os.Exit(1)
    }

    //io.Copy(os.Stdout, r.Body)
    pval, perr := xmlrpc.Parse(r.Body, true)
    if len(perr) != 0 {
        fmt.Fprintf(os.Stderr, "%s\n", perr)
        os.Exit(1)
    }

    fmt.Printf("XML-RPC returned %v <%T>\n", pval, pval)

    if r.Close {
        fmt.Printf("Closing ...")
        r.Body.Close()
        fmt.Printf(" done\n")
    }
}
