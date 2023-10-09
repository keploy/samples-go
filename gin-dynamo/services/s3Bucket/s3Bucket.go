package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	"github.com/aws/aws-sdk-go/service/s3"
)

var s3session s3.S3
var dynamo dynamodb.DynamoDB
var awsSes *session.Session

func Init() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-west-2"), // You can use any valid region.
		Credentials: credentials.NewStaticCredentials("dummy", "dummy", "dummy"),
		Endpoint:    aws.String("http://127.0.0.1:8000/"),
	}))

	s3 := s3.New(sess, &aws.Config{
		DisableRestProtocolURICleaning: aws.Bool(true),
	})
	awsSes = sess
	s3session = *s3
	dynamo = *dynamodb.New(sess)
}

func S3Service() *s3.S3 {
	return &s3session
}

func DynamodbService() *dynamodb.DynamoDB {
	return &dynamo
}
func AwsSession() *session.Session {
	return awsSes
}
