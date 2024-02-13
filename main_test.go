package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
)

type mockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI
}

func (m *mockCloudWatchClient) PutMetricData(input *cloudwatch.PutMetricDataInput) (*cloudwatch.PutMetricDataOutput, error) {
	return &cloudwatch.PutMetricDataOutput{}, nil
}

func TestExportToCloudwatch(t *testing.T) {
	mockSvc := &mockCloudWatchClient{}

	mockPHPFPMStatus := PHPFPMStatus{
		ListenQueue:     10,
		ActiveProcesses: 15,
		SlowRequests:    20,
	}

	_, err := ExportToCloudwatch(mockSvc, mockPHPFPMStatus, "test-service")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

}

func TestGetContainerServiceName(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadataResponse := MetadataResponse{
			Cluster:     "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster",
			ServiceName: "mock-service",
			TaskARN:     "arn:aws:ecs:us-east-1:123456789012:task/test-cluster/6d0b6d6d-5f5d-4c6d-8f5d-6d0b6d6d5f5d",
		}
		json.NewEncoder(w).Encode(metadataResponse)
	}))
	defer mockServer.Close()

	os.Setenv("ECS_CONTAINER_METADATA_URI_V4", mockServer.URL)

	serviceName, err := GetContainerServiceName()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedServiceName := "mock-service"
	if serviceName != expectedServiceName {
		t.Errorf("Expected service name: %s, but got: %s", expectedServiceName, serviceName)
	}
}
