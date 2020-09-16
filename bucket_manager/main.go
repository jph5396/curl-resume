package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

//Handler lambda event handler that will execute when
// an event occurs in the resume bucket. It will manage
// the file that lists available "endpoints".
func Handler(event events.S3Event) error {

	// get the first record in the recordlist.
	eventData := event.Records[0]

	fmt.Printf("operation on item %v. event type: %v", eventData.S3.Object.Key, eventData.EventName)

	s3Client := s3.New(session.New())
	input := &s3.ListObjectsInput{
		Bucket: aws.String(os.Getenv("storageBucket")),
		Prefix: aws.String(os.Getenv("baseKey")),
	}
	listOfObj, err := s3Client.ListObjects(input)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("The following endpoints are available: \n")

	for _, obj := range listOfObj.Contents {

		objectKey := aws.StringValue(obj.Key)

		// ListObjects will include the basekey as an item in the list.
		// since we do not want this to be returned, we skip to the
		// next iteration in the loop when the object key and base key are equal.
		if os.Getenv("baseKey") == objectKey {
			continue
		}

		//remove baseKey and .txt
		noBaseKey := strings.ReplaceAll(objectKey, os.Getenv("baseKey"), "")
		noTxt := strings.ReplaceAll(noBaseKey, ".txt", "")
		b.WriteString(fmt.Sprint("/", noTxt, "\n"))
	}

	body := []byte(b.String())
	putInput := &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("storageBucket")),
		Key:    aws.String(os.Getenv("endPoints")),
		Body:   bytes.NewReader(body),
	}

	//we are only concerned with the operations response if it has an error
	_, err = s3Client.PutObject(putInput)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	lambda.Start(Handler)
}
