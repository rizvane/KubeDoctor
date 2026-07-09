/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// GenerateRecommendation generates a recommendation based on pod logs and exit reason.
func GenerateRecommendation(logs string, reason string) string {
	lowerLogs := strings.ToLower(logs)
	lowerReason := strings.ToLower(reason)

	// Basic heuristic engine
	if strings.Contains(lowerLogs, "exit code 137") || strings.Contains(lowerReason, "oomkilled") {
		return "OOMKilled detected. Recommendation: Increase the memory limits and requests for this container in its Deployment/Pod configuration."
	}
	if strings.Contains(lowerLogs, "connection refused") {
		return "Connection refused. Recommendation: Check if the target service is running, listening on the correct port, and accessible from this pod (NetworkPolicies)."
	}
	if strings.Contains(lowerLogs, "no such file or directory") {
		return "File not found. Recommendation: Verify volume mounts, init containers, or the container image contents."
	}
	if strings.Contains(lowerReason, "crashloopbackoff") {
		return "CrashLoopBackOff detected. Recommendation: Review the logs above to identify why the application failed to start or crashed shortly after starting."
	}
	if strings.Contains(lowerReason, "imagepullbackoff") || strings.Contains(lowerReason, "errimagepull") {
		return "Image pull error. Recommendation: Verify that the image repository/tag exists, the image name is spelled correctly, and the node has the correct ImagePullSecrets to access the registry."
	}

	// Unhandled error, attempt to use LLM if configured
	llmRecommendation := askLLMForRecommendation(logs, reason)
	if llmRecommendation != "" {
		return fmt.Sprintf("LLM Analysis: %s", llmRecommendation)
	}

	return "No specific recommendation available. Consider sending these logs to a more advanced analysis tool."
}

// askLLMForRecommendation sends anonymized logs to an LLM for analysis if OPENAI_API_KEY is set.
func askLLMForRecommendation(logs string, reason string) string {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return ""
	}

	// Anonymize logs before sending them to the LLM to prevent leaking sensitive data
	anonymizedLogs := anonymizeLogs(logs)

	prompt := fmt.Sprintf("Analyze the following Kubernetes error logs and provide a brief recommendation on how to fix it.\nReason: %s\nLogs:\n%s", reason, anonymizedLogs)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "system", "content": "You are an expert Kubernetes architect. Provide concise, actionable advice for fixing cluster issues based on logs."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 150,
	})
	if err != nil {
		return ""
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return ""
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return ""
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return ""
	}

	content, ok := message["content"].(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(content)
}

// anonymizeLogs redacts sensitive information such as IPv4 addresses, emails, and tokens
func anonymizeLogs(logs string) string {
	// Redact IP Addresses
	ipRegex := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	logs = ipRegex.ReplaceAllString(logs, "[REDACTED_IP]")

	// Redact Email Addresses
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	logs = emailRegex.ReplaceAllString(logs, "[REDACTED_EMAIL]")

	// Redact Bearer Tokens
	tokenRegex := regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-\._~+/]+=*`)
	logs = tokenRegex.ReplaceAllString(logs, "Bearer [REDACTED_TOKEN]")

	return logs
}
