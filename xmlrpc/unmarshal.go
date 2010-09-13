package xmlrpc

import (
    "container/vector"
    "fmt"
    "io"
    "rpc2"
    "strconv"
    "strings"
    "xml"
)

func isSpace(c byte) bool {
        return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

type parseState int

var psStrings = []string { "Method", "MethodName", "InName", "Params",
    "Param", "Value", "EndValue", "EndParam", "EndParams", "EndMethod", "???", }

func (ps parseState) String() string { return psStrings[ps] }

const (
    psMethod parseState = iota
    psName
    psInName
    psParms
    psParam
    psValue
    psEndValue
    psEndParam
    psEndParms
    psEndMethod
)

type structState int

var ssStrings = []string { "Initial", "Member", "Name", "InName", "EndName",
    "Value", "InValue", "EndValue", "EndMember", "EndStruct", "???", }

func (ss structState) String() string { return ssStrings[ss] }

const (
    stInitial = iota
    stMember
    stName
    stInName
    stEndName
    stValue
    stInValue
    stEndValue
    stEndMember
    stEndStruct
)

func getStateVals(st parseState, isResp bool) (string, bool) {
    isEnd := (st == psEndMethod || st == psEndParms || st == psEndParam ||
        st == psEndValue)

    var tag string
    switch st {
    case psMethod, psEndMethod:
        if isResp {
            tag = "methodResponse"
        } else {
            tag = "methodCall"
        }
    case psName, psInName:
        tag = "methodName"
    case psParms, psEndParms:
        tag = "params"
    case psParam, psEndParam:
        tag = "param"
    case psValue, psEndValue:
        tag = "value"
    default:
        tag = "???"
    }

    return tag, isEnd
}

func getNextStructState(state structState) (structState, string, bool) {
    state += 1
    var stateTag string
    isEnd := state == stEndName || state == stEndValue ||
        state == stEndMember || state == stEndStruct

    switch state {
    case stMember, stEndMember:
        stateTag = "member"
    case stName, stEndName:
        stateTag = "name"
    case stValue, stEndValue:
        stateTag = "value"
    case stEndStruct:
        stateTag = "struct"
    default:
        stateTag = ""
    }

    return state, stateTag, isEnd
}

func parseStruct(p *xml.Parser) (interface{}, *rpc2.Error, bool) {
    state, stateTag, wantEnd := getNextStructState(stInitial)

    key := ""

    valMap := make(map[string]interface{})

    finished := false
    for ! finished {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return nil, &rpc2.Error{Msg:err.String()}, false
        }

        const debug = false
        if debug {
            var tokStr string
            if t2, ok := tok.(xml.CharData); ok {
                tokStr = string([]byte(t2))
            } else {
                tokStr = fmt.Sprintf("%v", tok)
            }

            fmt.Printf("st %v tag %s wantEnd %v tok %s<%T>\n", state, stateTag,
                wantEnd, tokStr, tok)
        }

        switch v := tok.(type) {
        case xml.StartElement:
            if wantEnd && state == stEndStruct && v.Name.Local == "member" {
                state = stMember
            } else if wantEnd || v.Name.Local != stateTag {
                err := fmt.Sprintf("Expected struct tag <%s>, not <%s>",
                    stateTag, v.Name.Local)
                return nil, &rpc2.Error{Msg:err}, false
            }

            if state == stValue {
                val, err, sawEndValTag := unmarshalValue(p)
                if err != nil {
                    return nil, err, false
                }

                valMap[key] = val

                if sawEndValTag {
                    state = stEndValue
                } else {
                    state = stEndValue - 1
                }
            }

            state, stateTag, wantEnd = getNextStructState(state)

        case xml.EndElement:
            if wantEnd && state == stEndStruct && v.Name.Local == "member" {
                state = stMember
            } else if ! wantEnd || v.Name.Local != stateTag {
                err := fmt.Sprintf("Expected struct tag </%s>, not </%s>",
                    stateTag, v.Name.Local)
                return nil, &rpc2.Error{Msg:err}, false
            }

            if state == stEndStruct {
                finished = true
            } else {
                state, stateTag, wantEnd = getNextStructState(state)
            }
        case xml.CharData:
            if state == stInName {
                key = string([]byte(v))
                state, stateTag, wantEnd = getNextStructState(state)
            } else {
                ignore := true
                for _, c := range v {
                    if !isSpace(c) {
                        ignore = false
                        break
                    }
                }

                if ! ignore {
                    err := &rpc2.Error{Msg:fmt.Sprintf("Found" +
                            " non-whitespace chars \"%s\" inside <struct>",
                            string([]byte(v)))}
                    return nil, err, false
                }
            }
        }
    }

    return valMap, nil, true
}

