#!/bin/bash

curl -X POST https://sqs.eu-north-1.amazonaws.com/ \
  -H "Accept-Encoding: gzip" \
  -H "Amz-Sdk-Invocation-Id: 77a08f06-009c-4e69-9dba-be350a19b7c7" \
  -H "Amz-Sdk-Request: attempt=1; max=3" \
  -H "Authorization: AWS4-HMAC-SHA256 Credential=AKIA5SBGXRVO2DWNG32U/20250318/eu-north-1/sqs/aws4_request, SignedHeaders=amz-sdk-invocation-id;amz-sdk-request;content-length;content-type;host;x-amz-date;x-amz-target;x-amzn-query-mode, Signature=186d753cc6a44c2c04bf49c425a377a46cdb364a4aa20068a04c17ea52659bc8" \
  -H "Content-Length: 120" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -H "Host: sqs.eu-north-1.amazonaws.com" \
  -H "User-Agent: aws-sdk-go-v2/1.36.3 ua/2.1 os/linux lang/go#1.22.1 md/GOOS#linux md/GOARCH#arm64 api/sqs#1.38.1 m/E,g" \
  -H "X-Amz-Date: 20250318T141442Z" \
  -H "X-Amz-Target: AmazonSQS.ReceiveMessage" \
  -H "X-Amzn-Query-Mode: true" \
#   -d '{"MaxNumberOfMessages":10,"QueueUrl":"https://sqs.eu-north-1.amazonaws.com/932089138525/allen-test","WaitTimeSeconds":4}'


# Wait for both background processes to finish
wait