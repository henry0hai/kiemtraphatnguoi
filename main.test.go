package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var captchaURL = "https://www.csgt.vn/lib/captcha/captcha.class.php"

func TestFetchCSGTCaptcha(t *testing.T) {
	// Call fetchCSGTCaptcha twice
	_, jar1, err1 := fetchCSGTCaptcha()
	if err1 != nil {
		t.Fatalf("First call to fetchCSGTCaptcha failed: %v", err1)
	}

	_, jar2, err2 := fetchCSGTCaptcha()
	if err2 != nil {
		t.Fatalf("Second call to fetchCSGTCaptcha failed: %v", err2)
	}

	// Check that the two cookie jars are different instances
	if jar1 == jar2 {
		t.Error("Expected different cookie jars for each request, but got the same instance")
	}
}

func TestFetchCSGTCaptcha_NetworkTimeout(t *testing.T) {
	// Create a test server that simulates a timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than the client timeout
	}))
	defer server.Close()

	// Replace the captchaURL with the test server URL
	originalCaptchaURL := captchaURL
	captchaURL = server.URL
	defer func() { captchaURL = originalCaptchaURL }()

	// Set a short timeout for the test
	originalTimeout := time.Duration(30 * time.Second)
	http.DefaultClient.Timeout = 1 * time.Second
	defer func() { http.DefaultClient.Timeout = originalTimeout }()

	// Call the function
	_, _, err := fetchCSGTCaptcha()

	// Check if the error is a timeout error
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("Expected timeout error, got: %v", err)
	}
}

func TestFetchCSGTCaptcha_NonOKStatus(t *testing.T) {
	// Create a test server that returns a non-200 status code
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Replace the captchaURL with the test server URL
	originalCaptchaURL := captchaURL
	captchaURL = server.URL
	defer func() { captchaURL = originalCaptchaURL }()

	// Call the function
	_, _, err := fetchCSGTCaptcha()

	// Check if the error message contains the correct status code
	if err == nil {
		t.Fatal("Expected error for non-200 status code, got nil")
	}
	expectedErrMsg := "captcha endpoint returned status code: 500"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Fatalf("Expected error message to contain %q, got: %v", expectedErrMsg, err)
	}
}

func TestFetchCSGTCaptcha_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a captcha image
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("mock captcha image"))
	}))
	defer server.Close()

	// Replace the captchaURL with the mock server URL
	originalCaptchaURL := captchaURL
	captchaURL = server.URL
	defer func() { captchaURL = originalCaptchaURL }()

	// Call the function
	imgBytes, jar, err := fetchCSGTCaptcha()

	// Check for errors
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check if imgBytes is not empty
	if len(imgBytes) == 0 {
		t.Error("Expected non-empty image bytes, got empty slice")
	}

	// Check if the content of imgBytes is correct
	if string(imgBytes) != "mock captcha image" {
		t.Errorf("Expected image content 'mock captcha image', got: %s", string(imgBytes))
	}

	// Check if jar is not nil
	if jar == nil {
		t.Error("Expected non-nil cookie jar, got nil")
	}
}