func getValue(p *xml.Parser, typeName string, b []byte) (interface{},
    *rpc2.Error, bool) {
    var unimplemented = &rpc2.Error{Msg:"Unimplemented"}

    valStr := string(b)
    if typeName == "array" {
        return nil, unimplemented, false
    } else if typeName == "base64" {
        return nil, unimplemented, false
    } else if typeName == "boolean" {
        if valStr == "1" {
            return true, nil, false
        } else if valStr == "0" {
            return false, nil, false
        } else {
            msg := fmt.Sprintf("Bad <boolean> value \"%s\"", valStr)
            return nil, &rpc2.Error{Msg:msg}, false
        }
    } else if typeName == "dateTime.iso8601" {
        return nil, unimplemented, false
    } else if typeName == "double" {
        f, err := strconv.Atof(valStr)
        if err != nil {
            return f, &rpc2.Error{Msg:err.String()}, false
        }

        return f, nil, false
    } else if typeName == "int" || typeName == "i4" {
        i, err := strconv.Atoi(valStr)
        if err != nil {
            return i, &rpc2.Error{Msg:err.String()}, false
        }

        return i, nil, false
    } else if typeName == "string" {
        return valStr, nil, false
    } else if typeName == "struct" {
        return parseStruct(p)
    }

    return nil, &rpc2.Error{Msg:fmt.Sprintf("Unknown type <%s> for \"%s\"",
            typeName, valStr)}, false
}

func unmarshalValue(p *xml.Parser) (interface{}, *rpc2.Error, bool) {
    var typeName string
    var rtnVal interface{}

    const noEndValTag = false

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return rtnVal, &rpc2.Error{Msg:err.String()}, noEndValTag
        }

        const debug = false
        if debug {
            var tokStr string
            if t2, ok := tok.(xml.CharData); ok {
                tokStr = string([]byte(t2))
            } else {
                tokStr = fmt.Sprintf("%v", tok)
            }

            fmt.Printf("ty %s rtn %v tok %s<%T>\n", typeName, rtnVal, tokStr,
                tok)
        }

        switch v := tok.(type) {
        case xml.StartElement:
            if typeName != "" {
                err := &rpc2.Error{Msg:fmt.Sprintf("Found multiple types" +
                        " (%s and %s) inside <value>", typeName, v.Name.Local)}
                return nil, err, noEndValTag
            }

            typeName = v.Name.Local
        case xml.EndElement:
            if typeName == "" && v.Name.Local == "value" {
                return "", nil, true
            } else if typeName != v.Name.Local {
                err := &rpc2.Error{Msg:fmt.Sprintf("Found unexpected </%s>" +
                        " (wanted </%s>)", v.Name.Local, typeName)}
                return nil, err, noEndValTag
            }

            if typeName == "string" && rtnVal == nil {
                rtnVal = ""
            }
            return rtnVal, nil, noEndValTag
        case xml.CharData:
            if typeName != "" && rtnVal == nil {
                var valErr *rpc2.Error
                var sawEndTypeTag bool
                rtnVal, valErr, sawEndTypeTag = getValue(p, typeName, v)
                if valErr != nil {
                    return rtnVal, valErr, noEndValTag
                }

                if sawEndTypeTag {
                    return rtnVal, nil, noEndValTag
                }
            } else {
                for _, c := range v {
                    if !isSpace(c) {
                        if rtnVal == nil {
                            return string([]byte(v)), nil, noEndValTag
                        }

                        err := &rpc2.Error{Msg:fmt.Sprintf("Found" +
                                " non-whitespace chars \"%s\" inside <value>",
                                string([]byte(v)))}
                        return nil, err, noEndValTag
                    }
                }
            }
        default:
            err := &rpc2.Error{Msg:fmt.Sprintf("Not handling <value> %v" +
                    " (type %T)", v, v)}
            return nil, err, noEndValTag
        }
    }

    if typeName == "" {
        return rtnVal, &rpc2.Error{Msg:"No type found inside <value>"},
        noEndValTag
    }

    return rtnVal, &rpc2.Error{Msg:fmt.Sprintf("Closing tag not found for" +
        " <%s>", typeName)}, noEndValTag
}

func extractParams(v *vector.Vector) interface{} {
    if v.Len() == 0 {
        return nil
    } else if v.Len() == 1 {
        return v.At(0)
    }

    pList := make([]interface{}, v.Len(), v.Len())
    for i := 0; i < v.Len(); i++ {
        pList[i] = v.At(i)
    }

    return pList
}

