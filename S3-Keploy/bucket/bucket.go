package bucket

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type BucketBasics struct {
	S3Client *s3.Client
}

func (basics BucketBasics) ListAllBuckets() (buckets []string) {
	result, err := basics.S3Client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		fmt.Printf("Couldn't list buckets for your account. Here's why: %v\n", err)
		return
	}
	if len(result.Buckets) == 0 {
		return append(buckets, "You don't have any buckets!")
	} else {
		for _, bucket := range result.Buckets {
			buckets = append(buckets, *bucket.Name)
		}
	}
	return buckets
}

func (basics BucketBasics) DeleteOneBucket(bucket string) (message string) {
	var msg string
	_, err := basics.S3Client.DeleteBucket(context.TODO(), &s3.DeleteBucketInput{
		Bucket: aws.String(bucket)})
	if err != nil {
		msg = "Couldn't delete bucket " + bucket + ". Here's why: " + err.Error()
	} else {
		msg = "Bucket " + bucket + " deleted successfully!"
	}
	return msg
}

func (basics BucketBasics) CreateOneBucket(bucket string) (message string) {

	_, err := basics.S3Client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint("ap-south-1"),
		},
	})
	var msg string
	if err != nil {
		log.Printf("Couldn't create bucket %v in Region %v. Here's why: %v\n",
			bucket, "ap-south-1", err)
	} else {
		msg = bucket + " Bucket created successfully!"
	}
	return msg
}

func (basic BucketBasics) UploadFile(fileName string, bucketName string) (message string) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Couldn't open file %v to upload. Here's why: %v\n", fileName, err)
		return "File " + fileName + " not uploaded"
	} else {
		defer file.Close()
		_, err = basic.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileName),
			Body:   file,
		})
		if err != nil {
			log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n",
				fileName, bucketName, fileName, err)
			return "File " + fileName + " not uploaded"
		}
		return "File " + fileName + " uploaded successfully"
	}
}

func (basic BucketBasics) DeleteAllObjects(bucketName string) (message string) {
	result, err := basic.S3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return "Couldn't delete objects from bucket " + bucketName + " . Here's why: " + err.Error() + "\n"
	}
	var objectKeys []string
	for _, key := range result.Contents {
		objectKeys = append(objectKeys, *key.Key)
	}
	var objectIds []types.ObjectIdentifier
	for _, key := range objectKeys {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: aws.String(key)})
	}
	_, err = basic.S3Client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
		Bucket: aws.String(bucketName),
		Delete: &types.Delete{Objects: objectIds},
	})
	if err != nil {
		return "Couldn't delete objects from bucket " + bucketName + " . Here's why: " + err.Error() + "\n"
	} else {
		return "All objects deleted successfully"
	}
}

func (basic BucketBasics) GetAllObjects(bucketName string) []types.Object {
	result, err := basic.S3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	var contents []types.Object
	if err != nil {
		log.Printf("Couldn't list objects in bucket %v. Here's why: %v\n", bucketName, err)
	} else {
		contents = result.Contents
	}
	return contents
}
