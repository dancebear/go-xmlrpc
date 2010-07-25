package xmlrpc

import (
    "fmt"
    "testing"
)

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
    const typeName = "string"

    respText := `
<?xml version='1.0'?>
<methodResponse>
  <params>
  </params>
</methodResponse>`

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    if val != nil {
        t.Fatalf("Got unexpected value %v <%T>", val, val)
    }
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

    respText := wrapType(typeName, boolStr, true)

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(bool)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != expVal {
        t.Fatalf("Returned value %v, not %v", i, expVal)
    }
}

func TestParseResponseInt(t *testing.T) {
    const typeName = "int"
    const expVal = 1279905716

    respText := wrapType(typeName, fmt.Sprintf("%v", expVal), true)

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(int)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != expVal {
        t.Fatalf("Returned value %v, not %v", i, expVal)
    }
}

func TestParseResponseI4(t *testing.T) {
    const typeName = "i4"
    const expVal = -433221

    respText := wrapType(typeName, fmt.Sprintf("%v", expVal), true)

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(int)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != expVal {
        t.Fatalf("Returned value %v, not %v", i, expVal)
    }
}

func TestParseResponseString(t *testing.T) {
    const typeName = "string"
    const expVal = "abc123"

    respText := wrapType(typeName, expVal, true)

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(string)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != expVal {
        t.Fatalf("Returned value %v, not %v", i, expVal)
    }
}

func TestParseResponseRawString(t *testing.T) {
    const typeName = "string"
    const expVal = "abc123"

    respText := fmt.Sprintf(`
<?xml version='1.0'?>
<methodResponse>
  <params>
    <param>
      <value>%s</value>
    </param>
  </params>
</methodResponse>`, expVal)

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(string)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != expVal {
        t.Fatalf("Returned value %v, not %v", i, expVal)
    }
}

func TestParseResponseDouble(t *testing.T) {
    const typeName = "double"
    const expVal = 123.456

    respText := wrapType(typeName, fmt.Sprintf("%v", expVal), true)

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(float)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != expVal {
        t.Fatalf("Returned value %v, not %v", i, expVal)
    }
}

func TestParseRequestInt(t *testing.T) {
    const typeName = "int"
    const expVal = 54321

    respText := wrapType(typeName, fmt.Sprintf("%v", expVal), false)

    val, err := ParseString(respText, false)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(int)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != expVal {
        t.Fatalf("Returned value %v, not %v", i, expVal)
    }
}
