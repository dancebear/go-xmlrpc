package main

import (
    "fmt"
    "xmlrpc"
)

func runClient(port int) {
    var name string
    var reply interface{}
    var perr *xmlrpc.Error
    var pfault *xmlrpc.Fault

    client, cerr := xmlrpc.NewClient("localhost", port)
    if cerr != nil {
        fmt.Printf("NewClient failed: %v\n", cerr)
        return
    }

    for i := 0; i < 2; i++ {
        switch i {
        case 0:
            name = "rpc_ping"
            reply, pfault, perr = client.RPCCall(name)
        case 1:
            name = "rpc_runset_events"
            reply, pfault, perr = client.RPCCall(name, 123, 4)
        }

        if perr != nil {
            fmt.Printf("%s failed: %v\n", name, perr)
        } else if pfault != nil {
            fmt.Printf("%s faulted: %v\n", name, pfault)
        } else {
            fmt.Printf("%s returned %v <%T>\n", name, reply, reply)
        }
    }
}

func main() {
    runClient(8080)
}
