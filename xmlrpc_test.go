package xmlrpc

import (
    "fmt"
    "reflect"
    "testing"
)

func parseAndCheck(t *testing.T, methodName string, typeName string,
    expVal interface{}, xmlStr string) {
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
            t.Fatalf("Returned type %T, not %s", val, typeName)
        }

        if ! reflect.DeepEqual(val, expVal) {
            t.Fatalf("Returned value %v, not %v", val, expVal)
        }
    }
}

func parseUnimplemented(t *testing.T, methodName string, typeName string,
    expVal interface{}) {
    xmlStr := wrapMethod(methodName, typeName, expVal)
    name, val, err := UnmarshalString(xmlStr)
    if err == nil || err.Msg != "Unimplemented" {
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

func wrapAndParse(t *testing.T, methodName string, typeName string,
    expVal interface{}) {
    xmlStr := wrapMethod(methodName, typeName, expVal)
    parseAndCheck(t, methodName, typeName, expVal, xmlStr)
}

func wrapMethod(methodName string, typeName string, val interface{}) string {
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
      <value>
        <%s>%v</%s>
      </value>
    </param>
  </params>
%s
`, frontStr, typeName, val, typeName, backStr)
}

func TestMakeRequestBool(t *testing.T) {
    expVal := true
    methodName := "foo"

    xmlStr, err := Marshal(methodName, expVal)
    if err != nil {
        t.Fatalf("Returned error %s", err)
    }

    var wrapVal int
    if expVal {
        wrapVal = 1
    } else {
        wrapVal = 0
    }

    expStr := wrapMethod(methodName, "boolean", wrapVal)
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

    // hack to make float values match
    expStr := wrapMethod(methodName, "double", fmt.Sprintf("%v001", expVal))
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

    expStr := wrapMethod(methodName, "int", expVal)
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
    wrapAndParse(t, "foo", "int", 54321)
}

func TestParseResponseNoData(t *testing.T) {
    xmlStr := `<?xml version="1.0"?>
<methodResponse>
  <params>
  </params>
</methodResponse>`

    parseAndCheck(t, "", "string", nil, xmlStr)
}

func TestParseResponseArray(t *testing.T) {
    var array = []int { 1, -1, 0, 1234567 }
    parseUnimplemented(t, "", "array", array)
}

func TestParseResponseBase64(t *testing.T) {
    parseUnimplemented(t, "", "base64", "eW91IGNhbid0IHJlYWQgdGhpcyE")
}

func TestParseResponseBool(t *testing.T) {
    const typeName = "boolean"
    const expVal = true

    var boolVal int
    if expVal {
        boolVal = 1
    } else {
        boolVal = 0
    }

    xmlStr := wrapMethod("", typeName, boolVal)

    parseAndCheck(t, "", typeName, expVal, xmlStr)
}

func TestParseResponseDatetime(t *testing.T) {
    parseUnimplemented(t, "", "dateTime.iso8601", "19980717T14:08:55")
}

func TestParseResponseDouble(t *testing.T) {
    wrapAndParse(t, "", "double", 123.456)
}

func TestParseResponseInt(t *testing.T) {
    wrapAndParse(t, "", "int", 1279905716)
}

func TestParseResponseI4(t *testing.T) {
    wrapAndParse(t, "", "i4", -433221)
}

func TestParseResponseString(t *testing.T) {
    wrapAndParse(t, "", "string", "abc123")
}

func TestParseResponseStringEmpty(t *testing.T) {
    wrapAndParse(t, "", "string", "")
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

    parseAndCheck(t, "", "string", expVal, xmlStr)
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

    parseAndCheck(t, "", "string", "", xmlStr)
}

func TestParseResponseStruct(t *testing.T) {
    parseUnimplemented(t, "", "struct", "Not really a structure")
}
