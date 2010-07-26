package xmlrpc

import (
    "fmt"
    "reflect"
    "testing"
)

func getTypeString(val interface{}) string {
    pre := "\n        "
    post := "\n      "
    switch v := val.(type) {
    case bool:
        var bVal int
        if v {
            bVal = 1
        } else {
            bVal = 0
        }
        return fmt.Sprintf("%s<boolean>%d</boolean>%s", pre, bVal, post)
    case float:
        // hack to make float values match
        fStr := fmt.Sprintf("%f", v)
        fLen := len(fStr)
        fSub := fStr[fLen-3:fLen]
        if fLen > 3 && fSub != "001" {
            fStr += "001"
        }

        return fmt.Sprintf("%s<double>%s</double>%s", pre, fStr, post)
    case int:
        return fmt.Sprintf("%s<int>%d</int>%s", pre, v, post)
    case string:
        return v
    }

    valKind := reflect.Typeof(val).Kind()
    if valKind == reflect.Array || valKind == reflect.Slice {
        return "<array>foo</array>"
    }

    return fmt.Sprintf("<???>%v(%T)</???>", val, val)
}

func parseAndCheck(t *testing.T, methodName string, expVal interface{},
    xmlStr string) {
    name, val, err := UnmarshalString(xmlStr)
    if err != nil {
        t.Fatalf("Returned error %s", err)
    }

    if name != methodName {
        if methodName == "" {
            t.Fatal("Did not expect method name \"%s\"", name)
        } else {
            t.Fatal("Expected method name \"%s\", not \"%s\"", methodName, name)
        }
    }

    if expVal == nil {
        if val != nil {
            t.Fatalf("Got unexpected value %v <%T>", val, val)
        }
    } else {
        if reflect.Typeof(val) != reflect.Typeof(expVal) {
            t.Fatalf("Returned type %T, not %T", val, expVal)
        }

        if ! reflect.DeepEqual(val, expVal) {
            t.Fatalf("Returned value %v, not %v", val, expVal)
        }
    }
}

func parseUnimplemented(t *testing.T, methodName string, expVal interface{}) {
    xmlStr := wrapMethod(methodName, expVal)
    name, val, err := UnmarshalString(xmlStr)
    if err == nil {
        t.Fatalf("Unimplemented type didn't return an error")
    } else if err.Msg != "Unimplemented" {
        t.Fatalf("Returned unexpected error %s", err)
    }

    if name != methodName {
        if methodName == "" {
            t.Fatal("Did not expect method name \"%s\"", name)
        } else {
            t.Fatal("Expected method name \"%s\", not \"%s\"", methodName, name)
        }
    }

    if val != nil {
        t.Fatalf("Got value %v from unimplemented type", val)
    }
}

func wrapAndParse(t *testing.T, methodName string, expVal interface{}) {
    xmlStr := wrapMethod(methodName, expVal)
    parseAndCheck(t, methodName, expVal, xmlStr)
}

func wrapMethod(methodName string, val interface{}) string {
    var frontStr, backStr string
    if methodName == "" {
        frontStr = "<methodResponse>"
        backStr = "</methodResponse>"
    } else {
        frontStr = fmt.Sprintf(`<methodCall>
  <methodName>%s</methodName>`, methodName)
        backStr = "</methodCall>"
    }

    return fmt.Sprintf(`<?xml version="1.0"?>
%s
  <params>
    <param>
      <value>%v</value>
    </param>
  </params>
%s
`, frontStr, getTypeString(val), backStr)
}

func TestMakeRequestBool(t *testing.T) {
    expVal := true
    methodName := "foo"

    xmlStr, err := Marshal(methodName, expVal)
    if err != nil {
        t.Fatalf("Returned error %s", err)
    }

    expStr := wrapMethod(methodName, expVal)
    if xmlStr != expStr {
        t.Fatalf("Returned \"%s\", not \"%s\"", xmlStr, expStr)
    }
}

