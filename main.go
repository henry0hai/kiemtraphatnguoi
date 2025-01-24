package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"net/http/cookiejar"

	// For OCR (example library):
	"github.com/PuerkitoBio/goquery"
	"github.com/otiai10/gosseract/v2"
)

var (
	ErrDataNotFound = errors.New("data not found")
)

type CsgtData struct {
	Plate           string `json:"plate"`
	PlateColor      string `json:"plate_color"`
	VehicleType     string `json:"vehicle_type"`
	ViolationTime   string `json:"violation_time"`
	ViolationPlace  string `json:"violation_place"`
	ViolationAction string `json:"violation_action"`
	Status          string `json:"status"`
	DetectedBy      string `json:"detected_by"`
	// For "Nơi giải quyết vụ việc," we might have multiple lines. You can
	// store them as a single string or break them out further.
	ResolutionLocation string `json:"resolution_location"`
}

func main() {
	http.HandleFunc("/checkplate", checkPlateHandler)
	http.HandleFunc("/checkplate-csgt", checkPlateCSGTHandler)

	fmt.Println("Starting server on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func checkPlateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed. Use POST.")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to parse form data: "+err.Error())
		return
	}

	plate := r.FormValue("bienso")
	if plate == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing or empty parameter: bienso")
		return
	}

	// Get `loaixe` parameter, default to "oto" (1)
	vehicleType := r.FormValue("loaixe")
	if vehicleType == "" {
		vehicleType = "oto" // Default to "oto"
	}

	// Map vehicle type to appropriate code ("1" for oto, "2" for xemay)
	vehicleCode := "1" // Default to "oto"
	switch strings.ToLower(vehicleType) {
	case "xemay":
		vehicleCode = "2"
	case "oto":
		vehicleCode = "1"
	default:
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Invalid vehicle type: %s", vehicleType))
		return
	}

	// Clean & validate plate
	plate, err := processPlate(plate)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 1) Check primary source
	data, err := fetchDataPhatNguoi(plate)
	if err != nil {
		if errors.Is(err, ErrDataNotFound) {
			// If no data in primary => fallback
			log.Printf("No data for plate %s from primary API. Attempting fallback to csgt.vn...\n", plate)

			// 2) Fallback: Solve captcha from csgt.vn, then fetch data
			fallbackData, fallbackErr := fallbackToCSGTWithVehicleCode(plate, vehicleCode)
			if fallbackErr != nil {
				log.Printf("Fallback failed: %v\n", fallbackErr)
				writeJSONError(w, http.StatusNotFound, "No data found on primary, fallback also failed: "+fallbackErr.Error())
				return
			}

			// Return fallback data
			writeJSON(w, http.StatusOK, fallbackData)
			return
		}
		// Some other error from primary
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// If found data from primary
	writeJSON(w, http.StatusOK, data)
}

func fallbackToCSGTWithVehicleCode(plate, vehicleCode string) (interface{}, error) {
	// 1) Fetch the captcha image
	imgBytes, cookieJar, err := fetchCSGTCaptcha()
	if err != nil {
		return nil, fmt.Errorf("fetch captcha failed: %w", err)
	}

	// 2) Solve the captcha (OCR)
	captchaText, err := solveCaptchaWithOCR(imgBytes)
	if err != nil {
		return nil, fmt.Errorf("captcha OCR failed: %w", err)
	}

	// Debug log: log recognized text
	log.Printf("Recognized captcha text = %q\n", captchaText)

	// 3) Use vehicle code and captcha to fetch data
	data, err := fetchDataCSGTWithSession(plate, vehicleCode, captchaText, cookieJar)

	log.Printf("data: %v\n", data)

	if err != nil {
		return nil, err
	}

	return data, nil
}

