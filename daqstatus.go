package main

import (
	"fmt"
    "xmlrpc"
)

func rpccall(methodName string, args ... interface{}) (interface{},
    *xmlrpc.XMLRPCError) {
    body, berr := xmlrpc.Marshal(methodName, args)
    if berr != nil {
        return nil, berr
    }

    r, err := xmlrpc.PostString("http://localhost:8080", "text/xml", body)
    if err != nil {
        return nil, &xmlrpc.XMLRPCError{Msg:err.String()}
    } else if r == nil {
        err := fmt.Sprintf("PostString for %s returned nil response\n",
            methodName)
        return nil, &xmlrpc.XMLRPCError{Msg:err}
    }

    _, pval, perr := xmlrpc.Unmarshal(r.Body)

    if r.Close {
        r.Body.Close()
    }

    return pval, perr
}

func main() {
    var pval interface{}
    var perr *xmlrpc.XMLRPCError

    pval, perr = rpccall("rpc_ping")
    if perr != nil {
        fmt.Printf("rpc_ping failed: %v\n", perr)
    } else {
        fmt.Printf("rpc_ping returned %v <%T>\n", pval, pval)
    }
}
