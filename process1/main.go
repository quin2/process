package main

import (
	"io/ioutil"
    "net/http"
    "net/url"
    "encoding/json"
    "strings"
    "math/rand"
    "time"

    "bytes"
	"context"

    "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type airRecords struct {
	Records []struct {
		Fields struct {
			Note string `json:"Note"`
			Category []string `json:"Category"`
		} `json:"fields"`
	} `json:"records"`
}

type Response events.APIGatewayProxyResponse

func Handler(ctx context.Context) (Response, error) {
	var buf bytes.Buffer

	airURL := process.env.AIRTABLE_URL

	req, err := http.NewRequest("GET", airURL, nil)			//HTTP(s) GET song & dance

	client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return Response{StatusCode: 404}, err
    }
    defer resp.Body.Close()

    body, readErr := ioutil.ReadAll(resp.Body)				//JSON parsing song & dance
	if readErr != nil {
		return Response{StatusCode: 404}, readErr
	}

	allLines := airRecords{}								//JSON marshaling now
	jsonErr := json.Unmarshal(body, &allLines)
	if jsonErr != nil {
		return Response{StatusCode: 404}, jsonErr
	}

	s1 := rand.NewSource(time.Now().UnixNano())				//random # seeding
	r1 := rand.New(s1)	

	i := r1.Intn(len(allLines.Records) + 1)
	toSend := allLines.Records[i].Fields.Category[0] + 
				": " + allLines.Records[i].Fields.Note

  	accountSid := process.env.TWILIO_SID
  	authToken := process.env.TWILIO_TOKEN

  	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + accountSid + "/Messages.json"	

  	numberTo := process.env.TO							//twilio source 
  	numberFrom := process.env.FROM

    msgData := url.Values{}									//stringify JSON message
	msgData.Set("To", numberTo)
	msgData.Set("From", numberFrom)
	msgData.Set("Body", toSend)
	msgDataReader := *strings.NewReader(msgData.Encode())

	twClient := &http.Client{}								//wrap data in request
	twReq, _ := http.NewRequest("POST", urlStr, &msgDataReader)
	twReq.SetBasicAuth(accountSid, authToken)
	twReq.Header.Add("Accept", "application/json")
	twReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")		

	_, twErr := twClient.Do(twReq)								//send request
	if twErr != nil {
	  return Response{StatusCode: 404}, twErr
	}

	lambdaResp := Response {								//cook up a fake response for AWS
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            buf.String(),
		Headers: map[string]string{
			"Content-Type":           "application/json",
			"X-MyCompany-Func-Reply": "hello-handler",
		},
	}

	return lambdaResp, nil	
}

func main() {
	lambda.Start(Handler)
}