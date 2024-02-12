package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	fcgiclient "github.com/tomasen/fcgi_client"
)

// MetadataResponse represents the JSON response from the ECS metadata endpoint.
// See https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4-fargate.html for more information.
type MetadataResponse struct {
	Cluster     string `json:"Cluster"`
	ServiceName string `json:"ServiceName"`
	TaskARN     string `json:"TaskARN"`
}

// PHPFPMStatus represents the JSON response from PHP-FPM status endpoint.
// See https://www.php.net/manual/en/fpm.status.php for more information.
type PHPFPMStatus struct {
	ListenQueue     int64 `json:"listen queue"`
	ActiveProcesses int64 `json:"active processes"`
	SlowRequests    int64 `json:"slow requests"`
}

// GetContainerServiceName retrieves the name of the container service.
// It makes an HTTP GET request to the ECS_CONTAINER_METADATA_URI_V4 environment variable
// and parses the response to extract the service name.
// If successful, it returns the service name and nil error.
// Otherwise, it returns an empty string and the encountered error.
func GetContainerServiceName() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/task", os.Getenv("ECS_CONTAINER_METADATA_URI_V4")))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var metadataResponse MetadataResponse

	body, err := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(body, &metadataResponse)

	if err != nil {
		return "", err
	}

	return metadataResponse.ServiceName, nil
}

// ExportToCloudwatch exports PHP-FPM metrics to CloudWatch.
// It takes a CloudWatch service client, PHPFPMStatus struct, and service name as input.
// It creates MetricDatum objects for ListenQueue, ActiveProcesses, and SlowRequests metrics,
// and sends them to CloudWatch using the PutMetricData API.
// It returns the PutMetricDataOutput and any error encountered.
func ExportToCloudwatch(svc cloudwatchiface.CloudWatchAPI, phpfpmstatus PHPFPMStatus, servicename string) (*cloudwatch.PutMetricDataOutput, error) {
	PutMetricDataOutput, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String("Monitoring/PHP-FPM"),
		MetricData: []*cloudwatch.MetricDatum{
			&cloudwatch.MetricDatum{
				MetricName: aws.String("ListenQueue"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(phpfpmstatus.ListenQueue)),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("ServiceName"),
						Value: aws.String(servicename),
					},
				},
			},
			&cloudwatch.MetricDatum{
				MetricName: aws.String("ActiveProcesses"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(phpfpmstatus.ActiveProcesses)),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("ServiceName"),
						Value: aws.String(servicename),
					},
				},
			},
			&cloudwatch.MetricDatum{
				MetricName: aws.String("SlowRequests"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(phpfpmstatus.SlowRequests)),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("ServiceName"),
						Value: aws.String(servicename),
					},
				},
			},
		},
	})

	if err != nil {
		fmt.Println("Error adding metrics:", err.Error())
		return PutMetricDataOutput, err
	}

	return PutMetricDataOutput, nil
}

// GetPHPFPMStatus retrieves the PHP-FPM status by making a request to the /status endpoint.
// It returns a PHPFPMStatus struct containing the parsed response and an error if any.
func GetPHPFPMStatus() (PHPFPMStatus, error) {

	port := os.Getenv("PHP_FPM_PORT")
	if port == "" {
		port = "9000"
	}

	host := os.Getenv("PHP_FPM_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	env := make(map[string]string)
	env["SCRIPT_FILENAME"] = "/status"
	env["SCRIPT_NAME"] = "/status"
	env["SERVER_SOFTWARE"] = "go / fcgiclient"
	env["REMOTE_ADDR"] = "127.0.0.1"
	env["QUERY_STRING"] = "json&full"

	fcgi, err := fcgiclient.Dial("tcp", fmt.Sprintf("%s:%s", host, port))

	if err != nil {
		log.Println("err:", err)
	}

	resp, err := fcgi.Get(env)
	if err != nil {
		log.Println("err:", err)
	}

	content, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println("err:", err)
	}

	// Unmarshal the JSON response into a PHPFPMStatus struct
	var stats PHPFPMStatus
	err = json.Unmarshal(content, &stats)
	if err != nil {
		return PHPFPMStatus{}, err
	}

	return stats, nil
}

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := cloudwatch.New(sess)

	var svc_name string
	var err error

	for {
		if os.Getenv("APPLICATION_NAME") == "" {
			svc_name, err = GetContainerServiceName()
			if err != nil {
				fmt.Println("Error getting service name:", err.Error())
				return
			}
		} else {
			svc_name = os.Getenv("APPLICATION_NAME")
		}

		stats, err := GetPHPFPMStatus()
		if err != nil {
			fmt.Println("Error getting PHP-FPM status:", err.Error())
			return
		}

		ExportToCloudwatch(svc, stats, svc_name)

		time.Sleep(1 * time.Minute)
	}
}
