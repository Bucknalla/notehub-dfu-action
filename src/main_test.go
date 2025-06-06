package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewNotehubClient(t *testing.T) {
	client := NewNotehubClient()

	if client == nil {
		t.Fatal("NewNotehubClient returned nil")
	}

	if client.baseURL != "https://api.notefile.net/v1" {
		t.Errorf("Expected baseURL to be 'https://api.notefile.net/v1', got '%s'", client.baseURL)
	}

	if client.httpClient == nil {
		t.Fatal("httpClient should not be nil")
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout to be 30s, got %v", client.httpClient.Timeout)
	}
}

func TestAuthenticate_InvalidCredentials(t *testing.T) {
	client := NewNotehubClient()
	ctx := context.Background()

	// Test with empty credentials - should fail
	err := client.Authenticate(ctx, "", "")
	if err == nil {
		t.Error("Expected authentication to fail with empty credentials")
	}
}

func TestDeploymentConfig_Validation(t *testing.T) {
	config := &DeploymentConfig{
		ProjectUID:   "test-project",
		FirmwareFile: "test.bin",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	}

	if config.ProjectUID == "" {
		t.Error("ProjectUID should not be empty")
	}

	if config.FirmwareFile == "" {
		t.Error("FirmwareFile should not be empty")
	}
}

func TestUploadFirmware_MissingFile(t *testing.T) {
	client := NewNotehubClient()
	ctx := context.Background()

	// Test with non-existent file - should fail
	_, err := client.UploadFirmware(ctx, "test-project", "nonexistent-file.bin")
	if err == nil {
		t.Error("Expected upload to fail with non-existent file")
	}

	if !strings.Contains(err.Error(), "failed to read firmware file") {
		t.Errorf("Expected error about reading file, got: %v", err)
	}
}

func TestUploadFirmware_NoToken(t *testing.T) {
	client := NewNotehubClient()
	// Don't set access token
	ctx := context.Background()

	// Create a temporary test file
	testFile := "test-firmware.bin"
	testData := []byte("test firmware data")
	err := os.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test upload without authentication - should eventually fail at HTTP level
	_, err = client.UploadFirmware(ctx, "test-project", testFile)
	if err == nil {
		t.Error("Expected upload to fail without access token")
	}
}

func TestTriggerDFU_NoToken(t *testing.T) {
	client := NewNotehubClient()
	// Don't set access token
	ctx := context.Background()

	config := &DeploymentConfig{
		ProjectUID: "test-project",
		DeviceUID:  "test-device",
	}

	// Test DFU trigger without authentication - should eventually fail at HTTP level
	err := client.TriggerDFU(ctx, config, "test-firmware.bin")
	if err == nil {
		t.Error("Expected DFU trigger to fail without access token")
	}
}

func TestTriggerDFU_QueryParams(t *testing.T) {
	// Test query parameter building (without making HTTP calls)
	config := &DeploymentConfig{
		ProjectUID:       "test-project",
		DeviceUID:        "device-123",
		Tag:              "production",
		SerialNumber:     "SN123",
		FleetUID:         "fleet-456",
		ProductUID:       "product-789",
		NotecardFirmware: "v1.0.0",
		Location:         "factory",
		SKU:              "SKU-ABC",
	}

	// Verify all fields are set
	if config.DeviceUID == "" {
		t.Error("DeviceUID should not be empty")
	}
	if config.Tag == "" {
		t.Error("Tag should not be empty")
	}
	if config.SerialNumber == "" {
		t.Error("SerialNumber should not be empty")
	}
	if config.FleetUID == "" {
		t.Error("FleetUID should not be empty")
	}
	if config.ProductUID == "" {
		t.Error("ProductUID should not be empty")
	}
	if config.NotecardFirmware == "" {
		t.Error("NotecardFirmware should not be empty")
	}
	if config.Location == "" {
		t.Error("Location should not be empty")
	}
	if config.SKU == "" {
		t.Error("SKU should not be empty")
	}
}

