# Lambda Test Function for Envoy
This is a Lambda function written in Go that can be used to test with the Envoy AWS Lambda HTTP filter.
https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/aws_lambda_filter

The function simply copies the request body to the response body and returns with a `200 OK` status.

## Usage

### Clone this repository

To get started, clone the repository.

```sh
git clone https://github.com/cthain/envoy-lambda-test
cd envoy-lambda-test
```

### Create a zip file

Create a `.zip` file that can be deployed to Lambda via the AWS console.

```sh
GOOS=linux go build main.go && zip envoy-lambda-test.zip main
```

Once the zip file is built you can create a Lambda function from the AWS console.

### Configure Envoy

Edit the [`envoy-config.yaml`](./envoy-config.yaml) configuration file:
- Replace `<REGION>` with the AWS region of your Lambda function.
- Replace `<LAMBDA-FUNCTION-ARN>` with the ARN of your Lambda function.

_**Note**_: The `strip_any_host_port` option must be set to `true` in the configuration.
If it is not set or set to `false` the AWS request signature won't match and you will
get an error like the following when calling the Lambda function:

```json
{"message":"The request signature we calculated does not match the signature you provided. Check your AWS Secret Access Key and signing method. Consult the service documentation for details."}
```

### Launch Envoy

You must provide Envoy with AWS credentials that have the `lambda:InvokeFunction` permissions.
This can be done using [environment variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html).

```sh
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...

```

Launch Envoy.
The example below uses the `envoyproxy/envoy` Docker image to launch Envoy with the static configuration provided in the file.
The AWS credentials are provided to Envoy as environment variables.

```sh
docker run -d --rm --name envoy \
  -e AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" \
  -e AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
  -e AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN \
  -p 9901:9901 \
  -p 10000:10000 \
  -v $(pwd)/envoy-config.yaml:/envoy-config.yaml \
  envoyproxy/envoy:v1.22.2 -c /envoy-config.yaml
```

## Call the Lambda function

Call the Lambda function through the proxy:

```sh
curl -s localhost:10000 -d '"hello world!"' | jq .
```

Envoy prints the response:

```json
{
  "body": "hello world!",
  "statusCode": 200
}
```
