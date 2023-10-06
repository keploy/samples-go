package helpers

import (
	"context"
	"fmt"
	"user-onboarding/config"
	dynamo "user-onboarding/services/s3Bucket"
	structs "user-onboarding/struct"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"golang.org/x/crypto/bcrypt"
)

func UserDetails(ctx context.Context, request *structs.SignUp) error {
	svc := dynamodb.New(dynamo.AwsSession())

	creationInfo := structs.UserDetails{
		CreationSource:     request.CreationSource,
		CreationSourceType: request.CreationSourceType,
	}
	key, err := dynamodbattribute.MarshalMap(creationInfo)
	if err != nil {
		return err
	}
	fmt.Println(config.Get().Table,"dsakgsdiuagsiudhaoshducasiudciuasicbasidbcaiusbdiu")
	input := &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(config.Get().Table),
	}
	result, err := svc.GetItem(input)
	if err != nil || result.Item != nil {
		fmt.Println(err)
		err = fmt.Errorf("email already exists")
		return err
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(request.Password), 1)
	creationInfo.Password = string(hashedPassword)

	key, err = dynamodbattribute.MarshalMap(creationInfo)

	if err != nil {
		return err
	}
	_, err = svc.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(config.Get().Table),
		Item:      key,
	})

	if err != nil {
		return err
	}

	return nil
}

func FetchUser(ctx context.Context, request *structs.UserDetails) (map[string]*dynamodb.AttributeValue, error) {
	svc := dynamodb.New(dynamo.AwsSession())

	email := structs.UserDetails{
		Email: request.Email,
	}

	key, err := dynamodbattribute.MarshalMap(email)

	if err != nil {
		return map[string]*dynamodb.AttributeValue{}, err
	}

	input := &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(config.Get().Table),
	}
	result, err := svc.GetItem(input)

	if err != nil {
		return map[string]*dynamodb.AttributeValue{}, err
	}

	return result.Item, err
}

func UserLogin(ctx context.Context, request *structs.SignUp) error {
	svc := dynamodb.New(dynamo.AwsSession())

	loginInfo := structs.UserDetails{
		CreationSource:     request.CreationSource,
		CreationSourceType: request.CreationSourceType,
	}
	key, err := dynamodbattribute.MarshalMap(loginInfo)

	if err != nil {
		return err
	}
	input := &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(config.Get().Table),
	}
	result, err := svc.GetItem(input)

	if err != nil {
		return err
	}

	userPassword := ""

	for key, v := range result.Item {
		if key == "password" {
			userPassword = *v.S
			break
		}
	}
	err = bcrypt.CompareHashAndPassword([]byte(userPassword), []byte(request.Password))

	if err != nil {
		return err
	}
	return err
}

func UpdateUserDetails(ctx context.Context, request *structs.UserDetails) error {
	svc := dynamodb.New(dynamo.AwsSession())

	creationInfo := structs.UserDetails{
		CreationSource:     request.CreationSource,
		CreationSourceType: request.CreationSourceType,
	}
	key, err := dynamodbattribute.MarshalMap(creationInfo)
	if err != nil {
		return err
	}
	input := &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(config.Get().Table),
	}
	result, err := svc.GetItem(input)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if request.Password != "" {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(request.Password), 1)
		request.Password = string(hashedPassword)
	}

	existingData := structs.UserDetails{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &existingData)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if request.BankDetails == "" && existingData.BankDetails != "" {
		request.BankDetails = existingData.BankDetails
	}
	if request.Bio == "" && existingData.Bio != "" {
		request.Bio = existingData.Bio
	}
	if request.Email == "" && existingData.Email != "" {
		fmt.Println(request.Email, existingData.Email)
		request.Email = existingData.Email
	}
	if request.Phone == "" && existingData.Phone != "" {
		request.Phone = existingData.Phone
	}
	if request.CreationSource == "" && existingData.CreationSource != "" {
		request.CreationSource = existingData.CreationSource
	}
	if request.CreationSourceType == "" && existingData.CreationSourceType != "" {
		request.CreationSourceType = existingData.CreationSourceType
	}
	if request.Skills == "" && existingData.Skills != "" {
		request.Skills = existingData.Skills
	}
	if request.SocialHandles == "" && existingData.SocialHandles != "" {
		request.SocialHandles = existingData.SocialHandles
	}

	if err != nil {
		return err
	}

	key, err = dynamodbattribute.MarshalMap(request)
	if err != nil {
		return err
	}

	_, err = svc.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(config.Get().Table),
		Item:      key,
	})

	if err != nil {
		return err
	}

	return nil
}
