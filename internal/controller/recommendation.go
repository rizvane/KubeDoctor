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
	lang := getLang()

	// Advanced heuristic engine with YAML patch proposals
	if strings.Contains(lowerLogs, "exit code 137") || strings.Contains(lowerReason, "oomkilled") {
		if lang == "fr" {
			return `OOMKilled détecté. Le conteneur a dépassé sa limite de mémoire.
Recommandation : Augmentez la limite (limit) et la requête (request) de mémoire pour ce conteneur dans son Deployment.

💡 Patch YAML Correctif (Ajustez les limites en conséquence) :
---
resources:
  requests:
    memory: "512Mi" # Augmentez par rapport à l'actuel
  limits:
    memory: "1Gi"   # Augmentez par rapport à l'actuel
`
		}
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
		if lang == "fr" {
			return `ImagePullBackOff détecté. Kubernetes ne peut pas télécharger l'image du conteneur.
Recommandation :
1. Vérifiez que le nom et le tag de l'image sont corrects.
2. Assurez-vous que le registre d'images est accessible.
3. Si le registre est privé, assurez-vous que les 'imagePullSecrets' sont configurés.

💡 Patch YAML Correctif (Si vous utilisez un registre privé) :
---
spec:
  template:
    spec:
      imagePullSecrets:
      - name: mon-secret-registre
`
		}
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
		if lang == "fr" {
			return `Connection refused (Connexion refusée) détecté. L'application essaie de communiquer avec un service injoignable.
Recommandation :
1. Vérifiez si le Service cible est en cours d'exécution et possède des endpoints (kubectl get ep).
2. Assurez-vous que les numéros de port correspondent entre le Service et le Pod.
3. Vérifiez qu'aucune NetworkPolicy ne bloque le trafic.

💡 Exemple YAML Correctif (Vérification du Service et NetworkPolicy) :
---
# Assurez-vous que votre service cible le bon port de conteneur
spec:
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080 # Doit correspondre au port d'écoute de votre conteneur
`
		}
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
		if lang == "fr" {
			return `Fichier ou Répertoire introuvable détecté.
Recommandation : Vérifiez vos montages de volumes, ConfigMaps ou Secrets. Si le fichier est censé être dans l'image, vérifiez le processus de build du Dockerfile.

💡 Patch YAML Correctif (Montage d'une ConfigMap manquante) :
---
spec:
  template:
    spec:
      containers:
      - name: mon-app
        volumeMounts:
        - name: config-volume
          mountPath: /etc/config
      volumes:
      - name: config-volume
        configMap:
          name: ma-configmap
`
		}
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
		if lang == "fr" {
			return `CreateContainerConfigError détecté. Le Pod ne démarre pas car un Secret ou une ConfigMap référencé(e) est manquant(e).
Recommandation : Assurez-vous que la ConfigMap ou le Secret référencé existe dans le même namespace.

💡 Commande pour vérifier les ressources manquantes :
kubectl get configmap,secret -n <namespace>
`
		}
		return `CreateContainerConfigError detected. The Pod is failing to start because a referenced Secret or ConfigMap is missing.
Recommendation: Ensure the referenced ConfigMap or Secret exists in the same namespace.

💡 Command to check missing resources:
kubectl get configmap,secret -n <namespace>
`
	}

	if strings.Contains(lowerLogs, "permission denied") {
		if lang == "fr" {
			return `Erreur Permission Denied (Accès refusé) détectée. Le processus du conteneur n'a pas les privilèges nécessaires pour accéder à un fichier ou exécuter une commande.
Recommandation :
1. Vérifiez les permissions du système de fichiers là où le conteneur tente de lire/écrire.
2. Si vous utilisez des volumes persistants, assurez-vous que le 'fsGroup' correspond à l'utilisateur exécutant le processus.

💡 Patch YAML Correctif (Configuration du SecurityContext) :
---
spec:
  template:
    spec:
      securityContext:
        fsGroup: 1000 # Exemple : Modifiez avec l'ID de l'utilisateur nécessitant l'accès
      containers:
      - name: mon-app
        securityContext:
          runAsUser: 1000
`
		}
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
		if lang == "fr" {
			return `Erreur No Space Left on Device (Espace disque insuffisant) détectée. Le stockage éphémère du noeud ou le volume du conteneur est plein.
Recommandation :
1. Vérifiez si l'application écrit des logs excessifs ou des fichiers temporaires.
2. Définissez des limites 'ephemeral-storage' pour éviter l'éviction de l'intégralité du Noeud.

💡 Patch YAML Correctif (Limites de stockage éphémère) :
---
spec:
  template:
    spec:
      containers:
      - name: mon-app
        resources:
          requests:
            ephemeral-storage: "1Gi"
          limits:
            ephemeral-storage: "2Gi"
`
		}
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
		if lang == "fr" {
			return `Exec Format Error détecté. L'image du conteneur a été construite pour une architecture CPU différente de celle du Noeud sur lequel elle s'exécute (ex: image ARM64 sur un noeud AMD64).
Recommandation :
1. Reconstruisez l'image Docker pour la bonne architecture cible via 'docker buildx'.
2. Ou utilisez des nodeSelectors pour forcer le placement du Pod uniquement sur des noeuds correspondants.

💡 Patch YAML Correctif (Contrainte NodeSelector) :
---
spec:
  template:
    spec:
      nodeSelector:
        kubernetes.io/arch: arm64 # Force le déploiement sur les noeuds ARM
`
		}
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
		if lang == "fr" {
			return `Java OutOfMemoryError détecté. Il s'agit d'un plantage de l'espace Heap, différent d'un OOMKilled de conteneur (Exit 137).
Recommandation : Augmentez la taille max du Heap JVM pour correspondre aux limites RAM du conteneur.

💡 Patch YAML Correctif (Ajustement de JAVA_OPTS) :
---
spec:
  template:
    spec:
      containers:
      - name: java-app
        env:
        - name: JAVA_OPTS
          value: "-Xms512m -Xmx1024m" # Assurez-vous que c'est inférieur à resources.limits.memory
`
		}
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
		if lang == "fr" {
			return `Erreur/Timeout de Résolution DNS détecté. Le Pod ne parvient pas à résoudre les noms d'hôtes internes ou externes.
Recommandation :
1. Vérifiez si les pods CoreDNS dans le namespace 'kube-system' fonctionnent correctement.
2. Vérifiez si une NetworkPolicy bloque le trafic UDP sur le port 53.
`
		}
		return `DNS Resolution Timeout/Error detected. The Pod cannot resolve external or internal hostnames.
Recommendation:
1. Check if the CoreDNS pods in the 'kube-system' namespace are running correctly.
2. Verify if a NetworkPolicy is blocking UDP port 53 traffic.
`
	}

	if strings.Contains(lowerLogs, "exit code 143") {
		if lang == "fr" {
			return `Exit Code 143 (SIGTERM Timeout) détecté. L'application a été invitée à s'arrêter proprement, mais a pris trop de temps et a été tuée de force.
Recommandation : Augmentez le paramètre terminationGracePeriodSeconds si l'application a légitimement besoin de plus de temps pour fermer ses connexions.

💡 Patch YAML Correctif :
---
spec:
  template:
    spec:
      terminationGracePeriodSeconds: 60 # Par défaut 30. Augmentez selon vos besoins.
`
		}
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
		if lang == "fr" {
			return `RunContainerError détecté. Le conteneur n'a pas pu démarrer complètement.
Recommandation :
1. Vérifiez la 'command' et les 'args' définis dans la spécification du Pod. L'exécutable n'existe peut-être pas dans l'image.
2. Vérifiez si le script d'Entrypoint a les permissions d'exécution (+x).
`
		}
		return `RunContainerError detected. The container failed to start completely.
Recommendation:
1. Verify the 'command' and 'args' defined in the Pod spec. The executable might not exist in the image path.
2. Check if the Entrypoint script has execution (+x) permissions.
`
	}

	if strings.Contains(lowerReason, "crashloopbackoff") {
		if lang == "fr" {
			return `CrashLoopBackOff détecté. L'application démarre mais plante de façon répétée.
Recommandation :
1. Vérifiez les logs ci-dessus pour repérer des traces d'erreurs (stack traces) spécifiques à l'application.
2. Assurez-vous que la sonde de vivacité (liveness probe) n'échoue pas trop rapidement.
3. Vérifiez que les variables d'environnement sont correctes.

💡 Patch YAML Correctif (Ajustement des Liveness Probes) :
---
spec:
  template:
    spec:
      containers:
      - name: mon-app
        livenessProbe:
          initialDelaySeconds: 30 # Donnez plus de temps à l'application pour démarrer
          periodSeconds: 10
`
		}
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
	llmRecommendation := askLLMForRecommendation(logs, reason, lang)
	if llmRecommendation != "" {
		if lang == "fr" {
			return fmt.Sprintf("Analyse Diagnostique IA :\n%s\n\n(Généré via LLM)", llmRecommendation)
		}
		return fmt.Sprintf("AI Diagnostic Analysis:\n%s\n\n(Generated via LLM)", llmRecommendation)
	}

	if lang == "fr" {
		return "Aucune recommandation automatisée spécifique disponible. Veuillez analyser manuellement les logs fournis ou configurer une IA."
	}
	return "No specific automated recommendation available. Please analyze the provided logs manually or configure an AI for insights."
}

