package xmlrpc

import (
    "fmt"
    "reflect"
    "testing"
)

func wrapAndParse(t *testing.T, methodName string, typeName string,
    expVal interface{}) {
    xmlStr := wrapType(methodName, typeName, fmt.Sprintf("%v", expVal))
    parseAndCheck(t, methodName, typeName, expVal, xmlStr)
}

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
    xmlStr := wrapType(methodName, typeName, fmt.Sprintf("%v", expVal))
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

func wrapType(methodName string, typeName string, s string) string {
    var frontStr, backStr string
    if methodName == "" {
        frontStr = "<methodResponse>"
        backStr = "</methodResponse>"
    } else {
        frontStr = fmt.Sprintf("<methodCall><methodName>%s</methodName>",
            methodName)
        backStr = "</methodCall>"
    }

    return fmt.Sprintf(`
<?xml version='1.0'?>
%s
  <params>
    <param>
      <value>
        <%s>%s</%s>
      </value>
    </param>
  </params>
%s`, frontStr, typeName, s, typeName, backStr)
}

func TestParseRequestInt(t *testing.T) {
    wrapAndParse(t, "foo", "int", 54321)
}

func TestParseResponseNoData(t *testing.T) {
    xmlStr := `
<?xml version='1.0'?>
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

    var boolStr string
    if expVal {
        boolStr = "1"
    } else {
        boolStr = "0"
    }

    xmlStr := wrapType("", typeName, boolStr)

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

    xmlStr := fmt.Sprintf(`
<?xml version='1.0'?>
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
    const expVal = ""

    xmlStr := fmt.Sprintf(`
<?xml version='1.0'?>
<methodResponse>
  <params>
    <param>
      <value>%s</value>
    </param>
  </params>
</methodResponse>`, expVal)

    parseAndCheck(t, "", "string", expVal, xmlStr)
}

func TestParseResponseStruct(t *testing.T) {
    parseUnimplemented(t, "", "struct", "Not really a structure")
}
