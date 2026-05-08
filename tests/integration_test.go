package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	baseURL = "http://localhost:8080"
)

func skipIfServicesNotAvailable(t *testing.T) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skip("Skipping integration test: API server not available. Run 'make run' first.")
	}
	resp.Body.Close()
}

type JobResponse struct {
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}

type StatusResponse struct {
	JobID      string `json:"job_id"`
	Status     string `json:"status"`
	RetryCount int    `json:"retry_count"`
}

type ResultResponse struct {
	Status string `json:"status"`
	Claims []struct {
		ClaimID  string  `json:"claim_id"`
		MemberID string  `json:"member_id"`
		Amount   float64 `json:"amount"`
	} `json:"claims"`
	Summary struct {
		TotalClaims int     `json:"total_claims"`
		TotalAmount float64 `json:"total_amount"`
	} `json:"summary"`
}

func TestIntegration_EndToEndJobProcessing(t *testing.T) {
	skipIfServicesNotAvailable(t)

	fileContent, err := os.ReadFile("fixtures/valid.edi")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.edi")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	if _, err := part.Write(fileContent); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	writer.Close()

	req, err := http.NewRequest("POST", baseURL+"/jobs", body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	var jobResp JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		t.Fatalf("Failed to decode job response: %v", err)
	}

	jobID := jobResp.JobID
	t.Logf("Created job with ID: %s", jobID)

	maxAttempts := 30
	var finalStatus string

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(1 * time.Second)

		resp, err := client.Get(baseURL + "/jobs/" + jobID)
		if err != nil {
			t.Fatalf("Failed to get job status: %v", err)
		}

		var statusResp StatusResponse
		if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
			resp.Body.Close()
			t.Fatalf("Failed to decode status response: %v", err)
		}
		resp.Body.Close()

		t.Logf("Job status (attempt %d): %s", i+1, statusResp.Status)
		finalStatus = statusResp.Status

		if statusResp.Status == "completed" || statusResp.Status == "failed" {
			break
		}
	}

	if finalStatus != "completed" {
		t.Fatalf("Job did not complete in time. Final status: %s", finalStatus)
	}

	resp, err = client.Get(baseURL + "/jobs/" + jobID + "/result")
	if err != nil {
		t.Fatalf("Failed to get job result: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	var resultResp ResultResponse
	if err := json.NewDecoder(resp.Body).Decode(&resultResp); err != nil {
		t.Fatalf("Failed to decode result response: %v", err)
	}

	if resultResp.Summary.TotalClaims != 5 {
		t.Errorf("Expected 5 claims, got %d", resultResp.Summary.TotalClaims)
	}

	expectedAmount := 13000.0
	if resultResp.Summary.TotalAmount != expectedAmount {
		t.Errorf("Expected total amount %.2f, got %.2f", expectedAmount, resultResp.Summary.TotalAmount)
	}

	if len(resultResp.Claims) != 5 {
		t.Errorf("Expected 5 claim records, got %d", len(resultResp.Claims))
	}

	t.Log("Integration test passed: End-to-end job processing successful")
}

func TestIntegration_InvalidEDIFile(t *testing.T) {
	skipIfServicesNotAvailable(t)

	fileContent, err := os.ReadFile("fixtures/invalid.edi")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "invalid.edi")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	if _, err := part.Write(fileContent); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	writer.Close()

	req, err := http.NewRequest("POST", baseURL+"/jobs", body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	var jobResp JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		t.Fatalf("Failed to decode job response: %v", err)
	}

	jobID := jobResp.JobID
	t.Logf("Created job with ID: %s", jobID)

	maxAttempts := 60
	var finalStatus string

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(1 * time.Second)

		resp, err := client.Get(baseURL + "/jobs/" + jobID)
		if err != nil {
			t.Fatalf("Failed to get job status: %v", err)
		}

		var statusResp StatusResponse
		if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
			resp.Body.Close()
			t.Fatalf("Failed to decode status response: %v", err)
		}
		resp.Body.Close()

		t.Logf("Job status (attempt %d): %s, retry count: %d", i+1, statusResp.Status, statusResp.RetryCount)
		finalStatus = statusResp.Status

		if statusResp.Status == "completed" || statusResp.Status == "failed" {
			break
		}
	}

	if finalStatus != "failed" {
		t.Errorf("Expected job to fail for invalid EDI, got status: %s", finalStatus)
	}

	t.Log("Integration test passed: Invalid EDI file correctly marked as failed")
}

func TestIntegration_HealthCheck(t *testing.T) {
	skipIfServicesNotAvailable(t)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var healthResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if healthResp["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", healthResp["status"])
	}

	t.Log("Health check passed")
}

func TestIntegration_ConcurrentJobSubmissions(t *testing.T) {
	skipIfServicesNotAvailable(t)

	fileContent, err := os.ReadFile("fixtures/valid.edi")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	numJobs := 5
	jobIDs := make(chan string, numJobs)
	errors := make(chan error, numJobs)

	for i := 0; i < numJobs; i++ {
		go func(index int) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			part, err := writer.CreateFormFile("file", "concurrent_test.edi")
			if err != nil {
				errors <- err
				return
			}

			if _, err := part.Write(fileContent); err != nil {
				errors <- err
				return
			}

			writer.Close()

			req, err := http.NewRequest("POST", baseURL+"/jobs", body)
			if err != nil {
				errors <- err
				return
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				errors <- err
				return
			}

			var jobResp JobResponse
			if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
				errors <- err
				return
			}

			jobIDs <- jobResp.JobID
		}(i)
	}

	var successfulJobs []string
	for i := 0; i < numJobs; i++ {
		select {
		case jobID := <-jobIDs:
			successfulJobs = append(successfulJobs, jobID)
		case err := <-errors:
			t.Errorf("Error submitting concurrent job: %v", err)
		case <-time.After(15 * time.Second):
			t.Error("Timeout waiting for concurrent job submission")
		}
	}

	if len(successfulJobs) != numJobs {
		t.Errorf("Expected %d successful job submissions, got %d", numJobs, len(successfulJobs))
	}

	t.Logf("Successfully submitted %d concurrent jobs", len(successfulJobs))
}

func TestIntegration_JobNotFound(t *testing.T) {
	skipIfServicesNotAvailable(t)

	fakeJobID := "00000000-0000-0000-0000-000000000000"

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/jobs/" + fakeJobID)
	if err != nil {
		t.Fatalf("Failed to call API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	t.Log("Job not found test passed")
}

func TestMain(m *testing.M) {
	code := m.Run()

	os.Exit(code)
}
