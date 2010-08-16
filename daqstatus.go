package main

import (
    "fmt"
    "net"
    "os"
    "rpc"
    "rpc/jsonrpc"
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

func runRPCXMLClient(port int) {
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

func dialJSON(host string, port int) (*rpc.Client, os.Error) {
    address := fmt.Sprintf("%s:%d", host, port)

    conn, err := net.Dial("tcp", "", address)
    if err != nil {
        return nil, err
    }

    return rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn)), nil
}

func runRPCJSONClient(port int) {
    client, cerr := dialJSON("localhost", port)
    if cerr != nil {
        fmt.Printf("Create failed: %v\n", cerr)
        return
    }

    var rep int
    client.Call("RPCFunc.Prettyprint",
        &Args{[]string{
            "Zero", "One",
        },
        []int{1,0,1,1,0,1,1,1,0,0,0}}, &rep)
    fmt.Printf("Len: %d\n", rep)
}

type ServerObject struct {
}

func (*ServerObject) rpc_ping() int {
    return 12345
}

func (*ServerObject) rpc_runset_events(rsid int, subrunNum int) (int, bool) {
    return 17, false
}

type RPCFunc uint8 

type Args struct {
     Resolver []string
     Nums     []int
}

func (*RPCFunc) Prettyprint(args *Args, result *int) os.Error {
    *result = 0
    for _,k := range args.Nums {
        fmt.Printf("%d => \"%s\"\n", k, args.Resolver[k])
        *result++
    }
    return nil
}

func runJSONServer(l net.Listener) {
    for {
        conn, _ := l.Accept()
        rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
    }
}

func startJSONServer(port int) net.Listener {
    rpc.Register(new (RPCFunc))
    l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        fmt.Fprintf(os.Stderr, "Could not open port: %s\n", err)
        os.Exit(1)
    }
    go runJSONServer(l)
    return l
}

func main() {
    var sobj *ServerObject
    var l net.Listener
    var r *xmlrpc.RPCHandler

    fmt.Printf("\n============================\n")
    fmt.Printf("--- Old XML client, Python server\n\n")
    runRPCXMLClient(8080)

    fmt.Printf("\n============================\n")
    fmt.Printf("--- Old XML client, Go server\n\n")
    sobj = new(ServerObject)
    l, r = xmlrpc.StartServer(8081)
    if l != nil && r != nil {
        r.Register("", sobj)
        runRPCXMLClient(8081)
        l.Close()
    }

    fmt.Printf("\n============================\n")
    fmt.Printf("--- New XML client, Python server\n\n")
    runMyXMLClient(8080)

    fmt.Printf("\n============================\n\n")
    fmt.Printf("--- New XML client, Go server\n")
    sobj = new(ServerObject)
    l, r = xmlrpc.StartServer(8082)
    if l != nil && r != nil {
        r.Register("", sobj)
        runMyXMLClient(8082)
        l.Close()
    }

    fmt.Printf("\n============================\n")
    fmt.Printf("--- Old JSON client, old JSON server\n\n")
    l = startJSONServer(8083)
    runRPCJSONClient(8083)

    l.Close()
}