// fallbackToCSGT demonstrates an automatic fallback check to csgt.vn
func fallbackToCSGT(plate string) (interface{}, error) {
	// 1) Fetch the captcha image
	imgBytes, cookieJar, err := fetchCSGTCaptcha()
	if err != nil {
		return nil, fmt.Errorf("fetch captcha failed: %w", err)
	}

	// 2) Solve the captcha (OCR)
	captchaText, err := solveCaptchaWithOCR(imgBytes)
	if err != nil {
		return nil, fmt.Errorf("captcha OCR failed: %w", err)
	}

	// Debug log: log recognized text
	log.Printf("Recognized captcha text = %q\n", captchaText)

	// 3) Now we have the recognized text; attempt csgt.vn data fetch.
	// Typically csgt.vn wants "Xe" param => "1" (ô tô), "2" (xe máy), ...
	// For demonstration, let's just use "1".
	data, err := fetchDataCSGTWithSession(plate, "1", captchaText, cookieJar)

	log.Printf("data: %v\n", data)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func processPlate(rawPlate string) (string, error) {
	rawPlate = strings.TrimSpace(rawPlate)
	rawPlate = strings.ToUpper(rawPlate)
	replacer := strings.NewReplacer("-", "", ".", "", " ", "")
	cleaned := replacer.Replace(rawPlate)

	if cleaned == "" {
		return "", errors.New("please provide a valid plate number")
	}
	// Example: 51K12345 / 51K123456
	matched, _ := regexp.MatchString(`^\d{2}[A-Z]\d{5,6}$`, cleaned)
	if !matched {
		return "", errors.New("invalid plate number format (expected e.g. 51K12345)")
	}
	return cleaned, nil
}

func fetchDataPhatNguoi(bienso string) (interface{}, error) {
	url := "https://api.checkphatnguoi.vn/phatnguoi"
	formData := "bienso=" + bienso

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(formData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 1) Unmarshal to the wrapper
	var w primaryWrapper
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, fmt.Errorf("could not parse JSON from primary: %w", err)
	}

	// 2) If the wrapper "data" is empty => no records => return ErrDataNotFound
	if len(w.Data) == 0 {
		return nil, ErrDataNotFound
	}

	// 3) Convert each item in w.Data to your CsgtData
	results := make([]*CsgtData, 0, len(w.Data))
	for _, item := range w.Data {
		results = append(results, parsePrimaryRecord(item))
	}

	// 4) Return the array of CsgtData
	return results, nil
}

type primaryWrapper struct {
	Status int                      `json:"status"`
	Data   []map[string]interface{} `json:"data"`
}

func parsePrimaryDataWrapper(body []byte) ([]*CsgtData, error) {
	var w primaryWrapper
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}

	var results []*CsgtData
	for _, item := range w.Data {
		results = append(results, parsePrimaryRecord(item))
	}
	return results, nil
}

func parsePrimaryDataArray(body []byte) ([]*CsgtData, error) {
	// The body is something like:
	// [
	//   {
	//     "Biển kiểm soát": "...",
	//     ...
	//   }
	// ]

	var arr []map[string]interface{}
	if err := json.Unmarshal(body, &arr); err != nil {
		return nil, err
	}

	result := make([]*CsgtData, 0, len(arr))
	for _, item := range arr {
		result = append(result, parsePrimaryRecord(item))
	}
	return result, nil
}

func parsePrimaryRecord(item map[string]interface{}) *CsgtData {
	data := &CsgtData{}

	// Safely get a string from an interface{}
	getString := func(k string) string {
		v, ok := item[k]
		if !ok || v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}

	data.Plate = getString("Biển kiểm soát")
	data.PlateColor = getString("Màu biển")
	data.VehicleType = getString("Loại phương tiện")
	data.ViolationTime = getString("Thời gian vi phạm")
	data.ViolationPlace = getString("Địa điểm vi phạm")
	data.ViolationAction = getString("Hành vi vi phạm")
	data.Status = getString("Trạng thái")
	data.DetectedBy = getString("Đơn vị phát hiện vi phạm")

	// For "Nơi giải quyết vụ việc" (array of strings), join with newlines:
	if arrAny, ok := item["Nơi giải quyết vụ việc"]; ok && arrAny != nil {
		if arr, ok := arrAny.([]interface{}); ok {
			var lines []string
			for _, lineAny := range arr {
				lines = append(lines, fmt.Sprintf("%v", lineAny))
			}
			data.ResolutionLocation = strings.Join(lines, "\n")
		}
	}

	return data
}

// ------------------------------------------------------------------------
// Secondary / Fallback: csgt.vn
// ------------------------------------------------------------------------

