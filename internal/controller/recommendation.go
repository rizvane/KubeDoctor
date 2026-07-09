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
		switch lang {
		case "fr":
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
		case "es":
			return `OOMKilled detectado. El contenedor excedió su límite de memoria.
Recomendación: Aumente el límite (limit) y la solicitud (request) de memoria para este contenedor en su Deployment.

💡 Parche YAML Correctivo (Ajuste los límites en consecuencia):
---
resources:
  requests:
    memory: "512Mi" # Aumentar respecto al actual
  limits:
    memory: "1Gi"   # Aumentar respecto al actual
`
		case "zh":
			return `检测到 OOMKilled。容器超出了其内存限制。
建议：在 Deployment 中增加该容器的内存限制（limit）和请求（request）。

💡 纠正性 YAML 补丁（相应地调整限制）：
---
resources:
  requests:
    memory: "512Mi" # 增加当前值
  limits:
    memory: "1Gi"   # 增加当前值
`
		default:
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
	}

	if strings.Contains(lowerReason, "imagepullbackoff") || strings.Contains(lowerReason, "errimagepull") {
		switch lang {
		case "fr":
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
		case "es":
			return `ImagePullBackOff detectado. Kubernetes no puede descargar la imagen del contenedor.
Recomendación:
1. Verifique que el nombre y la etiqueta de la imagen sean correctos.
2. Asegúrese de que el registro de imágenes sea accesible.
3. Si el registro es privado, asegúrese de que los 'imagePullSecrets' estén configurados.

💡 Parche YAML Correctivo (Si usa un registro privado):
---
spec:
  template:
    spec:
      imagePullSecrets:
      - name: mi-secreto-registro
`
		case "zh":
			return `检测到 ImagePullBackOff。Kubernetes 无法拉取容器镜像。
建议：
1. 验证镜像名称和标签是否正确。
2. 确保镜像仓库可访问。
3. 如果注册表是私有的，请确保配置了 'imagePullSecrets'。

💡 纠正性 YAML 补丁（如果使用私有注册表）：
---
spec:
  template:
    spec:
      imagePullSecrets:
      - name: my-registry-secret
`
		default:
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
	}

	if strings.Contains(lowerLogs, "connection refused") {
		switch lang {
		case "fr":
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
		case "es":
			return `Conexión rechazada detectada. La aplicación intenta comunicarse con un servicio inalcanzable.
Recomendación:
1. Verifique si el Servicio de destino se está ejecutando y tiene endpoints (kubectl get ep).
2. Asegúrese de que los números de puerto coincidan entre el Servicio y el Pod.
3. Verifique que no haya ninguna NetworkPolicy bloqueando el tráfico.
`
		case "zh":
			return `检测到连接被拒绝。应用程序试图与无法访问的服务通信。
建议：
1. 检查目标 Service 是否正在运行并且具有端点 (kubectl get ep)。
2. 确保 Service 和 Pod 之间的端口号匹配。
3. 验证 NetworkPolicies 是否未阻止流量。
`
		default:
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
	}

	if strings.Contains(lowerLogs, "no such file or directory") || strings.Contains(lowerLogs, "not found") {
		switch lang {
		case "fr":
			return `Fichier ou Répertoire introuvable détecté.
Recommandation : Vérifiez vos montages de volumes, ConfigMaps ou Secrets. Si le fichier est censé être dans l'image, vérifiez le processus de build du Dockerfile.
`
		case "es":
			return `Archivo o Directorio no encontrado detectado.
Recomendación: Verifique sus montajes de volúmenes, ConfigMaps o Secrets. Si el archivo debería estar en la imagen, verifique el proceso de compilación del Dockerfile.
`
		case "zh":
			return `检测到找不到文件或目录。
建议：验证您的卷挂载、ConfigMaps 或 Secrets。如果预期文件位于镜像内部，请验证 Dockerfile 构建过程。
`
		default:
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
	}

	if strings.Contains(lowerReason, "createcontainerconfigerror") {
		switch lang {
		case "fr":
			return `CreateContainerConfigError détecté. Le Pod ne démarre pas car un Secret ou une ConfigMap référencé(e) est manquant(e).
Recommandation : Assurez-vous que la ConfigMap ou le Secret référencé existe dans le même namespace.
`
		case "es":
			return `CreateContainerConfigError detectado. El Pod no se inicia porque falta un Secret o ConfigMap referenciado.
Recomendación: Asegúrese de que el ConfigMap o Secret referenciado exista en el mismo namespace.
`
		case "zh":
			return `检测到 CreateContainerConfigError。Pod 无法启动，因为缺少引用的 Secret 或 ConfigMap。
建议：确保引用的 ConfigMap 或 Secret 存在于同一命名空间中。
`
		default:
			return `CreateContainerConfigError detected. The Pod is failing to start because a referenced Secret or ConfigMap is missing.
Recommendation: Ensure the referenced ConfigMap or Secret exists in the same namespace.

💡 Command to check missing resources:
kubectl get configmap,secret -n <namespace>
`
		}
	}

	if strings.Contains(lowerLogs, "permission denied") {
		switch lang {
		case "fr":
			return `Erreur Permission Denied (Accès refusé) détectée. Le processus du conteneur n'a pas les privilèges nécessaires pour accéder à un fichier ou exécuter une commande.
Recommandation :
1. Vérifiez les permissions du système de fichiers là où le conteneur tente de lire/écrire.
2. Si vous utilisez des volumes persistants, assurez-vous que le 'fsGroup' correspond à l'utilisateur exécutant le processus.
`
		case "es":
			return `Error Permiso Denegado detectado. El proceso del contenedor no tiene los privilegios necesarios.
Recomendación:
1. Verifique los permisos del sistema de archivos.
2. Asegúrese de que el 'fsGroup' coincida con el usuario.
`
		case "zh":
			return `检测到权限被拒绝错误。容器进程缺乏访问文件或执行命令的必要权限。
建议：
1. 验证容器尝试读写位置的文件系统权限。
2. 确保 'fsGroup' 匹配。
`
		default:
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
	}

	if strings.Contains(lowerLogs, "no space left on device") {
		switch lang {
		case "fr":
			return `Erreur No Space Left on Device (Espace disque insuffisant) détectée. Le stockage éphémère du noeud ou le volume du conteneur est plein.
Recommandation :
1. Vérifiez si l'application écrit des logs excessifs ou des fichiers temporaires.
2. Définissez des limites 'ephemeral-storage' pour éviter l'éviction de l'intégralité du Noeud.
`
		case "es":
			return `Error No queda espacio en el dispositivo detectado.
Recomendación:
1. Verifique si la aplicación escribe demasiados registros o archivos temporales.
2. Establezca límites de 'ephemeral-storage'.
`
		case "zh":
			return `检测到设备上没有剩余空间错误。节点的临时存储或容器的卷已满。
建议：
1. 检查应用程序是否写入了过多的日志或临时文件。
2. 设置 'ephemeral-storage' 限制。
`
		default:
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
	}

	if strings.Contains(lowerLogs, "exec format error") {
		switch lang {
		case "fr":
			return `Exec Format Error détecté. L'image du conteneur a été construite pour une architecture CPU différente de celle du Noeud sur lequel elle s'exécute.
Recommandation : Reconstruisez l'image Docker pour la bonne architecture cible via 'docker buildx' ou utilisez des nodeSelectors.
`
		case "es":
			return `Error de formato de ejecución detectado. La imagen se construyó para una arquitectura diferente.
Recomendación: Recompile la imagen Docker para la arquitectura correcta.
`
		case "zh":
			return `检测到执行格式错误。容器镜像是为与运行它的节点不同的 CPU 架构构建的。
建议：使用 'docker buildx' 为正确的目标架构重新构建 Docker 镜像。
`
		default:
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
	}

	if strings.Contains(lowerLogs, "java.lang.outofmemoryerror") {
		switch lang {
		case "fr":
			return `Java OutOfMemoryError détecté. Il s'agit d'un plantage de l'espace Heap, différent d'un OOMKilled de conteneur (Exit 137).
Recommandation : Augmentez la taille max du Heap JVM pour correspondre aux limites RAM du conteneur.
`
		case "es":
			return `Java OutOfMemoryError detectado. Esto es un fallo del espacio Heap.
Recomendación: Aumente el tamaño máximo del Heap de JVM.
`
		case "zh":
			return `检测到 Java OutOfMemoryError。这是堆空间崩溃。
建议：增加 JVM 最大堆大小以匹配容器的 RAM 限制。
`
		default:
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
	}

	if strings.Contains(lowerLogs, "lookup") && strings.Contains(lowerLogs, "no such host") {
		switch lang {
		case "fr":
			return `Erreur/Timeout de Résolution DNS détecté. Le Pod ne parvient pas à résoudre les noms d'hôtes internes ou externes.
Recommandation : Vérifiez si les pods CoreDNS fonctionnent correctement ou si une NetworkPolicy bloque le port 53 UDP.
`
		case "es":
			return `Error de resolución DNS detectado. El Pod no puede resolver nombres de host.
Recomendación: Verifique CoreDNS y las NetworkPolicies.
`
		case "zh":
			return `检测到 DNS 解析超时/错误。Pod 无法解析主机名。
建议：检查 CoreDNS 和 NetworkPolicies。
`
		default:
			return `DNS Resolution Timeout/Error detected. The Pod cannot resolve external or internal hostnames.
Recommendation:
1. Check if the CoreDNS pods in the 'kube-system' namespace are running correctly.
2. Verify if a NetworkPolicy is blocking UDP port 53 traffic.
`
		}
	}

	if strings.Contains(lowerLogs, "exit code 143") {
		switch lang {
		case "fr":
			return `Exit Code 143 (SIGTERM Timeout) détecté. L'application a été invitée à s'arrêter proprement, mais a pris trop de temps et a été tuée de force.
Recommandation : Augmentez le paramètre terminationGracePeriodSeconds.
`
		case "es":
			return `Código de salida 143 (Timeout SIGTERM) detectado. La aplicación tardó demasiado en apagarse.
Recomendación: Aumente 'terminationGracePeriodSeconds'.
`
		case "zh":
			return `检测到退出代码 143 (SIGTERM 超时)。应用程序未能及时正常关闭。
建议：增加 'terminationGracePeriodSeconds'。
`
		default:
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
	}

	if strings.Contains(lowerReason, "runcontainererror") || strings.Contains(lowerReason, "starterror") {
		switch lang {
		case "fr":
			return `RunContainerError détecté. Le conteneur n'a pas pu démarrer complètement.
Recommandation : Vérifiez la 'command' et les 'args', ou vérifiez si l'Entrypoint a les permissions d'exécution (+x).
`
		case "es":
			return `RunContainerError detectado. El contenedor no pudo iniciarse por completo.
Recomendación: Verifique 'command' y 'args', o permisos (+x) en el Entrypoint.
`
		case "zh":
			return `检测到 RunContainerError。容器未能完全启动。
建议：验证 Pod 规范中定义的 'command' 和 'args'。
`
		default:
			return `RunContainerError detected. The container failed to start completely.
Recommendation:
1. Verify the 'command' and 'args' defined in the Pod spec. The executable might not exist in the image path.
2. Check if the Entrypoint script has execution (+x) permissions.
`
		}
	}

	if strings.Contains(lowerReason, "crashloopbackoff") {
		switch lang {
		case "fr":
			return `CrashLoopBackOff détecté. L'application démarre mais plante de façon répétée.
Recommandation : Vérifiez les logs ci-dessus. Assurez-vous que la sonde de vivacité (liveness probe) n'échoue pas trop rapidement.
`
		case "es":
			return `CrashLoopBackOff detectado. La aplicación se inicia pero falla repetidamente.
Recomendación: Verifique los registros. Asegúrese de que la prueba de liveness no falle demasiado rápido.
`
		case "zh":
			return `检测到 CrashLoopBackOff。应用程序启动但反复崩溃。
建议：检查日志。确保存活探针（liveness probe）不会太快失败。
`
		default:
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
	}

	// Unhandled error, attempt to use LLM if configured
	llmRecommendation := askLLMForRecommendation(logs, reason, lang)
	if llmRecommendation != "" {
		switch lang {
		case "fr":
			return fmt.Sprintf("Analyse Diagnostique IA :\n%s\n\n(Généré via LLM)", llmRecommendation)
		case "es":
			return fmt.Sprintf("Análisis Diagnóstico IA:\n%s\n\n(Generado vía LLM)", llmRecommendation)
		case "zh":
			return fmt.Sprintf("AI 诊断分析：\n%s\n\n(通过 LLM 生成)", llmRecommendation)
		default:
			return fmt.Sprintf("AI Diagnostic Analysis:\n%s\n\n(Generated via LLM)", llmRecommendation)
		}
	}

	switch lang {
	case "fr":
		return "Aucune recommandation automatisée spécifique disponible. Veuillez analyser manuellement les logs fournis ou configurer une IA."
	case "es":
		return "No hay ninguna recomendación automatizada disponible. Analice los registros manualmente o configure una IA."
	case "zh":
		return "没有具体的自动建议可用。请手动分析日志或配置 AI 分析。"
	default:
		return "No specific automated recommendation available. Please analyze the provided logs manually or configure an AI for insights."
	}
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

	switch lang {
	case "fr":
		systemPrompt = "Vous êtes un expert Kubernetes. Fournissez des conseils concis et actionnables pour résoudre les problèmes du cluster en vous basant sur les logs. Fournissez des patchs YAML corrects si approprié. Vous DEVEZ impérativement répondre en Français."
		prompt = fmt.Sprintf("Agissez comme un architecte Kubernetes expert. Analysez les logs d'erreur Kubernetes suivants et fournissez une recommandation détaillée sur la façon de corriger le problème, incluant un exemple spécifique de patch YAML Kubernetes si applicable.\nRaison : %s\nLogs :\n%s", reason, anonymizedLogs)
	case "es":
		systemPrompt = "Eres un experto en Kubernetes. Proporciona consejos concisos y aplicables para solucionar problemas del clúster basados en registros. Proporciona parches YAML correctos si es apropiado. DEBES responder imperativamente en Español."
		prompt = fmt.Sprintf("Actúa como un arquitecto experto en Kubernetes. Analiza los siguientes registros de error de Kubernetes y proporciona una recomendación detallada sobre cómo solucionarlo, incluyendo un ejemplo específico de parche YAML si es aplicable.\nRazón: %s\nRegistros:\n%s", reason, anonymizedLogs)
	case "zh":
		systemPrompt = "您是 Kubernetes 专家。根据日志提供简洁、可操作的建议来修复集群问题。如果适用，提供正确的 YAML 补丁。您必须用中文回复。"
		prompt = fmt.Sprintf("充当 Kubernetes 专家架构师。分析以下 Kubernetes 错误日志并提供关于如何修复它的详细建议，如果适用，包括特定的 Kubernetes YAML 补丁示例。\n原因：%s\n日志：\n%s", reason, anonymizedLogs)
	default:
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
	lang := strings.ToLower(os.Getenv("LANGUAGE"))
	switch lang {
	case "fr", "es", "zh", "en":
		return lang
	}
	return "en"
}