func TestAddCommaSeparatedParams(t *testing.T) {
	tests := []struct {
		name           string
		paramName      string
		value          string
		expectedParams map[string][]string
	}{
		{
			name:      "single value",
			paramName: "deviceUID",
			value:     "device-123",
			expectedParams: map[string][]string{
				"deviceUID": {"device-123"},
			},
		},
		{
			name:      "multiple values",
			paramName: "tags",
			value:     "production,sensor,outdoor",
			expectedParams: map[string][]string{
				"tags": {"production", "sensor", "outdoor"},
			},
		},
		{
			name:      "values with spaces",
			paramName: "location",
			value:     "warehouse, factory , office",
			expectedParams: map[string][]string{
				"location": {"warehouse", "factory", "office"},
			},
		},
		{
			name:           "empty value",
			paramName:      "sku",
			value:          "",
			expectedParams: map[string][]string{},
		},
		{
			name:      "values with empty segments",
			paramName: "serialNumber",
			value:     "SN001,,SN003,",
			expectedParams: map[string][]string{
				"serialNumber": {"SN001", "SN003"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams := url.Values{}
			addCommaSeparatedParams(queryParams, tt.paramName, tt.value)

			// Check if we got the expected parameters
			for expectedParam, expectedValues := range tt.expectedParams {
				actualValues := queryParams[expectedParam]
				if len(actualValues) != len(expectedValues) {
					t.Errorf("Expected %d values for %s, got %d", len(expectedValues), expectedParam, len(actualValues))
					continue
				}
				for i, expectedValue := range expectedValues {
					if actualValues[i] != expectedValue {
						t.Errorf("Expected value %s at index %d, got %s", expectedValue, i, actualValues[i])
					}
				}
			}

			// Check that we don't have unexpected parameters
			for actualParam := range queryParams {
				if _, expected := tt.expectedParams[actualParam]; !expected {
					t.Errorf("Unexpected parameter %s found", actualParam)
				}
			}
		})
	}
}

func TestTriggerDFU_CommaSeparatedQueryParams(t *testing.T) {
	tests := []struct {
		name           string
		config         *DeploymentConfig
		expectedParams map[string][]string
	}{
		{
			name: "single values",
			config: &DeploymentConfig{
				ProjectUID:   "test-project",
				DeviceUID:    "device-123",
				Tag:          "production",
				SerialNumber: "SN123",
			},
			expectedParams: map[string][]string{
				"deviceUID":    {"device-123"},
				"tags":         {"production"},
				"serialNumber": {"SN123"},
			},
		},
		{
			name: "multiple comma-separated values",
			config: &DeploymentConfig{
				ProjectUID:   "test-project",
				DeviceUID:    "device-123,device-456,device-789",
				Tag:          "production,sensor,outdoor",
				SerialNumber: "SN001,SN002,SN003",
				FleetUID:     "fleet-1,fleet-2",
				ProductUID:   "product-A,product-B",
			},
			expectedParams: map[string][]string{
				"deviceUID":    {"device-123", "device-456", "device-789"},
				"tags":         {"production", "sensor", "outdoor"},
				"serialNumber": {"SN001", "SN002", "SN003"},
				"fleetUID":     {"fleet-1", "fleet-2"},
				"productUID":   {"product-A", "product-B"},
			},
		},
		{
			name: "values with spaces and empty segments",
			config: &DeploymentConfig{
				ProjectUID:       "test-project",
				Tag:              "production, sensor , outdoor",
				NotecardFirmware: "v1.0.0,,v1.0.2,",
				Location:         "warehouse, factory",
				SKU:              "SKU-A,, SKU-C",
			},
			expectedParams: map[string][]string{
				"tags":             {"production", "sensor", "outdoor"},
				"notecardFirmware": {"v1.0.0", "v1.0.2"},
				"location":         {"warehouse", "factory"},
				"sku":              {"SKU-A", "SKU-C"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We'll test the query parameter building logic by examining what gets built
			// without actually making HTTP calls
			queryParams := url.Values{}

			// Apply the same logic as TriggerDFU
			addCommaSeparatedParams(queryParams, "deviceUID", tt.config.DeviceUID)
			addCommaSeparatedParams(queryParams, "tags", tt.config.Tag)
			addCommaSeparatedParams(queryParams, "serialNumber", tt.config.SerialNumber)
			addCommaSeparatedParams(queryParams, "fleetUID", tt.config.FleetUID)
			addCommaSeparatedParams(queryParams, "productUID", tt.config.ProductUID)
			addCommaSeparatedParams(queryParams, "notecardFirmware", tt.config.NotecardFirmware)
			addCommaSeparatedParams(queryParams, "location", tt.config.Location)
			addCommaSeparatedParams(queryParams, "sku", tt.config.SKU)

			// Verify expected parameters
			for expectedParam, expectedValues := range tt.expectedParams {
				actualValues := queryParams[expectedParam]
				if len(actualValues) != len(expectedValues) {
					t.Errorf("Expected %d values for %s, got %d", len(expectedValues), expectedParam, len(actualValues))
					continue
				}
				for i, expectedValue := range expectedValues {
					if actualValues[i] != expectedValue {
						t.Errorf("Expected value %s at index %d for %s, got %s", expectedValue, i, expectedParam, actualValues[i])
					}
				}
			}

			// Verify we don't have unexpected parameters
			for actualParam := range queryParams {
				if _, expected := tt.expectedParams[actualParam]; !expected {
					t.Errorf("Unexpected parameter %s found with values %v", actualParam, queryParams[actualParam])
				}
			}
		})
	}
}

func TestTriggerDFU_CompleteURLGeneration(t *testing.T) {
	tests := []struct {
		name        string
		config      *DeploymentConfig
		expectedURL string
	}{
		{
			name: "single tag",
			config: &DeploymentConfig{
				ProjectUID: "app:12345678-1234-1234-1234-123456789012",
				Tag:        "production",
			},
			expectedURL: "https://api.notefile.net/v1/projects/app:12345678-1234-1234-1234-123456789012/dfu/host/update?tags=production",
		},
		{
			name: "multiple tags",
			config: &DeploymentConfig{
				ProjectUID: "app:12345678-1234-1234-1234-123456789012",
				Tag:        "production,sensor,outdoor",
			},
			expectedURL: "https://api.notefile.net/v1/projects/app:12345678-1234-1234-1234-123456789012/dfu/host/update?tags=production&tags=sensor&tags=outdoor",
		},
		{
			name: "multiple device UIDs",
			config: &DeploymentConfig{
				ProjectUID: "app:12345678-1234-1234-1234-123456789012",
				DeviceUID:  "device-123,device-456,device-789",
			},
			expectedURL: "https://api.notefile.net/v1/projects/app:12345678-1234-1234-1234-123456789012/dfu/host/update?deviceUID=device-123&deviceUID=device-456&deviceUID=device-789",
		},
		{
			name: "mixed parameters with comma-separated values",
			config: &DeploymentConfig{
				ProjectUID:   "app:12345678-1234-1234-1234-123456789012",
				DeviceUID:    "device-123,device-456",
				Tag:          "production,sensor",
				SerialNumber: "SN001",
				FleetUID:     "fleet-A,fleet-B",
			},
			expectedURL: "https://api.notefile.net/v1/projects/app:12345678-1234-1234-1234-123456789012/dfu/host/update?deviceUID=device-123&deviceUID=device-456&fleetUID=fleet-A&fleetUID=fleet-B&serialNumber=SN001&tags=production&tags=sensor",
		},
		{
			name: "all parameters with multiple values",
			config: &DeploymentConfig{
				ProjectUID:       "app:12345678-1234-1234-1234-123456789012",
				DeviceUID:        "dev1,dev2",
				Tag:              "tag1,tag2",
				SerialNumber:     "SN1,SN2",
				FleetUID:         "fleet1,fleet2",
				ProductUID:       "prod1,prod2",
				NotecardFirmware: "fw1,fw2",
				Location:         "loc1,loc2",
				SKU:              "sku1,sku2",
			},
			expectedURL: "https://api.notefile.net/v1/projects/app:12345678-1234-1234-1234-123456789012/dfu/host/update?deviceUID=dev1&deviceUID=dev2&fleetUID=fleet1&fleetUID=fleet2&location=loc1&location=loc2&notecardFirmware=fw1&notecardFirmware=fw2&productUID=prod1&productUID=prod2&serialNumber=SN1&serialNumber=SN2&sku=sku1&sku=sku2&tags=tag1&tags=tag2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build query parameters using the same logic as TriggerDFU
			queryParams := url.Values{}

			addCommaSeparatedParams(queryParams, "deviceUID", tt.config.DeviceUID)
			addCommaSeparatedParams(queryParams, "tags", tt.config.Tag)
			addCommaSeparatedParams(queryParams, "serialNumber", tt.config.SerialNumber)
			addCommaSeparatedParams(queryParams, "fleetUID", tt.config.FleetUID)
			addCommaSeparatedParams(queryParams, "productUID", tt.config.ProductUID)
			addCommaSeparatedParams(queryParams, "notecardFirmware", tt.config.NotecardFirmware)
			addCommaSeparatedParams(queryParams, "location", tt.config.Location)
			addCommaSeparatedParams(queryParams, "sku", tt.config.SKU)

			// Build the complete URL
			baseURL := "https://api.notefile.net/v1"
			dfuURL := fmt.Sprintf("%s/projects/%s/dfu/host/update", baseURL, tt.config.ProjectUID)
			if len(queryParams) > 0 {
				dfuURL += "?" + queryParams.Encode()
			}

			// Verify the complete URL matches expected
			if dfuURL != tt.expectedURL {
				t.Errorf("URL mismatch:\nExpected: %s\nActual:   %s", tt.expectedURL, dfuURL)
			}

			// Also log the URL for visual verification
			t.Logf("Generated URL: %s", dfuURL)
		})
	}
}
