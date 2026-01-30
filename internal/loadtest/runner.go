package loadtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

// CurlRequest represents a parsed curl command
type CurlRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    string
}

// Runner executes load tests with concurrent workers
type Runner struct {
	client *http.Client
}

// NewRunner creates a new load test runner
func NewRunner() *Runner {
	return &Runner{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Run executes the load test scenario with the given accounts
func (r *Runner) Run(scenario Scenario, accounts []Account) *LoadTestResult {
	result := &LoadTestResult{
		ScenarioName:   scenario.Name,
		TotalAccounts:  len(accounts),
		AccountResults: make([]AccountResult, 0, len(accounts)),
	}

	startTime := time.Now()

	// Create worker pool
	concurrency := scenario.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	if concurrency > len(accounts) {
		concurrency = len(accounts)
	}

	accountChan := make(chan Account, len(accounts))
	resultChan := make(chan AccountResult, len(accounts))
	globalVars := make(map[string]map[string]string)
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for account := range accountChan {
				accountResult := r.runAccountSteps(scenario, account, globalVars)
				resultChan <- accountResult
			}
		}(i)
	}

	// Send accounts to workers
	for _, account := range accounts {
		accountChan <- account
	}
	close(accountChan)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for accountResult := range resultChan {
		result.AccountResults = append(result.AccountResults, accountResult)
		if accountResult.Success {
			result.SuccessCount++
		} else {
			result.FailureCount++
		}
	}

	result.TotalDuration = time.Since(startTime).Milliseconds()
	if len(result.AccountResults) > 0 {
		var totalAccountTime int64
		for _, ar := range result.AccountResults {
			totalAccountTime += ar.TotalTime
		}
		result.AvgDuration = totalAccountTime / int64(len(result.AccountResults))
	}

	globalVars = nil
	return result
}

// runAccountSteps runs all steps for a single account
func (r *Runner) runAccountSteps(scenario Scenario, account Account, globalVars map[string]map[string]string) AccountResult {
	result := AccountResult{
		Username:    account.Username,
		StepResults: make([]StepResult, 0, len(scenario.Steps)),
		Success:     true,
	}

	startTime := time.Now()

	// Token storage for saving and using tokens between steps
	globalVars[account.Username] = make(map[string]string)
	globalVars[account.Username]["username"] = account.Username
	globalVars[account.Username]["password"] = account.Password

	for _, step := range scenario.Steps {
		stepResult := r.executeStep(account.Username, step, globalVars)
		result.StepResults = append(result.StepResults, stepResult)

		if !stepResult.Success {
			result.Success = false
			break // Stop on first failure
		}
	}

	result.TotalTime = time.Since(startTime).Milliseconds()
	return result
}

// executeStep executes a single step and returns the result
// If step.Path is a curl command, it parses and executes the curl
// Otherwise, it executes a normal HTTP request
func (r *Runner) executeStep(userName string, step Step, globalVars map[string]map[string]string) StepResult {
	// Check if the path is a curl command
	if isCurlCommand(step.Path) {
		return r.executeCurlStep(userName, step, globalVars)
	}

	// Normal HTTP request execution
	result := StepResult{
		StepName: step.Name,
	}

	startTime := time.Now()

	// Build URL
	reqURL := r.replacePlaceholders(step.Path, globalVars[userName])

	// Build request body
	var bodyReader io.Reader
	if step.Body != "" {
		body := r.replacePlaceholders(step.Body, globalVars[userName])
		bodyReader = bytes.NewBufferString(body)
	}

	// Create request
	req, err := http.NewRequest(step.Method, reqURL, bodyReader)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range step.Headers {
		req.Header.Set(key, r.replacePlaceholders(value, globalVars[userName]))
	}

	// Execute request
	resp, err := r.client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("request failed: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(startTime).Milliseconds()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to read response: %v", err)
		return result
	}

	// Check expected status
	if step.ExpectStatus > 0 && resp.StatusCode != step.ExpectStatus {
		result.ErrorMessage = fmt.Sprintf("expected status %d, got %d", step.ExpectStatus, resp.StatusCode)
		return result
	}

	// Extract and save variables from response headers and body
	// Use "header.X-Header-Name" for headers, or JSON path for body
	r.extractAndSaveVariables(userName, step.SaveVariables, resp.Header, respBody, globalVars)

	result.Success = true
	return result
}