// Translate an XML string into a local data object
func Unmarshal(r io.Reader) (string, interface{}, *rpc2.Error, *rpc2.Fault) {
    p := xml.NewParser(r)

    state := psMethod
    isResp := true
    stateTag := "???"
    wantEnd := false
    methodName := ""

    params := &vector.Vector{}

    isFault := false
    var faultVal *rpc2.Fault

    for {
        tok, err := p.Token()
        if tok == nil {
            break
        }

        if err != nil {
            return methodName, extractParams(params),
            &rpc2.Error{Msg:err.String()}, faultVal
        }

        const debug = false
        if debug {
            var tokStr string
            if t2, ok := tok.(xml.CharData); ok {
                tokStr = string([]byte(t2))
            } else {
                tokStr = fmt.Sprintf("%v", tok)
            }

            fmt.Printf("ps %s tag %s wantEnd %v tok %s<%T>\n", state, stateTag,
                wantEnd, tokStr, tok)
        }

        switch v := tok.(type) {
        case xml.StartElement:
            if state == psMethod {
                if v.Name.Local == "methodResponse" {
                    state = psParms
                    isResp = true
                    stateTag, wantEnd = getStateVals(state, isResp)
                } else if v.Name.Local == "methodCall" {
                    state = psName
                    isResp = false
                    stateTag, wantEnd = getStateVals(state, isResp)
                } else {
                    err := &rpc2.Error{Msg:fmt.Sprintf("Unexpected initial" +
                            " tag <%s>", v.Name.Local)}
                    return methodName, extractParams(params), err, faultVal
                }
            } else if v.Name.Local == stateTag && ! wantEnd {
                if state != psValue {
                    state += 1
                    stateTag, wantEnd = getStateVals(state, isResp)
                } else {
                    var uVal interface{}
                    var uErr *rpc2.Error
                    var sawEndValTag bool
                    uVal, uErr, sawEndValTag = unmarshalValue(p)
                    if uErr != nil {
                        return methodName, extractParams(params), uErr,
                        faultVal
                    }
                    if isFault {
                        if uVal == nil {
                            err := &rpc2.Error{Msg:"No fault value returned"}
                            return methodName, extractParams(params), err, nil
                        }

                        if fmap, ok := uVal.(map[string]interface{}); ! ok {
                            err := fmt.Sprintf("Bad type %T for fault", uVal)
                            return methodName, extractParams(params),
                            &rpc2.Error{Msg:err},
                            nil
                        } else {
                            if code, ok := fmap["faultCode"].(int); ! ok {
                                err := fmt.Sprintf("Fault code should be an" +
                                    " int, not %T", code)
                                return methodName, extractParams(params),
                                &rpc2.Error{Msg:err}, nil
                            } else if msg, ok := fmap["faultString"].(string);
                            ! ok {
                                err := fmt.Sprintf("Fault string should be a" +
                                    " string, not %T", msg)
                                return methodName, extractParams(params),
                                &rpc2.Error{Msg:err}, nil
                            } else {
                                faultVal = &rpc2.Fault{Code:code, Msg:msg}
                            }
                        }

                        if ! sawEndValTag {
                            state += 1
                            stateTag, wantEnd = getStateVals(state, isResp)
                        } else {
                            state = psEndParms
                            stateTag = "fault"
                            wantEnd = true
                        }
                    } else {
                        params.Push(uVal)
                        if ! sawEndValTag {
                            state += 1
                        } else {
                            state = psEndParam
                        }
                        stateTag, wantEnd = getStateVals(state, isResp)
                    }
                }
            } else if state == psParms && v.Name.Local == "fault" {
                isFault = true
                state = psValue
                stateTag, wantEnd = getStateVals(state, isResp)
            } else if wantEnd && state == psEndParms &&
                v.Name.Local == "param" {
                state = psValue
                stateTag, wantEnd = getStateVals(state, isResp)
            } else {
                err := &rpc2.Error{Msg:fmt.Sprintf("Unexpected <%s> token" +
                        " for state %s", v.Name.Local, state)}
                return methodName, extractParams(params), err, faultVal
            }
        case xml.EndElement:
            if state == psEndMethod {
                if methodName == "" {
                    stateTag = "methodResponse"
                } else {
                    stateTag = "methodCall"
                }
                if v.Name.Local == stateTag {
                    state += 1
                    stateTag = "???"
                    wantEnd = false
                }
            } else if v.Name.Local == stateTag && wantEnd {
                if isFault && state == psEndValue {
                    state = psEndParms
                    stateTag = "fault"
                    wantEnd = true
                } else {
                    state += 1
                    stateTag, wantEnd = getStateVals(state, isResp)
                }
            } else if state == psParam && ! wantEnd &&
                v.Name.Local == "params" {
                state = psEndMethod
                stateTag, wantEnd = getStateVals(state, isResp)
            } else {
                err := &rpc2.Error{Msg:fmt.Sprintf("Unexpected </%s> token" +
                        " for state %s", v.Name.Local, state)}
                return methodName, extractParams(params), err, faultVal
            }
        case xml.CharData:
            if state == psInName {
                methodName = string([]byte(v))
                wantEnd = true
            } else {
                for _, c := range v {
                    if !isSpace(c) {
                        err := fmt.Sprintf("Found non-whitespace" +
                            " chars \"%s\" for state %s", string([]byte(v)),
                            state)
                        return methodName, extractParams(params),
                        &rpc2.Error{Msg:err},
                            faultVal
                    }
                }
            }
        case xml.ProcInst:
            // ignored
        default:
            err := &rpc2.Error{Msg:fmt.Sprintf("Not handling %v (type %T)" +
                    " for state %s", v, v, state)}
            return methodName, extractParams(params), err, faultVal
        }
    }

    return methodName, extractParams(params), nil, faultVal
}

// Translate an XML string into a local data object
func UnmarshalString(s string) (string, interface{}, *rpc2.Error,
    *rpc2.Fault) {
    return Unmarshal(strings.NewReader(s))
}