// checkPlateCSGTHandler is the existing handler that requires manual captcha input.
// For the automatic fallback approach, we won't call this directly from the client,
// but the logic is the same.
func checkPlateCSGTHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed. Use POST.")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to parse form data: "+err.Error())
		return
	}

	plate := r.FormValue("bienso")
	vehicleType := r.FormValue("vehicle_type")
	captcha := r.FormValue("captcha")

	if plate == "" || vehicleType == "" || captcha == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing bienso, vehicle_type, or captcha")
		return
	}

	data, err := fetchDataCSGT(plate, vehicleType, captcha)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, data)
}

// fetchDataCSGT is the direct approach.
// For an automatic fallback, we might need to re-use a cookie jar, etc.
func fetchDataCSGT(plate, vehicleType, captcha string) (interface{}, error) {
	return fetchDataCSGTWithSession(plate, vehicleType, captcha, nil)
}

// fetchDataCSGTWithSession is the same but allows us to carry cookies from captcha request if needed.
func fetchDataCSGTWithSession(plate, vehicleType, captcha string, cookieJar http.CookieJar) (interface{}, error) {
	url := "https://www.csgt.vn/?mod=contact&task=tracuu_post&ajax"
	formData := fmt.Sprintf("BienKS=%s&Xe=%s&captcha=%s&ipClient=9.9.9.91&cUrl=", plate, vehicleType, captcha)

	client := &http.Client{Timeout: 30 * time.Second}
	if cookieJar != nil {
		client.Jar = cookieJar
	}

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(formData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Go-http-client/1.1)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	content := strings.TrimSpace(string(body))
	log.Printf("Raw CSGT response: %s\n", content)

	if content == "404" {
		return nil, errors.New("csgt.vn: captcha incorrect or request rejected (404)")
	}

	// // Attempt to parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	log.Printf("CSGT response (JSON): %v\n", result)

	// See if there's an href we can follow for the actual data:
	hrefVal, ok := result["href"].(string)
	if ok && hrefVal != "" {
		// 1) Fetch that HTML page
		htmlContent, err := fetchCSGTHtml(hrefVal, client)

		// log.Println("CSGT full HTML:\n", htmlContent)

		if err != nil {
			return nil, fmt.Errorf("failed to fetch csgt HTML at %q: %w", hrefVal, err)
		}

		// 2) Parse the HTML
		parsedData, err := parseCSGTHtml(htmlContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse csgt HTML from %q: %w", hrefVal, err)
		}

		log.Println("parseCSGTHtml HTML:\n", parsedData)

		// 3) Return the structured data
		return parsedData, nil
	}

	// Otherwise, just return what we got
	return result, nil
}

func fetchCSGTHtml(url string, client *http.Client) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("GET %q failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("csgt.vn returned non-200 status: %d", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body: %w", err)
	}

	return string(bytes), nil
}

// ------------------------------------------------------------------------
// Captcha retrieval + OCR
// ------------------------------------------------------------------------

// fetchCSGTCaptcha retrieves the captcha image and saves it locally for debugging.
func fetchCSGTCaptcha() ([]byte, http.CookieJar, error) {
	captchaURL := "https://www.csgt.vn/lib/captcha/captcha.class.php"

	// Create a new cookie jar explicitly
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     cookieJar,
	}

	req, err := http.NewRequest("GET", captchaURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create captcha request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch captcha image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("captcha endpoint returned status code: %d", resp.StatusCode)
	}

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read captcha image: %w", err)
	}

	// Save the captcha image to the folder
	if saveErr := saveCaptchaImage(imgBytes); saveErr != nil {
		log.Printf("Failed to save captcha image: %v\n", saveErr)
	}

	return imgBytes, cookieJar, nil
}

// saveCaptchaImage saves the captcha image to the "captchaImageLogs" folder with a timestamped filename.
func saveCaptchaImage(imgBytes []byte) error {
	// Create the directory if it doesn't exist
	dir := "captchaImageLogs"
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	// Generate a timestamped filename
	timestamp := time.Now().Format("2006-01-02_15-04-05") // Example: 2025-01-24_10-30-15
	filename := filepath.Join(dir, fmt.Sprintf("%s-captcha.png", timestamp))

	// Write the image bytes to the file
	if err := os.WriteFile(filename, imgBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", filename, err)
	}

	log.Printf("Captcha image saved to %s\n", filename)
	return nil
}

