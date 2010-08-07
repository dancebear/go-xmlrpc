package main

import (
    "fmt"
    "os"
    "rpc"
    "xmlrpc"
)

func runClient(port int) {
    var name string
    var params []interface{}
    var reply interface{}
    var cerr os.Error

    var newClient *rpc.Client
    newClient, cerr = xmlrpc.Dial("localhost", port)
    if cerr != nil {
        fmt.Printf("Create failed: %v\n", cerr)
        return
    }

    for i := 0; i < 2; i++ {
        fmt.Printf("+++++++++ Case#%d\n", i)
        switch i {
        case 0:
            name = "rpc_ping"
            params = make([]interface{}, 0, 0)
            cerr = newClient.Call(name, params, &reply)
        case 1:
            name = "rpc_runset_events"
            params = make([]interface{}, 2, 2)
            params[0] = 123
            params[1] = 4
            cerr = newClient.Call(name, params, &reply)
        }

        if cerr != nil {
            fmt.Printf("%s failed: %v\n", name, cerr)
        } else {
            fmt.Printf("%s returned %v <%T>\n", name, reply, reply)
        }
    }
}

func main() {
    runClient(8080)
}
