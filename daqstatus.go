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
        fmt.Printf("%s returned %v\n", name, reply)
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

type ServerObject struct {
}

func (*ServerObject) rpc_ping() int {
    return 12345
}

func (*ServerObject) rpc_runset_events(rsid int, subrunNum int) (int, bool) {
    return 17, false
}

func main() {
    runClient(8080)

    fmt.Printf("\n============================\n\n")
    sobj := new(ServerObject)
    l, r := xmlrpc.StartServerPlus(8082)
    r.Register("", sobj)
    runClient(8082)
    l.Close()
}
