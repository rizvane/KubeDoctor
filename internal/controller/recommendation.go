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

	// Advanced heuristic engine with YAML patch proposals
	if strings.Contains(lowerLogs, "exit code 137") || strings.Contains(lowerReason, "oomkilled") {
		return `OOMKilled detected. The container exceeded its memory limit.
Recommendation: Increase the memory limit and request for this container in its Deployment.

💡 Corrective YAML Patch (Adjust limits accordingly):
---
resources:
  requests:
    memory: "512Mi" # Increase from current
  limits:
    memory: "1Gi"   # Increase from current
`
	}

	if strings.Contains(lowerReason, "imagepullbackoff") || strings.Contains(lowerReason, "errimagepull") {
		return `ImagePullBackOff detected. Kubernetes cannot pull the container image.
Recommendation:
1. Verify that the image name and tag are correct.
2. Ensure the image repository is accessible.
3. If the registry is private, ensure imagePullSecrets are configured.

💡 Corrective YAML Patch (If using a private registry):
---
spec:
  template:
    spec:
      imagePullSecrets:
      - name: my-registry-secret
`
	}

	if strings.Contains(lowerLogs, "connection refused") {
		return `Connection refused detected. The application is trying to communicate with a service that is unreachable.
Recommendation:
1. Check if the target Service is running and has endpoints (kubectl get ep).
2. Ensure the port numbers match between the Service and the Pod.
3. Verify NetworkPolicies are not blocking traffic.

💡 Corrective YAML Example (Service & NetworkPolicy checks):
---
# Ensure your service targets the correct container port
spec:
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080 # This must match your container's listening port
`
	}

	if strings.Contains(lowerLogs, "no such file or directory") || strings.Contains(lowerLogs, "not found") {
		return `File or Directory not found detected.
Recommendation: Verify your volume mounts, ConfigMaps, or Secrets. If the file is expected to be inside the image, verify the Dockerfile build process.

💡 Corrective YAML Patch (Mounting a missing ConfigMap):
---
spec:
  template:
    spec:
      containers:
      - name: my-app
        volumeMounts:
        - name: config-volume
          mountPath: /etc/config
      volumes:
      - name: config-volume
        configMap:
          name: my-configmap
`
	}

	if strings.Contains(lowerReason, "createcontainerconfigerror") {
		return `CreateContainerConfigError detected. The Pod is failing to start because a referenced Secret or ConfigMap is missing.
Recommendation: Ensure the referenced ConfigMap or Secret exists in the same namespace.

💡 Command to check missing resources:
kubectl get configmap,secret -n <namespace>
`
	}

	if strings.Contains(lowerLogs, "permission denied") {
		return `Permission Denied error detected. The container process lacks the necessary privileges to access a file or execute a command.
Recommendation:
1. Verify the file system permissions where the container is trying to read/write.
2. If using persistent volumes, ensure the 'fsGroup' matches the user running the process.

💡 Corrective YAML Patch (Setting SecurityContext):
---
spec:
  template:
    spec:
      securityContext:
        fsGroup: 1000 # Example: Change to the ID of the user needing access
      containers:
      - name: my-app
        securityContext:
          runAsUser: 1000
`
	}

	if strings.Contains(lowerLogs, "no space left on device") {
		return `No Space Left on Device error detected. The node's ephemeral storage or the container's volume is full.
Recommendation:
1. Check if the application is writing excessive logs or temp files.
2. Set ephemeral-storage limits to prevent evicting the entire Node.

💡 Corrective YAML Patch (Ephemeral Storage limits):
---
spec:
  template:
    spec:
      containers:
      - name: my-app
        resources:
          requests:
            ephemeral-storage: "1Gi"
          limits:
            ephemeral-storage: "2Gi"
`
	}

	if strings.Contains(lowerLogs, "exec format error") {
		return `Exec Format Error detected. The container image was built for a different CPU architecture than the Node it's running on (e.g., ARM64 image on an AMD64 node).
Recommendation:
1. Rebuild the Docker image for the correct target architecture using 'docker buildx'.
2. Or use nodeSelectors to ensure the Pod schedules only on matching architecture nodes.

💡 Corrective YAML Patch (NodeSelector constraint):
---
spec:
  template:
    spec:
      nodeSelector:
        kubernetes.io/arch: arm64 # Force scheduling on ARM nodes
`
	}

	if strings.Contains(lowerLogs, "java.lang.outofmemoryerror") {
		return `Java OutOfMemoryError detected. This is a Heap Space crash, different from a Container OOMKilled (Exit 137).
Recommendation: Increase the JVM max heap size to match the container's RAM limits.

💡 Corrective YAML Patch (Adjusting JAVA_OPTS):
---
spec:
  template:
    spec:
      containers:
      - name: java-app
        env:
        - name: JAVA_OPTS
          value: "-Xms512m -Xmx1024m" # Ensure this is lower than resources.limits.memory
`
	}

	if strings.Contains(lowerLogs, "lookup") && strings.Contains(lowerLogs, "no such host") {
		return `DNS Resolution Timeout/Error detected. The Pod cannot resolve external or internal hostnames.
Recommendation:
1. Check if the CoreDNS pods in the 'kube-system' namespace are running correctly.
2. Verify if a NetworkPolicy is blocking UDP port 53 traffic.
`
	}

	if strings.Contains(lowerLogs, "exit code 143") {
		return `Exit Code 143 (SIGTERM Timeout) detected. The application was asked to shut down gracefully but took too long and was forcefully killed.
Recommendation: Increase the terminationGracePeriodSeconds if the application legitimately needs more time to clean up connections.

💡 Corrective YAML Patch:
---
spec:
  template:
    spec:
      terminationGracePeriodSeconds: 60 # Default is 30. Increase as needed.
`
	}

	if strings.Contains(lowerReason, "runcontainererror") || strings.Contains(lowerReason, "starterror") {
		return `RunContainerError detected. The container failed to start completely.
Recommendation:
1. Verify the 'command' and 'args' defined in the Pod spec. The executable might not exist in the image path.
2. Check if the Entrypoint script has execution (+x) permissions.
`
	}

	if strings.Contains(lowerReason, "crashloopbackoff") {
		return `CrashLoopBackOff detected. The application starts but crashes repeatedly.
Recommendation:
1. Check the logs above for application-specific panic/error stack traces.
2. Ensure the liveness probe is not failing too quickly.
3. Verify environment variables are correct.

💡 Corrective YAML Patch (Adjusting Liveness Probes):
---
spec:
  template:
    spec:
      containers:
      - name: my-app
        livenessProbe:
          initialDelaySeconds: 30 # Give the app more time to start
          periodSeconds: 10
`
	}

	// Unhandled error, attempt to use LLM if configured
	llmRecommendation := askLLMForRecommendation(logs, reason)
	if llmRecommendation != "" {
		return fmt.Sprintf("AI Diagnostic Analysis:\n%s\n\n(Generated via LLM)", llmRecommendation)
	}

	return "No specific automated recommendation available. Please analyze the provided logs manually or configure OPENAI_API_KEY for AI-driven insights."
}

