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
    var methodName string
    var pval interface{}
    var perr *xmlrpc.XMLRPCError

    methodName = "rpc_ping"
    pval, perr = rpccall(methodName)
    if perr != nil {
        fmt.Printf("%s failed: %v\n", methodName, perr)
    } else {
        fmt.Printf("%s returned %v <%T>\n", methodName, pval, pval)
    }
}
