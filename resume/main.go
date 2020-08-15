package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Response of type ApiGatewayProxyResponse
type Response events.APIGatewayProxyResponse

// GetItemOutput a struct that represents the output of GetItem
type GetItemOutput struct {
	StatusCode int
	Body       string
}

// Handler lambda proxy handler that will take
func Handler(request events.APIGatewayProxyRequest) (Response, error) {
	var bodyBuilder strings.Builder
	var statusCode int

	// check if user agent is not curl. If it isn't append a note at the beginning
	// to let the client know this was designed to be used via curl..
	if !strings.Contains(request.RequestContext.Identity.UserAgent, "curl") {
		fmt.Print("User Agent is: ", request.RequestContext.Identity.UserAgent)
		bodyBuilder.WriteString("Note: these endpoints were designed to be used with curl. They may not appear as intended when accessed via other methods. \n")
	}

	// If the path is "/" and there is not an item in path parameters
	if request.Path == "/" {
		requestedItem, err := GetItem("resume")
		if err != nil {
			return HandleGeneralErr(err), nil
		}
		bodyBuilder.WriteString(requestedItem.Body)
		statusCode = 200

		// if item is requested, return that specific item.
	} else if value, present := request.PathParameters["item"]; present && request.RequestContext.ResourcePath == "/{item}" {

		requestedItem, err := GetItem(value)
		if err != nil {
			fmt.Print("failed when trying to get ", value)

			// test to see if this is an aws error.
			if awserror, ok := err.(awserr.Error); ok {
				switch awserror.Code() {
				case s3.ErrCodeNoSuchKey:
					return HandleNoSuchItemError(request.Path), nil
				}
			}
		}

		bodyBuilder.WriteString(requestedItem.Body)
		statusCode = 200
	} else {

		return HandleNoSuchItemError(request.Path), nil
	}

	fmt.Print("body builder: ", bodyBuilder.String())
	response := Response{
		StatusCode:      statusCode,
		IsBase64Encoded: false,
		Body:            bodyBuilder.String(),
		Headers: map[string]string{
			"Content-Type":           "text/plain",
			"X-MyCompany-Func-Reply": "resume-handler",
		},
	}

	return response, nil
}

// GetItem read the desired item from s3 and return its contents as a string
func GetItem(item string) (GetItemOutput, error) {

	fmt.Print("I am attempting to get an item")

	// create new session to pass to s3 client
	sess, err := session.NewSession()
	if err != nil {
		fmt.Print("an error occured when attempting to create a new session")
		return GetItemOutput{
			StatusCode: 500,
			Body:       "",
		}, err
	}
	s3Client := s3.New(sess)
	input := &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("storageBucket")),
		Key:    aws.String(fmt.Sprintf("%s/%s.txt", os.Getenv("baseKey"), item)),
	}

	// try to get object
	obj, err := s3Client.GetObject(input)
	if err != nil {
		fmt.Print("An error occured when attempting to get item: ", item)
		return GetItemOutput{
			StatusCode: 500,
			Body:       "",
		}, err
	}
	// try to read file body.
	readFileText, err := ioutil.ReadAll(obj.Body)
	if err != nil {
		fmt.Print("An error occured when trying to read item", item)
		return GetItemOutput{
			StatusCode: 500,
			Body:       "",
		}, err
	}

	var responseBuilder strings.Builder
	responseBuilder.Write(readFileText)

	// make sure text ends with a new line.
	responseBuilder.WriteString(" \n")
	response := responseBuilder.String()
	return GetItemOutput{
		StatusCode: 200,
		Body:       response,
	}, nil

}

// HandleGeneralErr should be called when only a general error needs to be sent back as response.
func HandleGeneralErr(err error) Response {
	resp := Response{
		StatusCode:      500,
		IsBase64Encoded: false,
		Body:            "An internal error has occured and your request cannot be completed. Please try again later. \n",
		Headers: map[string]string{
			"Content-Type":           "text/plain",
			"X-MyCompany-Func-Reply": "resume-handler",
		},
	}
	fmt.Print(err)
	return resp
}

// HandleNoSuchItemError Handles a bad item request
func HandleNoSuchItemError(item string) Response {
	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("%s is not a valid endpoint. \n\n", item))

	// if another error occurs when attempting to list end points we will log it and throw a general error and log that it happened.
	available, err := ListAvailableEndPoints()
	if err != nil {
		fmt.Print("failed when listing endpoints.")
		return HandleGeneralErr(err)
	}
	responseBuilder.WriteString(available)

	resp := Response{
		StatusCode:      404,
		IsBase64Encoded: false,
		Body:            responseBuilder.String(),
		Headers: map[string]string{
			"Content-Type":           "text/plain",
			"X-MyCompany-Func-Reply": "resume-handler",
		},
	}

	return resp
}

// ListAvailableEndPoints loops through bucket to see what items are available
func ListAvailableEndPoints() (string, error) {
	var responseBuilder strings.Builder
	responseBuilder.WriteString("The following endpoints are available: \n")

	s3Client := s3.New(session.New())
	input := &s3.ListObjectsInput{
		Bucket: aws.String(os.Getenv("storageBucket")),
		Prefix: aws.String(os.Getenv("baseKey")),
	}
	listOfObj, err := s3Client.ListObjects(input)
	if err != nil {
		return "", err
	}

	for _, obj := range listOfObj.Contents {
		responseBuilder.WriteString(fmt.Sprint(aws.StringValue(obj.Key), "\n"))
	}

	return responseBuilder.String(), nil
}

func main() {
	lambda.Start(Handler)
}
