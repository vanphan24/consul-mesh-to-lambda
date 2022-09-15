package main

import (
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
)

// HandleRequest handles a request from the Envoy aws_lambda http filter for the consul integration tests.
// See https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/aws_lambda_filter
//
// The body of the request is simply copied to the response body.
// It handles both cases for payload passthrough, true and false.
func HandleRequest(i interface{}) (map[string]interface{}, error) {
	// Copy the request body to the response body
	response := make(map[string]interface{})
	response["statusCode"] = http.StatusOK
	response["body"] = i

	request, ok := i.(map[string]interface{})
	if !ok {
		// The input is a primitive type or payload_passthrough is true so return
		return response, nil
	}

	if body, ok := request["body"]; ok {
		// There's a body field so assume that payload_passthrough = false
		// The response body is the body field of the request.
		response["body"] = body
		if encoded, ok := request["is_base64_encoded"]; ok {
			// The input is base64 encoded so set the flag on the output.
			response["isBase64Encoded"] = encoded
		}
	}
	return response, nil
}

func main() {
	lambda.Start(HandleRequest)
}
