package xmlrpc

import (
    "fmt"
    "reflect"
    "testing"
)

func wrapAndParse(t *testing.T, typeName string, expVal interface{},
    isResp bool) {
    xmlStr := wrapType(typeName, fmt.Sprintf("%v", expVal), isResp)
    parseAndCheck(t, typeName, expVal, isResp, xmlStr)
}

func parseAndCheck(t *testing.T, typeName string, expVal interface{},
    isResp bool, xmlStr string) {
    val, err := UnmarshalString(xmlStr, isResp)
    if err != "" {
        t.Fatalf("Returned error %s", err)
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

func wrapType(typeName string, s string, isResp bool) string {
    var rKey string
    if isResp {
        rKey = "Response"
    } else {
        rKey = "Request"
    }

    return fmt.Sprintf(`
<?xml version='1.0'?>
<method%s>
  <params>
    <param>
      <value>
        <%s>%s</%s>
      </value>
    </param>
  </params>
</method%s>`, rKey, typeName, s, typeName, rKey)
}

func TestParseResponseNoData(t *testing.T) {
    xmlStr := `
<?xml version='1.0'?>
<methodResponse>
  <params>
  </params>
</methodResponse>`

    parseAndCheck(t, "string", nil, true, xmlStr)
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

    xmlStr := wrapType(typeName, boolStr, true)

    parseAndCheck(t, typeName, expVal, true, xmlStr)
}

func TestParseResponseInt(t *testing.T) {
    wrapAndParse(t, "int", 1279905716, true)
}

func TestParseResponseI4(t *testing.T) {
    wrapAndParse(t, "i4", -433221, true)
}

func TestParseResponseString(t *testing.T) {
    wrapAndParse(t, "string", "abc123", true)
}

func TestParseResponseStringEmpty(t *testing.T) {
    wrapAndParse(t, "string", "", true)
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

    parseAndCheck(t, "string", expVal, true, xmlStr)
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

    parseAndCheck(t, "string", expVal, true, xmlStr)
}

func TestParseResponseDouble(t *testing.T) {
    wrapAndParse(t, "double", 123.456, true)
}

func TestParseRequestInt(t *testing.T) {
    wrapAndParse(t, "int", 54321, false)
}