// replacePlaceholders replaces {{key}} placeholders with values from tokens
func (r *Runner) replacePlaceholders(s string, globalVars map[string]string) string {
	result := s
	for key, value := range globalVars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// extractAndSaveVariables extracts variables from response headers and body
// SaveVariables format:
//   - "header.X-Header-Name" -> extracts from response header
//   - "json.path.here" -> extracts from response body (JSON)
func (r *Runner) extractAndSaveVariables(
	userName string,
	saveVariables map[string]string,
	respHeaders http.Header,
	respBody []byte,
	globalVars map[string]map[string]string,
) {
	for varName, path := range saveVariables {
		if varName == "" || path == "" {
			continue
		}

		// Check if extracting from header
		if strings.HasPrefix(path, "header.") {
			headerName := strings.TrimPrefix(path, "header.")
			headerValue := respHeaders.Get(headerName)
			if headerValue != "" {
				globalVars[userName][varName] = headerValue
			}
			continue
		}

		// Extract from response body (JSON)
		jsonValue := gjson.GetBytes(respBody, path)
		if jsonValue.Exists() {
			value := jsonValue.Raw
			if jsonValue.Type == gjson.String {
				value = jsonValue.String()
			}
			globalVars[userName][varName] = value
		}
	}
}

// isCurlCommand checks if the path is a curl command
func isCurlCommand(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "curl ") || strings.HasPrefix(trimmed, "curl\t")
}

// parseCurl parses a curl command and extracts URL, method, headers, and body
func parseCurl(curlCommand string) (*CurlRequest, error) {
	result := &CurlRequest{
		Method:  "GET",
		Headers: make(map[string]string),
	}

	if curlCommand == "" {
		return nil, fmt.Errorf("empty curl command")
	}

	// Normalize the curl command - handle line continuations and multiple lines
	normalized := curlCommand
	normalized = regexp.MustCompile(`\\\r?\n`).ReplaceAllString(normalized, " ")
	normalized = regexp.MustCompile(`\r?\n`).ReplaceAllString(normalized, " ")
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	normalized = strings.TrimSpace(normalized)

	// Extract URL - handle both --location and direct URL formats
	// Pattern: curl --location 'URL' or curl --location "URL" or curl 'URL' or curl URL
	urlPatterns := []string{
		`curl\s+(?:--location\s+)?['"]([^'"]+)['"]`,
		`curl\s+(?:--location\s+)?(\S+)`,
	}

	for _, pattern := range urlPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(normalized); len(matches) > 1 {
			result.URL = matches[1]
			break
		}
	}

	// Extract method - check for -X or --request
	methodRe := regexp.MustCompile(`(?:-X|--request)\s+['"]?([A-Z]+)['"]?`)
	if matches := methodRe.FindStringSubmatch(normalized); len(matches) > 1 {
		result.Method = strings.ToUpper(matches[1])
	} else if strings.Contains(normalized, "--data") || strings.Contains(normalized, "-d ") {
		// If there's data but no explicit method, assume POST
		result.Method = "POST"
	}

	// Extract headers - handle both -H and --header
	headerRe := regexp.MustCompile(`(?:-H|--header)\s+['"]([^'"]+)['"]`)
	headerMatches := headerRe.FindAllStringSubmatch(normalized, -1)
	for _, match := range headerMatches {
		if len(match) > 1 {
			headerStr := match[1]
			colonIndex := strings.Index(headerStr, ":")
			if colonIndex > 0 {
				key := strings.TrimSpace(headerStr[:colonIndex])
				value := strings.TrimSpace(headerStr[colonIndex+1:])
				result.Headers[key] = value
			}
		}
	}

	// Extract body/data - handle --data, --data-raw, --data-binary, -d
	dataPatterns := []string{
		`(?:--data-raw|--data-binary|--data|-d)\s+'([^']+)'`,
		`(?:--data-raw|--data-binary|--data|-d)\s+"([^"]+)"`,
		`(?:--data-raw|--data-binary|--data|-d)\s+(\{[^}]+\})`,
	}

	for _, pattern := range dataPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(normalized); len(matches) > 1 {
			result.Body = matches[1]
			break
		}
	}

	if result.URL == "" {
		return nil, fmt.Errorf("could not extract URL from curl command")
	}

	return result, nil
}

// buildCurlCommand constructs a curl command string from an HTTP request
func buildCurlCommand(req *http.Request, body string) string {
	var parts []string
	parts = append(parts, "curl")
	parts = append(parts, fmt.Sprintf("'%s'", req.URL.String()))
	parts = append(parts, fmt.Sprintf("-X %s", req.Method))

	for key, values := range req.Header {
		for _, value := range values {
			parts = append(parts, fmt.Sprintf("-H '%s: %s'", key, value))
		}
	}

	if body != "" {
		parts = append(parts, fmt.Sprintf("-d '%s'", body))
	}

	return strings.Join(parts, " ")
}

