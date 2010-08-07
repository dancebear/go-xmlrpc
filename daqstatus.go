package main

import (
    "fmt"
    "xmlrpc"
)

func runClient(port int) {
    var methodName string
    var pval interface{}
    var perr *xmlrpc.XMLRPCError
    var pfault *xmlrpc.XMLRPCFault

    client, cerr := xmlrpc.NewClient(fmt.Sprintf("http://localhost:%d", port))
    if cerr != nil {
        fmt.Printf("NewClient failed: %v\n", cerr)
        return
    }

    
    methodName = "rpc_ping"
    pval, pfault, perr = client.RPCCall(methodName)
    if perr != nil {
        fmt.Printf("%s failed: %v\n", methodName, perr)
    } else if pfault != nil {
        fmt.Printf("%s faulted: %v\n", methodName, pfault)
    } else {
        fmt.Printf("%s returned %v <%T>\n", methodName, pval, pval)
    }

    methodName = "rpc_runset_events"
    pval, pfault, perr = client.RPCCall(methodName, 123, 4)
    if perr != nil {
        fmt.Printf("%s failed: %v\n", methodName, perr)
    } else if pfault != nil {
        fmt.Printf("%s faulted: %v\n", methodName, pfault)
    } else {
        fmt.Printf("%s returned %v <%T>\n", methodName, pval, pval)
    }
}

func main() {
    runClient(8080)
}
