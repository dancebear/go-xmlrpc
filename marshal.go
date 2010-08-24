package xmlrpc;

import (
    "fmt"
    "io"
)

func wrapParam(xval interface{}) (string, *Error) {
    var valStr string

    if xval == nil {
        valStr = "<nil/>"
    } else {
        switch val := xval.(type) {
        case bool:
            var bval int
            if val {
                bval = 1
            } else {
                bval = 0
            }
            valStr = fmt.Sprintf("<boolean>%d</boolean>", bval)
        case float:
            valStr = fmt.Sprintf("<double>%f</double>", val)
        case int:
            valStr = fmt.Sprintf("<int>%d</int>", val)
        case string:
            valStr = fmt.Sprintf("<string>%s</string>", val)
        default:
            err := fmt.Sprintf("Not wrapping type %T (%v)", val, val)
            return "", &Error{Msg:err}
        }
    }

    return fmt.Sprintf(`    <param>
      <value>
        %s
      </value>
    </param>
`, valStr), nil
}

// Write a local data object as an XML-RPC request
func Marshal(w io.Writer, methodName string, args ... interface{}) *Error {
    return marshalArray(w, methodName, args)
}

func marshalArray(w io.Writer, methodName string, args []interface{}) *Error {
    var name string
    var addExtra bool
    if methodName == "" {
        name = "Response"
        addExtra = false
    } else {
        name = "Call"
        addExtra = true
    }

    fmt.Fprintf(w, "<?xml version=\"1.0\"?>\n<method%s>\n", name)
    if addExtra {
        fmt.Fprintf(w, "  <methodName>%s</methodName>\n", methodName)
    }

    fmt.Fprintf(w, "  <params>\n")

    for _, a := range args {
        valStr, err := wrapParam(a)
        if err != nil {
            return err
        }

        fmt.Fprintf(w, valStr)
    }

    fmt.Fprintf(w, "  </params>\n</method%s>\n", name)

    return nil
}
