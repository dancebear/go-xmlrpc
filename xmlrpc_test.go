package xmlrpc

import (
    "fmt"
    "testing"
)

func wrapType(s string) string {
    return fmt.Sprintf(`
<?xml version='1.0'?>
<methodResponse>
  <params>
    <param>
      <value>
        %s
      </value>
    </param>
  </params>
</methodResponse>`, s)
}

func TestParseResponseInt(t *testing.T) {
    const intVal = 1279905716

    respText := wrapType(fmt.Sprintf("<int>%d</int>", intVal))

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(int)
    if ! ok {
        t.Fatalf("Returned type %T, not int", val)
    }

    if i != intVal {
        t.Fatalf("Returned value %d, not %d", i, intVal)
    }
}

func TestParseResponseI4(t *testing.T) {
    const i4Val = -433221

    respText := wrapType(fmt.Sprintf("<i4>%d</i4>", i4Val))

    val, err := ParseString(respText, true)
    if err != "" {
        t.Fatalf("Returned error %s", err)
    }

    i, ok := val.(int)
    if ! ok {
        t.Fatalf("Returned type %T, not int", val)
    }

    if i != i4Val {
        t.Fatalf("Returned value %d, not %d", i, i4Val)
    }
}
