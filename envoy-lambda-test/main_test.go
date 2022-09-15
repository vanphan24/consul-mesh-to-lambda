package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleRequest(t *testing.T) {
	cases := map[string]struct {
		statusCode int
		input      interface{}
	}{
		"payload passthrough = true, object": {
			statusCode: http.StatusOK,
			input: map[string]interface{}{
				"id":      "payload-passthrough",
				"message": "payload passthrough is true",
			},
		},
		"payload passthrough = false": {
			statusCode: http.StatusOK,
			input: map[string]interface{}{
				"is_base64_encoded": false,
				"body": map[string]interface{}{
					"id":      "id1",
					"message": "message",
				},
			},
		},
		"base64-encoded": {
			statusCode: http.StatusOK,
			input: map[string]interface{}{
				"is_base64_encoded": true,
				"body":              "eyJpZCI6InRlc3QtaWQiLCJtZXNzYWdlIjoiSGVsbG8sIHdvcmxkIn0=",
			},
		},
		"string": {
			statusCode: http.StatusOK,
			input:      "testing",
		},
		"bool": {
			statusCode: http.StatusOK,
			input:      true,
		},
		"int": {
			statusCode: http.StatusOK,
			input:      69,
		},
		"string-body": {
			statusCode: http.StatusOK,
			input:      map[string]interface{}{"body": "testing"},
		},
		"bool-body": {
			statusCode: http.StatusOK,
			input:      map[string]interface{}{"body": true},
		},
		"int-body": {
			statusCode: http.StatusOK,
			input:      map[string]interface{}{"body": 69},
		},
	}

	t.Parallel()
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			c := c

			// If the type is a primitive (not an object), or we're simulating payload_passthrough = true
			// then the expected response is the input for the test case.
			expected := c.input
			input, inputIsMap := c.input.(map[string]interface{})
			if inputIsMap {
				body, inputHasBody := input["body"]
				if inputHasBody {
					// payload_passthrough = false so the expected response is the input body field.
					expected = body
				}
			}

			response, err := HandleRequest(c.input)
			require.NoError(t, err)
			require.Equal(t, c.statusCode, response["statusCode"])
			require.Equal(t, expected, response["body"])
		})
	}
}