func TestMakeRequestDouble(t *testing.T) {
    expVal := 123.123
    methodName := "foo"

    xmlStr, err := Marshal(methodName, expVal)
    if err != nil {
        t.Fatalf("Returned error %s", err)
    }

    expStr := wrapMethod(methodName, expVal)
    if xmlStr != expStr {
        t.Fatalf("Returned \"%s\", not \"%s\"", xmlStr, expStr)
    }
}

func TestMakeRequestInt(t *testing.T) {
    expVal := 123456
    methodName := "foo"

    xmlStr, err := Marshal(methodName, expVal)
    if err != nil {
        t.Fatalf("Returned error %s", err)
    }

    expStr := wrapMethod(methodName, expVal)
    if xmlStr != expStr {
        t.Fatalf("Returned \"%s\", not \"%s\"", xmlStr, expStr)
    }
}

func TestMakeRequestNoData(t *testing.T) {
    methodName := "foo"

    xmlStr, err := Marshal(methodName)
    if err != nil {
        t.Fatalf("Returned error %s", err)
    }

    expStr := fmt.Sprintf(`<?xml version="1.0"?>
<methodCall>
  <methodName>%s</methodName>
  <params>
  </params>
</methodCall>
`, methodName)

    if xmlStr != expStr {
        t.Fatalf("Returned \"%s\", not \"%s\"", xmlStr, expStr)
    }
}

func TestParseRequestInt(t *testing.T) {
    wrapAndParse(t, "foo", 54321)
}

func TestParseResponseNoData(t *testing.T) {
    xmlStr := `<?xml version="1.0"?>
<methodResponse>
  <params>
  </params>
</methodResponse>`

    parseAndCheck(t, "", nil, xmlStr)
}

func TestParseResponseArray(t *testing.T) {
    var array = []int { 1, -1, 0, 1234567 }
    parseUnimplemented(t, "", array)
}

func TestParseResponseBase64(t *testing.T) {
    tnm := "base64"
    val := "eW91IGNhbid0IHJlYWQgdGhpcyE"
    parseUnimplemented(t, "", fmt.Sprintf("<%s>%v</%s>", tnm, val, tnm))
}

func TestParseResponseBool(t *testing.T) {
    const expVal = true

    xmlStr := wrapMethod("", expVal)

    parseAndCheck(t, "", expVal, xmlStr)
}

func TestParseResponseDatetime(t *testing.T) {
    tnm := "dateTime.iso8601"
    val := "19980717T14:08:55"
    parseUnimplemented(t, "", fmt.Sprintf("<%s>%v</%s>", tnm, val, tnm))
}

func TestParseResponseDouble(t *testing.T) {
    wrapAndParse(t, "", 123.456)
}

func TestParseResponseInt(t *testing.T) {
    wrapAndParse(t, "", 1279905716)
}

func TestParseResponseI4(t *testing.T) {
    tnm := "i4"
    val := -433221

    xmlStr := wrapMethod("", fmt.Sprintf("<%s>%v</%s>", tnm, val, tnm))
    parseAndCheck(t, "", val, xmlStr)
}

func TestParseResponseString(t *testing.T) {
    wrapAndParse(t, "", "abc123")
}

func TestParseResponseStringEmpty(t *testing.T) {
    wrapAndParse(t, "", "")
}

func TestParseResponseStringRaw(t *testing.T) {
    const expVal = "abc123"

    xmlStr := fmt.Sprintf(`<?xml version='1.0'?>
<methodResponse>
  <params>
    <param>
      <value>%s</value>
    </param>
  </params>
</methodResponse>`, expVal)

    parseAndCheck(t, "", expVal, xmlStr)
}

func TestParseResponseStringRawEmpty(t *testing.T) {
    xmlStr := `<?xml version='1.0'?>
<methodResponse>
  <params>
    <param>
      <value></value>
    </param>
  </params>
</methodResponse>`

    parseAndCheck(t, "", "", xmlStr)
}

func TestParseResponseStruct(t *testing.T) {
    tnm := "struct"
    val := "Bad value"
    parseUnimplemented(t, "", fmt.Sprintf("<%s>%v</%s>", tnm, val, tnm))
}
