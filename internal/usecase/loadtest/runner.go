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
		// Check if step should be executed based on condition
		shouldExecute, err := r.evaluateCondition(step.Condition, globalVars[account.Username])
		if err != nil {
			result.Success = false
			result.StepResults = append(result.StepResults, StepResult{
				StepName:     step.Name,
				StatusCode:   400,
				ErrorMessage: err.Error(),
			})
			break
		}
		if !shouldExecute {
			// Skip this step - condition not met
			continue
		}

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

	// Normal HTTP request execution with retry support
	result := StepResult{
		StepName: step.Name,
	}

	startTime := time.Now()
	retryDeadline := startTime.Add(time.Duration(step.RetryForSeconds) * time.Second)
	retryCount := 0

	// Retry loop
	for {
		retryCount++
		// Build URL
		reqURL, err := r.replacePlaceholders(step.Path, globalVars[userName])
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to replace placeholders in path: %v", err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}
		// Build request body
		var bodyReader io.Reader
		var bodyString string
		var contentType string
		// Normal JSON body
		if step.Body != "" {
			bodyString, err = r.replacePlaceholders(step.Body, globalVars[userName])
			if err != nil {
				result.ErrorMessage = fmt.Sprintf("failed to replace placeholders in body: %v", err)
				result.Duration = time.Since(startTime).Milliseconds()
				return result
			}
			bodyReader = bytes.NewBufferString(bodyString)
		}
		contentType = "application/json"

		// Create request
		req, err := http.NewRequest(step.Method, reqURL, bodyReader)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		// Set headers
		req.Header.Set("Content-Type", contentType)
		for key, value := range step.Headers {
			headerValue, err := r.replacePlaceholders(value, globalVars[userName])
			if err != nil {
				result.ErrorMessage = fmt.Sprintf("failed to replace placeholders in header %s: %v", key, err)
				result.Duration = time.Since(startTime).Milliseconds()
				return result
			}
			req.Header.Set(key, headerValue)
		}

		// Print curl command for debugging
		curlCmd := buildCurlCommand(req, bodyString)
		fmt.Printf("\n[DEBUG] Curl Command:\n%s\n\n", curlCmd)

		// Execute request
		attemptStart := time.Now()
		resp, err := r.client.Do(req)
		if err != nil {
			// Check if we should retry (check both time and count limits)
			shouldRetry := step.RetryForSeconds > 0 && time.Now().Before(retryDeadline)
			if step.MaxRetryTimes > 0 {
				shouldRetry = shouldRetry && retryCount < step.MaxRetryTimes
			}
			if shouldRetry {
				time.Sleep(1 * time.Second) // Wait 1 second before retrying
				continue
			}
			result.ErrorMessage = fmt.Sprintf("request failed after %d attempt(s): %v", retryCount, err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		result.StatusCode = resp.StatusCode

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			// Check if we should retry (check both time and count limits)
			shouldRetry := step.RetryForSeconds > 0 && time.Now().Before(retryDeadline)
			if step.MaxRetryTimes > 0 {
				shouldRetry = shouldRetry && retryCount < step.MaxRetryTimes
			}
			if shouldRetry {
				time.Sleep(1 * time.Second)
				continue
			}
			result.ErrorMessage = fmt.Sprintf("failed to read response after %d attempt(s): %v", retryCount, err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		// Check expected status
		if step.ExpectStatus > 0 && resp.StatusCode != step.ExpectStatus {
			// Check if we should retry (check both time and count limits)
			shouldRetry := step.RetryForSeconds > 0 && time.Now().Before(retryDeadline)
			if step.MaxRetryTimes > 0 {
				shouldRetry = shouldRetry && retryCount < step.MaxRetryTimes
			}
			if shouldRetry {
				time.Sleep(1 * time.Second)
				continue
			}
			result.ErrorMessage = fmt.Sprintf("expected status %d, got %d after %d attempt(s): %s", step.ExpectStatus, resp.StatusCode, retryCount, respBody)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		// Success - extract and save variables from response headers and body
		r.extractAndSaveVariables(userName, step.SaveVariables, resp.Header, respBody, globalVars)

		result.Success = true
		result.Duration = time.Since(attemptStart).Milliseconds()

		// Wait after successful execution if configured
		if step.WaitAfterSeconds > 0 {
			time.Sleep(time.Duration(step.WaitAfterSeconds) * time.Second)
		}

		return result
	}
}

// replacePlaceholders replaces {{key}} placeholders with values from tokens
// Returns error if any placeholder is not found in globalVars
func (r *Runner) replacePlaceholders(s string, globalVars map[string]string) (string, error) {
	// Find all placeholders in the string
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := re.FindAllStringSubmatch(s, -1)

	// Check if all placeholders exist in globalVars
	var missingVars []string
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if _, exists := globalVars[varName]; !exists {
				missingVars = append(missingVars, varName)
			}
		}
	}

	// If there are missing variables, return error
	if len(missingVars) > 0 {
		return s, fmt.Errorf("missing variables: %s", strings.Join(missingVars, ", "))
	}

	// Replace all placeholders
	result := s
	for key, value := range globalVars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result, nil
}

// evaluateCondition evaluates a condition string with support for complex boolean expressions
// Supported formats:
//   - Simple: "{{variable}} == true"
//   - AND: "{{var1}} == true && {{var2}} > 0"
//   - OR: "{{var1}} == true || {{var2}} == false"
//   - Parentheses: "({{var1}} == true && {{var2}} > 0) || {{var3}} == \"active\""
//   - Operators: ==, !=, >, <, >=, <=
func (r *Runner) evaluateCondition(condition string, globalVars map[string]string) (bool, error) {
	// If no condition, always execute
	if strings.TrimSpace(condition) == "" {
		return true, nil
	}

	// Replace placeholders with actual values
	evaluated, err := r.replacePlaceholders(condition, globalVars)
	if err != nil {
		// If there are missing variables in condition, log and return false (skip step)
		fmt.Printf("[WARN] Condition evaluation failed: %v. Condition: %s\n", err, condition)
		return false, fmt.Errorf("missing variables")
	}
	evaluated = strings.TrimSpace(evaluated)

	// Evaluate the boolean expression
	return r.evaluateBooleanExpression(evaluated)
}

// evaluateBooleanExpression evaluates a boolean expression with &&, ||, and parentheses
func (r *Runner) evaluateBooleanExpression(expr string) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Handle parentheses first (innermost first)
	for strings.Contains(expr, "(") {
		// Find innermost parentheses
		start := strings.LastIndex(expr, "(")
		if start == -1 {
			break
		}

		end := strings.Index(expr[start:], ")")
		if end == -1 {
			// Malformed expression, treat as false
			return false, fmt.Errorf("Malformed expression")
		}
		end += start

		// Evaluate expression inside parentheses
		inner := expr[start+1 : end]
		innerResult, _ := r.evaluateBooleanExpression(inner)

		// Replace parentheses expression with result
		replacement := "true"
		if !innerResult {
			replacement = "false"
		}
		expr = expr[:start] + replacement + expr[end+1:]
	}

	// Now handle OR operator (lower precedence)
	if strings.Contains(expr, "||") {
		parts := strings.Split(expr, "||")
		for _, part := range parts {
			result, err := r.evaluateBooleanExpression(strings.TrimSpace(part))
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	// Handle AND operator (higher precedence)
	if strings.Contains(expr, "&&") {
		parts := strings.Split(expr, "&&")
		for _, part := range parts {
			result, err := r.evaluateBooleanExpression(strings.TrimSpace(part))
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	// No logical operators, evaluate as simple comparison
	return r.evaluateSimpleCondition(expr), nil
}

// evaluateSimpleCondition evaluates a simple comparison without logical operators
func (r *Runner) evaluateSimpleCondition(condition string) bool {
	condition = strings.TrimSpace(condition)

	// Check for comparison operators
	operators := []string{"==", "!=", ">=", "<=", ">", "<"}

	for _, op := range operators {
		if strings.Contains(condition, op) {
			parts := strings.SplitN(condition, op, 2)
			if len(parts) != 2 {
				continue
			}

			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])

			// Remove quotes from string literals
			left = strings.Trim(left, "\"'")
			right = strings.Trim(right, "\"'")

			return r.compareValues(left, right, op)
		}
	}

	// No operator found, check if the value is truthy
	// "true", "1", non-empty strings are truthy
	// "false", "0", empty strings are falsy
	condition = strings.Trim(condition, "\"'")
	return condition != "" && condition != "false" && condition != "0"
}

// compareValues compares two string values using the given operator
func (r *Runner) compareValues(left, right, operator string) bool {
	switch operator {
	case "==":
		return left == right
	case "!=":
		return left != right
	case ">", "<", ">=", "<=":
		// Try to parse as numbers for numeric comparison
		leftNum, leftErr := parseNumber(left)
		rightNum, rightErr := parseNumber(right)

		if leftErr == nil && rightErr == nil {
			switch operator {
			case ">":
				return leftNum > rightNum
			case "<":
				return leftNum < rightNum
			case ">=":
				return leftNum >= rightNum
			case "<=":
				return leftNum <= rightNum
			}
		}

		// Fall back to string comparison
		switch operator {
		case ">":
			return left > right
		case "<":
			return left < right
		case ">=":
			return left >= right
		case "<=":
			return left <= right
		}
	}

	return false
}

// parseNumber attempts to parse a string as a float64
func parseNumber(s string) (float64, error) {
	s = strings.TrimSpace(s)
	var num float64
	_, err := fmt.Sscanf(s, "%f", &num)
	return num, err
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
	retryDeadline := startTime.Add(time.Duration(step.RetryForSeconds) * time.Second)
	retryCount := 0

	// Replace placeholders in the curl command first
	curlCmd, err := r.replacePlaceholders(step.Path, globalVars[userName])
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to replace placeholders in curl command: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}

	// Parse the curl command
	curlReq, err := parseCurl(curlCmd)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to parse curl: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}

	// Retry loop
	for {
		retryCount++

		// Build request body
		var bodyReader io.Reader
		var body string
		var contentType string
		var err error

		// Replace placeholders in body - use step.Body if provided (override), otherwise use curl body
		body = curlReq.Body
		if step.Body != "" {
			body, err = r.replacePlaceholders(step.Body, globalVars[userName])
			if err != nil {
				result.ErrorMessage = fmt.Sprintf("failed to replace placeholders in body: %v", err)
				result.Duration = time.Since(startTime).Milliseconds()
				return result
			}
		}

		if body != "" {
			bodyReader = bytes.NewBufferString(body)
		}
		contentType = "application/json"

		// Use method from step if provided (override), otherwise use parsed method
		method := curlReq.Method
		if step.Method != "" {
			method = step.Method
		}

		// Parse and rebuild URL to handle any placeholders that might be in query params
		parsedURL, err := r.replacePlaceholders(curlReq.URL, globalVars[userName])
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to replace placeholders in URL: %v", err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		req, err := http.NewRequest(method, parsedURL, bodyReader)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		// Set Content-Type (use multipart content type if files are uploaded, otherwise default to JSON)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		} else {
			req.Header.Set("Content-Type", "application/json")
		}

		// Apply headers from curl (with placeholder replacement)
		for key, value := range curlReq.Headers {
			headerValue, err := r.replacePlaceholders(value, globalVars[userName])
			if err != nil {
				result.ErrorMessage = fmt.Sprintf("failed to replace placeholders in curl header %s: %v", key, err)
				result.Duration = time.Since(startTime).Milliseconds()
				return result
			}
			req.Header.Set(key, headerValue)
		}

		// Apply headers from step (override curl headers)
		for key, value := range step.Headers {
			headerValue, err := r.replacePlaceholders(value, globalVars[userName])
			if err != nil {
				result.ErrorMessage = fmt.Sprintf("failed to replace placeholders in step header %s: %v", key, err)
				result.Duration = time.Since(startTime).Milliseconds()
				return result
			}
			req.Header.Set(key, headerValue)
		}

		// Print curl command for debugging
		curlCmd := buildCurlCommand(req, body)
		fmt.Printf("\n[DEBUG] Curl Command:\n%s\n\n", curlCmd)

		// Execute request
		attemptStart := time.Now()
		resp, err := r.client.Do(req)
		if err != nil {
			// Check if we should retry (check both time and count limits)
			shouldRetry := step.RetryForSeconds > 0 && time.Now().Before(retryDeadline)
			if step.MaxRetryTimes > 0 {
				shouldRetry = shouldRetry && retryCount < step.MaxRetryTimes
			}
			if shouldRetry {
				time.Sleep(1 * time.Second) // Wait 1 second before retrying
				continue
			}
			result.ErrorMessage = fmt.Sprintf("request failed after %d attempt(s): %v", retryCount, err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		result.StatusCode = resp.StatusCode

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			// Check if we should retry (check both time and count limits)
			shouldRetry := step.RetryForSeconds > 0 && time.Now().Before(retryDeadline)
			if step.MaxRetryTimes > 0 {
				shouldRetry = shouldRetry && retryCount < step.MaxRetryTimes
			}
			if shouldRetry {
				time.Sleep(1 * time.Second)
				continue
			}
			result.ErrorMessage = fmt.Sprintf("failed to read response after %d attempt(s): %v", retryCount, err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		// Check expected status
		if step.ExpectStatus > 0 && resp.StatusCode != step.ExpectStatus {
			// Check if we should retry (check both time and count limits)
			shouldRetry := step.RetryForSeconds > 0 && time.Now().Before(retryDeadline)
			if step.MaxRetryTimes > 0 {
				shouldRetry = shouldRetry && retryCount < step.MaxRetryTimes
			}
			if shouldRetry {
				time.Sleep(1 * time.Second)
				continue
			}
			result.ErrorMessage = fmt.Sprintf("expected status %d, got %d after %d attempt(s): %s", step.ExpectStatus, resp.StatusCode, retryCount, respBody)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}

		// Success - extract and save variables from response headers and body
		r.extractAndSaveVariables(userName, step.SaveVariables, resp.Header, respBody, globalVars)

		result.MetaData = map[string]string{
			"ad_id": globalVars[userName]["ad_id"],
		}
		result.Success = true
		result.Duration = time.Since(attemptStart).Milliseconds()

		// Wait after successful execution if configured
		if step.WaitAfterSeconds > 0 {
			time.Sleep(time.Duration(step.WaitAfterSeconds) * time.Second)
		}

		return result
	}
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
