package main

import (
    "fmt"
    "rpc"
    "xmlrpc"
)

func call(client *rpc.Client, name string, params []interface{}) interface{} {
    var reply interface{}

    defer func() {
        if err := recover(); err != nil {
            fmt.Printf("Caught fault: %v\n", err)
        }
    }()

    cerr := client.Call(name, params, &reply)
    if cerr != nil {
        fmt.Printf("%s failed: %v\n", name, cerr)
    } else {
        fmt.Printf("%s returned %v <%T>\n", name, reply, reply)
    }

    return reply
}

func runClient(port int) {
    var name string
    var params []interface{}

    client, cerr := xmlrpc.Dial("localhost", port)
    if cerr != nil {
        fmt.Printf("Create failed: %v\n", cerr)
        return
    }

    for i := 0; i < 2; i++ {
        switch i {
        case 0:
            name = "rpc_ping"
            params = make([]interface{}, 0, 0)
        case 1:
            name = "rpc_runset_events"
            params = make([]interface{}, 2, 2)
            params[0] = 123
            params[1] = 4
        }

        call(client, name, params)
    }
}

func main() {
    runClient(8080)
}
