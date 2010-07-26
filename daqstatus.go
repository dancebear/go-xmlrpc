package main

import (
	"fmt"
	"os"
    "xmlrpc"
)

func main() {
    body, berr := xmlrpc.Marshal("rpc_ping")
    if berr != nil {
        fmt.Fprintf(os.Stderr, "Marshal failed: %v\n", berr)
    }

    r, err := xmlrpc.PostString("http://localhost:8080", "text/xml", body)
    if err != nil {
        fmt.Fprintf(os.Stderr, "PostString failed: %v\n", err)
        os.Exit(1)
    } else if r == nil {
        fmt.Fprintf(os.Stderr, "PostString returned nil response\n")
        os.Exit(1)
    }

    //io.Copy(os.Stdout, r.Body)
    _, pval, perr := xmlrpc.Unmarshal(r.Body)
    if perr != nil {
        fmt.Fprintf(os.Stderr, "%v\n", perr)
        os.Exit(1)
    }

    if r.Close {
        r.Body.Close()
    }

    fmt.Printf("rpc_ping returned %v <%T>\n", pval, pval)
}