// solveCaptchaWithOCR uses an OCR library to decode the captcha text.
// We'll use github.com/otiai10/gosseract/v2 as an example.
func solveCaptchaWithOCR(imgBytes []byte) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()

	// Feed the raw image bytes to the OCR engine
	if err := client.SetImageFromBytes(imgBytes); err != nil {
		return "", fmt.Errorf("failed to set image bytes to OCR: %w", err)
	}

	// Optionally configure some OCR settings, e.g.:
	//   client.SetLanguage("eng")
	//   client.SetWhitelist("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	text, err := client.Text()
	if err != nil {
		return "", fmt.Errorf("failed to recognize text via OCR: %w", err)
	}

	text = strings.TrimSpace(text)
	return text, nil
}

// ------------------------------------------------------------------------
// JSON utility
// ------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	resp, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		http.Error(w, `{"error": "Failed to encode JSON"}`, http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(resp)
}

func writeJSONError(w http.ResponseWriter, statusCode int, errMsg string) {
	writeJSON(w, statusCode, map[string]string{"error": errMsg})
}

// -----------------------------------------------------------------------
// Parse HTML
// -----------------------------------------------------------------------

func parseCSGTHtml(htmlContent string) ([]*CsgtData, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to create goquery document: %w", err)
	}

	data := &CsgtData{}

	// 1) First, parse the rows with a label + value
	//
	// The relevant portion for these is:
	//   <div class="form-group">
	//       <div class="row">
	//           <label class="control-label col-md-3 text-right">
	//               <span>Biển kiểm soát:</span>
	//           </label>
	//           <div class="col-md-9">98E1-714.78</div>
	//       </div>
	//   </div>
	//
	// We'll look under #bodyPrint123 .form-group .row, grab the label text
	// and the next .col-md-9 text, and switch on it.

	doc.Find("#bodyPrint123 .form-group .row").Each(func(i int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find("label span").Text())
		value := strings.TrimSpace(s.Find("div.col-md-9").Text())

		switch label {
		case "Biển kiểm soát:":
			data.Plate = value
		case "Màu biển:":
			data.PlateColor = value
		case "Loại phương tiện:":
			data.VehicleType = value
		case "Thời gian vi phạm:":
			data.ViolationTime = value
		case "Địa điểm vi phạm:":
			data.ViolationPlace = value
		case "Hành vi vi phạm:":
			data.ViolationAction = value
		case "Trạng thái:":
			data.Status = value
		case "Đơn vị phát hiện vi phạm:":
			data.DetectedBy = value
		case "Nơi giải quyết vụ việc:":
			// The row is present, but the value might be empty if details come after
			// We’ll parse subsequent lines separately below.
			// In some csgt.vn pages, there's an empty <div class="col-md-9"></div>.
		}
	})

	// 2) Now, parse the additional "form-group" blocks that do NOT have a .row,
	//    since the example includes lines like:
	//    <div class="form-group">1. Đội Cảnh sát giao thông, ...</div>
	//    <div class="form-group">Địa chỉ: ...</div>
	//    <div class="form-group">Số điện thoại liên hệ: 0911595121</div>
	//    <div class="form-group">2. Đội Cảnh sát giao thông ... </div>
	//    ...
	// We can collect them into a slice or a single multiline string.
	var resolutionLines []string

	doc.Find("#bodyPrint123 .form-group").Each(func(i int, sel *goquery.Selection) {
		// If this .form-group does NOT have a nested .row, it’s probably a free-text line
		if sel.Find(".row").Length() == 0 {
			txt := strings.TrimSpace(sel.Text())
			if txt != "" {
				resolutionLines = append(resolutionLines, txt)
			}
		}
	})

	// Join them into one block, or parse them further if needed
	if len(resolutionLines) > 0 {
		data.ResolutionLocation = strings.Join(resolutionLines, "\n")
	}

	// 3) Return as a slice with one element
  //    (or potentially more than one if you had multiple “blocks” on one page)
  return []*CsgtData{data}, nil
}
