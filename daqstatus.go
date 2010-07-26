package main

import (
	"fmt"
    "xmlrpc"
)

func rpccall(methodName string, args ... interface{}) (interface{},
    *xmlrpc.XMLRPCError, *xmlrpc.XMLRPCFault) {
    body, berr := xmlrpc.Marshal(methodName, args)
    if berr != nil {
        return nil, berr, nil
    }

    r, err := xmlrpc.PostString("http://localhost:8080", "text/xml", body)
    if err != nil {
        return nil, &xmlrpc.XMLRPCError{Msg:err.String()}, nil
    } else if r == nil {
        err := fmt.Sprintf("PostString for %s returned nil response\n",
            methodName)
        return nil, &xmlrpc.XMLRPCError{Msg:err}, nil
    }

    _, pval, perr, pfault := xmlrpc.Unmarshal(r.Body)

    if r.Close {
        r.Body.Close()
    }

    return pval, perr, pfault
}

func main() {
    var methodName string
    var pval interface{}
    var perr *xmlrpc.XMLRPCError
    var pfault *xmlrpc.XMLRPCFault

    methodName = "rpc_ping"
    pval, perr, pfault = rpccall(methodName)
    if perr != nil {
        fmt.Printf("%s failed: %v\n", methodName, perr)
    } else if pfault != nil {
        fmt.Printf("%s faulted: %v\n", methodName, pfault)
    } else {
        fmt.Printf("%s returned %v <%T>\n", methodName, pval, pval)
    }

    methodName = "rpc_runset_events"
    pval, perr, pfault = rpccall(methodName, 123, 4)
    if perr != nil {
        fmt.Printf("%s failed: %v\n", methodName, perr)
    } else if pfault != nil {
        fmt.Printf("%s faulted: %v\n", methodName, pfault)
    } else {
        fmt.Printf("%s returned %v <%T>\n", methodName, pval, pval)
    }
}
