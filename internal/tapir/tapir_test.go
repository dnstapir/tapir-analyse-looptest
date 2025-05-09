package tapir

import (
    "bytes"
    "testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

    "github.com/dnstapir/tapir-analyse-looptest/app/ext"
)

func TestSchemaValidationSha90202b31(t *testing.T) {
	var tests = []struct {
		name     string
		indata   string
	}{
		{"basic", "example.com"},
	}

	schemaCompiler := jsonschema.NewCompiler()
	schema, err := schemaCompiler.Compile("testdata/90202b31b10745c8f92f2a5d3ca02cef1f303a97.json")
	if err != nil {
        t.Fatalf("Error compiling schema: %s", err)
	}

    tapirHandle := Handle {
        Log: ext.FakeLogger{},
    }

	for _, tt := range tests {
        schemaCopy := schema
		t.Run(tt.name, func(t *testing.T) {
            testMsg, err := tapirHandle.GenerateMsg(tt.indata, 2048)
	        if err != nil {
                t.Fatalf("Error generating message: %s", err)
	        }

	        testMsgReader := bytes.NewReader([]byte(testMsg))
	        testJsonObj, err := jsonschema.UnmarshalJSON(testMsgReader)
	        if err != nil {
                t.Fatalf("Error unmarshalling byte stream: %s", err)
	        }

	        err = schemaCopy.Validate(testJsonObj)
	        if err != nil {
                t.Fatalf("Error validating tapir message: %s. Got: %s", err, testMsg)
	        }
		})
	}
}