// executeCurlStep executes a step where path contains a curl command
func (r *Runner) executeCurlStep(userName string, step Step, globalVars map[string]map[string]string) StepResult {
	result := StepResult{
		StepName: step.Name,
	}

	startTime := time.Now()

	// Replace placeholders in the curl command first
	curlCmd := r.replacePlaceholders(step.Path, globalVars[userName])

	// Parse the curl command
	curlReq, err := parseCurl(curlCmd)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to parse curl: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}

	// Replace placeholders in body - use step.Body if provided (override), otherwise use curl body
	body := curlReq.Body
	if step.Body != "" {
		body = r.replacePlaceholders(step.Body, globalVars[userName])
	}

	// Build request
	var bodyReader io.Reader
	if body != "" {
		bodyReader = bytes.NewBufferString(body)
	}

	// Use method from step if provided (override), otherwise use parsed method
	method := curlReq.Method
	if step.Method != "" {
		method = step.Method
	}

	// Parse and rebuild URL to handle any placeholders that might be in query params
	parsedURL := r.replacePlaceholders(curlReq.URL, globalVars[userName])

	req, err := http.NewRequest(method, parsedURL, bodyReader)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}

	// Set default Content-Type
	req.Header.Set("Content-Type", "application/json")

	// Apply headers from curl (with placeholder replacement)
	for key, value := range curlReq.Headers {
		req.Header.Set(key, r.replacePlaceholders(value, globalVars[userName]))
	}

	// Apply headers from step (override curl headers)
	for key, value := range step.Headers {
		req.Header.Set(key, r.replacePlaceholders(value, globalVars[userName]))
	}

	// Log the request for debugging
	// fmt.Printf("[DEBUG] Executing request for user %s:\n", userName)
	// fmt.Printf("  Method: %s\n", req.Method)
	// fmt.Printf("  URL: %s\n", req.URL.String())
	// fmt.Printf("  Headers:\n")
	// for key, values := range req.Header {
	// 	for _, value := range values {
	// 		fmt.Printf("    %s: %s\n", key, value)
	// 	}
	// }
	// if body != "" {
	// 	fmt.Printf("  Body: %s\n", body)
	// }
	// fmt.Println()
	// // Log the curl command for Postman import
	// curlForPostman := buildCurlCommand(req, body)
	// fmt.Printf("[POSTMAN CURL] %s\n\n", curlForPostman)
	// Execute request
	resp, err := r.client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("request failed: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(startTime).Milliseconds()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to read response: %v", err)
		return result
	}

	// Check expected status
	if step.ExpectStatus > 0 && resp.StatusCode != step.ExpectStatus {
		result.ErrorMessage = fmt.Sprintf("expected status %d, got %d", step.ExpectStatus, resp.StatusCode)
		return result
	}

	// Extract and save variables from response headers and body
	r.extractAndSaveVariables(userName, step.SaveVariables, resp.Header, respBody, globalVars)

	result.MetaData = map[string]string{
		"ad_id": globalVars[userName]["ad_id"],
	}
	result.Success = true
	return result
}

// PrintResult prints the load test result to stdout
func (r *Runner) PrintResult(result *LoadTestResult) {
	fmt.Println("\n========================================")
	fmt.Printf("Load Test Result: %s\n", result.ScenarioName)
	fmt.Println("========================================")
	fmt.Printf("Total Accounts:  %d\n", result.TotalAccounts)
	fmt.Printf("Success:         %d\n", result.SuccessCount)
	fmt.Printf("Failure:         %d\n", result.FailureCount)
	fmt.Printf("Success Rate:    %.2f%%\n", float64(result.SuccessCount)/float64(result.TotalAccounts)*100)
	fmt.Printf("Total Duration:  %dms\n", result.TotalDuration)
	fmt.Printf("Avg Duration:    %dms\n", result.AvgDuration)
	fmt.Println("----------------------------------------")

	// Print detailed results for failed accounts
	for _, ar := range result.AccountResults {
		if !ar.Success {
			fmt.Printf("\nFailed: %s (took %dms)\n", ar.Username, ar.TotalTime)
			for _, sr := range ar.StepResults {
				status := "✓"
				if !sr.Success {
					status = "✗"
				}
				fmt.Printf("  %s %s: %dms", status, sr.StepName, sr.Duration)
				if sr.ErrorMessage != "" {
					fmt.Printf(" - %s", sr.ErrorMessage)
				}
				fmt.Println()
			}
		}
	}

	fmt.Println("\n========================================")
}

// ExportResultJSON exports the result as JSON
func (r *Runner) ExportResultJSON(result *LoadTestResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}