// askLLMForRecommendation sends anonymized logs to an LLM for analysis.
// If using a local model like Ollama, an API key is often not required.
func askLLMForRecommendation(logs string, reason string) string {
	llmURL := os.Getenv("LLM_API_URL")
	if llmURL == "" {
		// Default to a local, in-cluster Ollama instance
		llmURL = "http://ollama.kubedoctor-system.svc.cluster.local:11434/v1/chat/completions"
	}

	llmModel := os.Getenv("LLM_MODEL")
	if llmModel == "" {
		// Default to a free, local model
		llmModel = "llama3"
	}

	// Anonymize logs before sending them to the LLM to prevent leaking sensitive data
	anonymizedLogs := anonymizeLogs(logs)

	prompt := fmt.Sprintf("Act as an expert Kubernetes architect. Analyze the following Kubernetes error logs and provide a detailed recommendation on how to fix it, including a specific Kubernetes YAML patch example if applicable.\nReason: %s\nLogs:\n%s", reason, anonymizedLogs)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model": llmModel,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a Kubernetes expert. Provide concise, actionable advice for fixing cluster issues based on logs. Provide correct YAML patches if appropriate."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 300,
	})
	if err != nil {
		return ""
	}

	req, err := http.NewRequest("POST", llmURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return ""
	}

	// Add Authorization header only if an API key is provided
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

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
