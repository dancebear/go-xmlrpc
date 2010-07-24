package xmlrpc

import (
    "fmt"
    "io"
    "os"
    "strconv"
    "strings"
    "xml"
)

func isSpace(c byte) bool {
        return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

type ParseState int

var stateName = []string { "Method", "Params", "Param", "Value",
    "EndValue", "EndParam", "EndParams", "EndMethod" }

func (ps ParseState) String() string { return stateName[ps] }

const (
    psMethod ParseState = iota
    psParms
    psParam
    psValue
    psEndValue
    psEndParam
    psEndParms
    psEndMethod
)

func getStateVals(st ParseState, isResp bool) (string, bool) {
    isEnd := (st == psEndMethod || st == psEndParms || st == psEndParam ||
        st == psEndValue)

    var key string
    switch st {
    case psMethod, psEndMethod:
        if isResp {
            key = "methodResponse"
        } else {
            key = "methodRequest"
        }
    case psParms, psEndParms:
        key = "params"
    case psParam, psEndParam:
        key = "param"
    case psValue, psEndValue:
        key = "value"
    default:
        key = "???"
    }

    return key, isEnd
}

func getValue(typeName string, b []byte) (interface{}, string) {
    valStr := string(b)
    if typeName == "boolean" {
        if valStr == "1" {
            return true, ""
        } else if valStr == "0" {
            return false, ""
        } else {
            return nil, fmt.Sprintf("Bad <boolean> value \"%s\"", valStr)
        }
    } else if typeName == "double" {
        f, err := strconv.Atof(valStr)
        if err != nil {
            return f, err.String()
        }

        return f, ""
    } else if typeName == "int" || typeName == "i4" {
        i, err := strconv.Atoi(valStr)
        if err != nil {
            return i, err.String()
        }

        return i, ""
    } else if typeName == "string" {
        return valStr, ""
    }

    return nil, fmt.Sprintf("Unknown type <%s> for \"%s\"", typeName, valStr)
}

func parseValue(p *xml.Parser) (interface{}, string) {
    var typeName string
    var rtnVal interface{}

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            fmt.Fprintf(os.Stderr, "Token returned %v", err)
            os.Exit(1)
        }

        switch v := tok.(type) {
        case xml.StartElement:
            if typeName != "" {
                return nil, fmt.Sprintf("Found multiple types (%s and %s)" +
                    " inside <value>", typeName, v.Name.Local)
            }

            typeName = v.Name.Local
        case xml.EndElement:
            if typeName != v.Name.Local {
                return nil, fmt.Sprintf("Found unexpected </%s>", v.Name.Local)
            }
            return rtnVal, ""
        case xml.CharData:
            if typeName != "" && rtnVal == nil {
                var valErr string
                rtnVal, valErr = getValue(typeName, v)
                if valErr != "" {
                    return rtnVal, valErr
                }
            } else {
                for _, c := range v {
                    if !isSpace(c) {
                        if rtnVal == nil {
                            var valErr string
                            rtnVal, valErr = getValue("string", v)
                            return rtnVal, valErr
                        }

                        return nil, "Found non-whitespace chars inside <value>"
                    }
                }
            }
        default:
            return nil, fmt.Sprintf("Not handling <value> %v (type %T)", v, v)
        }
    }

    if typeName == "" {
        return rtnVal, "No type found inside <value>"
    }

    return rtnVal, fmt.Sprintf("Closing tag not found for <%s>", typeName)
}

func Parse(r io.Reader, isResp bool) (interface{}, string) {
    p := xml.NewParser(r)

    state := psMethod
    stateKey, wantEnd := getStateVals(state, isResp)

    var rtnVal interface{}

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            fmt.Fprintf(os.Stderr, "Token returned %v", err)
            os.Exit(1)
        }

        //fmt.Printf("ps %s key %s wantEnd %v tok %v<%T>\n", state, stateKey,
        //    wantEnd, tok, tok)

        switch v := tok.(type) {
        case xml.StartElement:
            if v.Name.Local == stateKey && ! wantEnd {
                if state != psValue {
                    state += 1
                    stateKey, wantEnd = getStateVals(state, isResp)
                } else {
                    var rtnErr string
                    rtnVal, rtnErr = parseValue(p)
                    if rtnErr != "" {
                        return nil, rtnErr
                    }
                    state += 1
                    stateKey, wantEnd = getStateVals(state, isResp)
                }
            } else {
                return nil, fmt.Sprintf("Unexpected <%s> token", v.Name.Local)
            }
        case xml.EndElement:
            if v.Name.Local == stateKey && wantEnd {
                state += 1
                stateKey, wantEnd = getStateVals(state, isResp)
            } else {
                return nil, fmt.Sprintf("Unexpected </%s> token", v.Name.Local)
            }
        case xml.CharData:
            for _, c := range v {
                if !isSpace(c) {
                    return nil, "Found non-whitespace chars"
                }
            }
        case xml.ProcInst:
            // ignored
        default:
            return nil, fmt.Sprintf("Not handling %v (type %T)", v, v)
        }
    }

    return rtnVal, ""
}

func ParseString(s string, isResp bool) (interface{}, string) {
    return Parse(strings.NewReader(s), isResp)
}
 