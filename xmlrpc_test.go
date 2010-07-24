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

func TestParseResponseBool(t *testing.T) {
    const typeName = "boolean"
    const boolVal = true

    var boolStr string
    if boolVal {
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

    if i != boolVal {
        t.Fatalf("Returned value %d, not %v", i, boolVal)
    }
}

func TestParseResponseInt(t *testing.T) {
    const typeName = "int"
    const intVal = 1279905716

    respText := wrapType(typeName, fmt.Sprintf("%d", intVal), true)

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(int)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != intVal {
        t.Fatalf("Returned value %d, not %d", i, intVal)
    }
}

func TestParseResponseI4(t *testing.T) {
    const typeName = "i4"
    const i4Val = -433221

    respText := wrapType(typeName, fmt.Sprintf("%d", i4Val), true)

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(int)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != i4Val {
        t.Fatalf("Returned value %d, not %d", i, i4Val)
    }
}

func TestParseResponseString(t *testing.T) {
    const typeName = "string"
    const strVal = "abc123"

    respText := wrapType(typeName, strVal, true)

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(string)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != strVal {
        t.Fatalf("Returned value %d, not %v", i, strVal)
    }
}

func TestParseRequestInt(t *testing.T) {
    const typeName = "int"
    const intVal = 54321

    respText := wrapType(typeName, fmt.Sprintf("%d", intVal), false)

    val, err := ParseString(respText, false)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(int)
    if ! ok {
        t.Fatalf("Returned type %T, not %s", val, typeName)
    }

    if i != intVal {
        t.Fatalf("Returned value %d, not %d", i, intVal)
    }
}
