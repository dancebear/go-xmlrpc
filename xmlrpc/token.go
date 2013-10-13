package xmlrpc

import (
    "encoding/xml"
    "errors"
    "fmt"
)

// internal XML parser tokens
const (
    // unknown token
    tokenUnknown = -3

    // marker for XML character data
    tokenText = -2

    // ignored XML data
    tokenProcInst = -1

    // keyword tokens
    tokenFault = iota
    tokenMember
    tokenMethodCall
    tokenMethodName
    tokenMethodResponse
    tokenName
    tokenParam
    tokenParams
    tokenValue

    // marker for data types
    tokenDataType

    // data type tokens
    tokenArray
    tokenBase64
    tokenBoolean
    tokenData
    tokenDateTime
    tokenDouble
    tokenInt
    tokenNil
    tokenString
    tokenStruct
)

// map token strings to constant values
var tokenMap map[string]int

// load the tokens into the token map
func initTokenMap() {
    tokenMap = make(map[string]int)
    tokenMap["array"] = tokenArray
    tokenMap["base64"] = tokenBase64
    tokenMap["boolean"] = tokenBoolean
    tokenMap["data"] = tokenData
    tokenMap["dateTime.iso8601"] = tokenDateTime
    tokenMap["double"] = tokenDouble
    tokenMap["fault"] = tokenFault
    tokenMap["int"] = tokenInt
    tokenMap["member"] = tokenMember
    tokenMap["methodCall"] = tokenMethodCall
    tokenMap["methodName"] = tokenMethodName
    tokenMap["methodResponse"] = tokenMethodResponse
    tokenMap["name"] = tokenName
    tokenMap["nil"] = tokenNil
    tokenMap["param"] = tokenParam
    tokenMap["params"] = tokenParams
    tokenMap["string"] = tokenString
    tokenMap["struct"] = tokenStruct
    tokenMap["value"] = tokenValue
}

type xmlToken struct {
    token int
    isStart bool
    text string
}

func (tok *xmlToken) Is(val int) bool {
    return tok.token == val
}

func (tok *xmlToken) IsDataType() bool {
    return tok.token > tokenDataType
}

func (tok *xmlToken) IsNone() bool {
    return tok.token == tokenProcInst
}

func (tok *xmlToken) IsStart() bool {
    return tok.token >= 0 && tok.isStart
}

func (tok *xmlToken) IsText() bool {
    return tok.token == tokenText
}

func (tok *xmlToken) Name() string {
    if tok.token == tokenProcInst {
        return "ProcInst"
    }

    for k, v := range tokenMap {
        if v == tok.token {
            return k
        }
    }

    return fmt.Sprintf("??#%d??", tok.token)
}

func (tok *xmlToken) Text() string {
    if tok.token != tokenText {
        return ""
    }

    return tok.text
}

func (tok *xmlToken) String() string {
    if tok.token == tokenText {
        return fmt.Sprintf("\"%v\"", tok.text)
    }

    var slash string
    if tok.isStart {
        slash = ""
    } else {
        slash = "/"
    }

    return fmt.Sprintf("{%s%s#%d}", slash, tok.Name(), tok.token)
}

func getTagToken(tag string) (int, error) {
    tok, ok := tokenMap[tag]
    if !ok {
        if tag == "i4" {
            tok = tokenInt
        } else {
            return tokenUnknown, errors.New(fmt.Sprintf("Unknown tag <%s>",
                tag))
        }
    }

    return tok, nil
}

func getNextToken(p *xml.Decoder) (*xmlToken, error) {
const debug2 = false
    tag, err := p.Token()
    if tag == nil {
if(debug2){fmt.Printf("P-> EOF\n")}
        return nil, nil
    } else if err != nil {
if(debug2){fmt.Printf("P!!#1 %s\n", err.Error())}
        return nil, err
    }

    const debug = false
    if debug {
        var tagStr string
        if t2, ok := tag.(xml.CharData); ok {
            tagStr = string([]byte(t2))
        } else {
            tagStr = fmt.Sprintf("%v", tag)
        }

        fmt.Printf("tag %s<%T>\n", tagStr, tagStr)
    }

    if tokenMap == nil {
        initTokenMap()
    }

    switch v := tag.(type) {
    case xml.StartElement:
        tok, err := getTagToken(v.Name.Local)
        if err != nil {
if(debug2){fmt.Printf("P!!#2 %v\n", err)}
            return nil, err
        }

if(debug2){fmt.Printf("P-> <%s>#%d\n", v.Name.Local, tok)}
        return &xmlToken{token:tok, isStart:true}, nil
    case xml.EndElement:
        tok, err := getTagToken(v.Name.Local)
        if err != nil {
if(debug2){fmt.Printf("P!!#3 %s\n", err.Error())}
            return nil, err
        }

if(debug2){fmt.Printf("P-> </%s>#%d\n", v.Name.Local, tok)}
        return &xmlToken{token:tok, isStart:false}, nil
    case xml.CharData:
if(debug2){fmt.Printf("P-> \"%s\"\n", string(v))}
        return &xmlToken{token:tokenText, text:string(v)}, nil
    case xml.ProcInst:
        return &xmlToken{token:tokenProcInst}, nil
    default:
if(debug2){fmt.Printf("P!! %v(%T)\n", v, v)}
        err := errors.New(fmt.Sprintf("Not handling <value> %v (type %T)",
            v, v))
        return nil, err
    }
}