// askLLMForRecommendation sends anonymized logs to an LLM for analysis.
// If using a local model like Ollama, an API key is often not required.
func askLLMForRecommendation(logs string, reason string, lang string) string {
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

	var prompt string
	var systemPrompt string

	if lang == "fr" {
		systemPrompt = "Vous êtes un expert Kubernetes. Fournissez des conseils concis et actionnables pour résoudre les problèmes du cluster en vous basant sur les logs. Fournissez des patchs YAML corrects si approprié. Vous DEVEZ impérativement répondre en Français."
		prompt = fmt.Sprintf("Agissez comme un architecte Kubernetes expert. Analysez les logs d'erreur Kubernetes suivants et fournissez une recommandation détaillée sur la façon de corriger le problème, incluant un exemple spécifique de patch YAML Kubernetes si applicable.\nRaison : %s\nLogs :\n%s", reason, anonymizedLogs)
	} else {
		systemPrompt = "You are a Kubernetes expert. Provide concise, actionable advice for fixing cluster issues based on logs. Provide correct YAML patches if appropriate. You MUST reply in English."
		prompt = fmt.Sprintf("Act as an expert Kubernetes architect. Analyze the following Kubernetes error logs and provide a detailed recommendation on how to fix it, including a specific Kubernetes YAML patch example if applicable.\nReason: %s\nLogs:\n%s", reason, anonymizedLogs)
	}

	requestBody, err := json.Marshal(map[string]interface{}{
		"model": llmModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
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

func getLang() string {
	lang := os.Getenv("LANGUAGE")
	if lang == "fr" || lang == "FR" {
		return "fr"
	}
	return "en"
}
