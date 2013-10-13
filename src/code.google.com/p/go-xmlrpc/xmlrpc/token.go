package xmlrpc

import (
    "encoding/xml"
    "errors"
    "fmt"
)

const (
    // unknown token
    TokenUnknown = -3

    // marker for XML character data
    TokenText = -2

    // ignored XML data
    TokenProcInst = -1

    // keyword tokens
    TokenFault = iota
    TokenMember
    TokenMethodCall
    TokenMethodName
    TokenMethodResponse
    TokenName
    TokenParam
    TokenParams
    TokenValue

    // marker for data types
    TokenDataType

    // data type tokens
    TokenArray
    TokenBase64
    TokenBoolean
    TokenData
    TokenDateTime
    TokenDouble
    TokenInt
    TokenNil
    TokenString
    TokenStruct
)

var tokenMap map[string]int

func initTokenMap() {
    tokenMap = make(map[string]int)
    tokenMap["array"] = TokenArray
    tokenMap["base64"] = TokenBase64
    tokenMap["boolean"] = TokenBoolean
    tokenMap["data"] = TokenData
    tokenMap["dateTime.iso8601"] = TokenDateTime
    tokenMap["double"] = TokenDouble
    tokenMap["fault"] = TokenFault
    tokenMap["int"] = TokenInt
    tokenMap["member"] = TokenMember
    tokenMap["methodCall"] = TokenMethodCall
    tokenMap["methodName"] = TokenMethodName
    tokenMap["methodResponse"] = TokenMethodResponse
    tokenMap["name"] = TokenName
    tokenMap["nil"] = TokenNil
    tokenMap["param"] = TokenParam
    tokenMap["params"] = TokenParams
    tokenMap["string"] = TokenString
    tokenMap["struct"] = TokenStruct
    tokenMap["value"] = TokenValue
}

type XToken struct {
    token int
    isStart bool
    text string
}

func (tok *XToken) Is(val int) bool {
    return tok.token == val
}

func (tok *XToken) IsDataType() bool {
    return tok.token > TokenDataType
}

func (tok *XToken) IsNone() bool {
    return tok.token == TokenProcInst
}

func (tok *XToken) IsStart() bool {
    return tok.token >= 0 && tok.isStart
}

func (tok *XToken) IsText() bool {
    return tok.token == TokenText
}

func (tok *XToken) Name() string {
    if tok.token == TokenProcInst {
        return "ProcInst"
    }

    for k, v := range tokenMap {
        if v == tok.token {
            return k
        }
    }

    return fmt.Sprintf("??#%d??", tok.token)
}

func (tok *XToken) Text() string {
    if tok.token != TokenText {
        return ""
    }

    return tok.text
}

func (tok *XToken) String() string {
    if tok.token == TokenText {
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
            tok = TokenInt
        } else {
            return TokenUnknown, errors.New(fmt.Sprintf("Unknown tag <%s>",
                tag))
        }
    }

    return tok, nil
}

func getNextToken(p *xml.Decoder) (*XToken, error) {
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
        return &XToken{token:tok, isStart:true}, nil
    case xml.EndElement:
        tok, err := getTagToken(v.Name.Local)
        if err != nil {
if(debug2){fmt.Printf("P!!#3 %s\n", err.Error())}
            return nil, err
        }

if(debug2){fmt.Printf("P-> </%s>#%d\n", v.Name.Local, tok)}
        return &XToken{token:tok, isStart:false}, nil
    case xml.CharData:
if(debug2){fmt.Printf("P-> \"%s\"\n", string(v))}
        return &XToken{token:TokenText, text:string(v)}, nil
    case xml.ProcInst:
        return &XToken{token:TokenProcInst}, nil
    default:
if(debug2){fmt.Printf("P!! %v(%T)\n", v, v)}
        err := errors.New(fmt.Sprintf("Not handling <value> %v (type %T)",
            v, v))
        return nil, err
    }
}
