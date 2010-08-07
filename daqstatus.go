package main

import (
    "fmt"
    "xmlrpc"
)

func runClient(port int) {
    var methodName string
    var params []interface{}
    var reply interface{}
    var perr *xmlrpc.XMLRPCError
    var pfault *xmlrpc.XMLRPCFault

    client, cerr := xmlrpc.NewClient(fmt.Sprintf("http://localhost:%d", port))
    if cerr != nil {
        fmt.Printf("NewClient failed: %v\n", cerr)
        return
    }

    for i := 0; i < 2; i++ {
        switch i {
        case 0:
            methodName = "rpc_ping"
            params = nil
        case 1:
            methodName = "rpc_runset_events"
            params = make([]interface{}, 2, 2)
            params[0] = 123
            params[1] = 4
        }

        reply, pfault, perr = client.RPCCall(methodName, params)
        if perr != nil {
            fmt.Printf("%s failed: %v\n", methodName, perr)
        } else if pfault != nil {
            fmt.Printf("%s faulted: %v\n", methodName, pfault)
        } else {
            fmt.Printf("%s returned %v <%T>\n", methodName, reply, reply)
        }
    }
}

func main() {
    runClient(8080)
}
