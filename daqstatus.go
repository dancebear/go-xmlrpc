package main

import (
    "fmt"
    "net"
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

func runRPCClient(port int, useXML bool) {
    var name string
    var params []interface{}

    client, cerr := xmlrpc.Dial("localhost", port, useXML)
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

func runMyXMLClient(port int) interface{} {
    client, cerr := xmlrpc.NewClient("localhost", port)
    if cerr != nil {
        fmt.Printf("Create failed: %v\n", cerr)
        return nil
    }

    var reply interface{}
    for i := 0; i < 2; i++ {
        var name string
        var fault *xmlrpc.Fault
        var err *xmlrpc.Error
        switch i {
        case 0:
            name = "rpc_ping"
            reply, fault, err = client.RPCCall(name)
        case 1:
            name = "rpc_runset_events"
            reply, fault, err = client.RPCCall(name, 123, 4)
        }

        if err != nil {
            fmt.Printf("%s failed: %v\n", name, err)
        } else if fault != nil {
            fmt.Printf("%s faulted: %v\n", name, fault)
        } else {
            fmt.Printf("%s returned %v\n", name, reply)
        }
    }

    return reply
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
    var sobj *ServerObject
    var l net.Listener

    sobj = new(ServerObject)

    handler := xmlrpc.NewHandler()
    handler.Register("", sobj)

    fmt.Printf("\n============================\n")
    fmt.Printf("--- Old XML client, Python server\n\n")
    runRPCClient(8080, true)

    fmt.Printf("\n============================\n")
    fmt.Printf("--- Old XML client, Go server\n\n")
    l = xmlrpc.StartServer(8081, handler)
    if l != nil {
        runRPCClient(8081, true)
        l.Close()
    }

    fmt.Printf("\n============================\n")
    fmt.Printf("--- New XML client, Python server\n\n")
    runMyXMLClient(8080)

    fmt.Printf("\n============================\n\n")
    fmt.Printf("--- New XML client, Go server\n")
    sobj = new(ServerObject)
    l = xmlrpc.StartServer(8082, handler)
    if l != nil{
        runMyXMLClient(8082)
        l.Close()
    }

    fmt.Printf("\n============================\n")
    fmt.Printf("--- Old JSON client, Python server\n\n")
    runRPCClient(8090, false)
}
