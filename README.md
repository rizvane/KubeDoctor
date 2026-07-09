# 🩺 KubeDoctor

**KubeDoctor** is a production-grade Kubernetes/OpenShift Self-Healing and Diagnostic Operator built in Go using Kubebuilder/Operator SDK.

When things go wrong in a Kubernetes cluster, analyzing logs and identifying the root cause can be tedious. KubeDoctor automates this process: it acts as a silent architect, monitoring your cluster for failing Pods, fetching logs automatically, and diagnosing the root cause. It provides actionable solutions, including **corrective YAML patches**, to resolve the issue.

It also integrates seamlessly with OpenAI (or other LLMs) as a fallback for analyzing unclassified errors.

---

## 🏗️ Architecture

1. **The Controller (Manager)**
   - Monitors `corev1.Pod` events across the cluster.
   - Triggers when it detects specific failure states: `CrashLoopBackOff`, `OOMKilled`, `ImagePullBackOff`, `CreateContainerConfigError`, or non-zero exit codes.
2. **Log Retrieval & Diagnostic Analysis**
   - Retrieves the last 50 lines of logs using a native `kubernetes.Clientset` stream.
   - Passes the logs to the **Intelligent Remediation Engine**.
   - Generates a Custom Resource Definition (`DiagnosticReport`) holding the pod information, captured logs, failure reason, and a specific recommendation / YAML fix.
3. **Intelligent Remediation Engine**
   - **Local Heuristics:** Matches standard errors (OOMKilled, Image Pull issues) and generates corrective YAML patches (e.g., memory limits modifications).
   - **AI Fallback:** If `OPENAI_API_KEY` is present, it scrubs sensitive data (IPs, Emails, Tokens) using an Anonymizer and requests an AI-generated fix.
4. **Dashboard (GKE/Kubernetes Native)**
   - A lightweight Go-based UI that renders `DiagnosticReport` objects directly, perfect for developers tracking issues on GKE prior to an OpenShift integration.

---

## 🚀 Quick Start / Setup Instructions

### 1. Prerequisites
- **Kubernetes Cluster** (e.g., GKE, Minikube, Kind)
- **kubectl** configured to communicate with your cluster.
- **Go 1.21+** installed locally.
- **Operator SDK** (v1.34+)

### 2. Install the Custom Resource Definitions (CRDs)
Apply the `DiagnosticReport` CRD to your cluster:

```bash
make manifests
make install
```

*(You should see `customresourcedefinition.apiextensions.k8s.io/diagnosticreports.diagnostics.rizvane.com created`)*

### 3. Run the Operator Locally
You can run the controller locally (outside the cluster) for testing. It will use your current `~/.kube/config`.

```bash
# Optional: Set the OpenAI Key for AI-driven fallback recommendations
export OPENAI_API_KEY="sk-your-key..."

make run
```

### 4. Deploying to the Cluster
To deploy the operator as a Pod inside your cluster:

```bash
# Build the Docker image
make docker-build docker-push IMG=<some-registry>/kubedoctor:v0.1.0

# Deploy the operator to the cluster
make deploy IMG=<some-registry>/kubedoctor:v0.1.0
```

---

## 📊 The KubeDoctor Dashboard

KubeDoctor comes with a portable, lightweight web dashboard designed specifically for native Kubernetes (GKE).

### Run the Dashboard Locally
Make sure you are authenticated to your cluster (`KUBECONFIG`), then run:

```bash
go build -o bin/dashboard dashboard/main.go
./bin/dashboard
```

Access the dashboard at `http://localhost:8082`.

### Deploying the Dashboard on Kubernetes
You can containerize the `dashboard/main.go` file and deploy it behind a standard Kubernetes `Service` (NodePort/LoadBalancer) or `Ingress` to give your developers a real-time view of all cluster faults.

---

## 🧠 Recommendation Engine Details

### Standard Debugging (No LLM Required)
The core of KubeDoctor does not require an LLM. It features a robust heuristic engine. Example fixes include:

* **OOMKilled (Exit Code 137)**
  * *Diagnosis:* Container exceeded memory limit.
  * *Patch:* Recommends a YAML snippet adjusting `resources.limits.memory` and `resources.requests.memory`.
* **ImagePullBackOff**
  * *Diagnosis:* Docker image not found or access denied.
  * *Patch:* Recommends adding `imagePullSecrets` to the Pod Spec.
* **Connection Refused**
  * *Diagnosis:* Networking failure.
  * *Patch:* Advises checking Service definitions, ports, and NetworkPolicies.
* **Missing ConfigMaps/Secrets (CreateContainerConfigError)**
  * *Diagnosis:* Missing referenced configurations.
  * *Patch:* Reminds you to verify `volumeMounts` and Secret presence.

### AI Debugging (Fallback)
If an unknown application crash occurs, the AI engine takes over.
* Set the `OPENAI_API_KEY` environment variable on the Operator Pod.
* **Security First:** The operator scrubs IP addresses, email addresses, and `Bearer <token>` strings from logs before transmission.

---

## 🛠️ Generating the OLM Bundle (OperatorHub)

For OpenShift and OperatorHub integrations, KubeDoctor is configured to generate an Operator Lifecycle Manager (OLM) bundle.

```bash
make bundle
```
This generates the CSV (ClusterServiceVersion) in `bundle/manifests/app.clusterserviceversion.yaml`, making KubeDoctor production-ready and distributable on enterprise catalogs.

---

## 🛡️ RBAC Permissions
The operator automatically requests the following permissions via Kubebuilder tags:
- `pods` (get, list, watch)
- `pods/log` (get) - *Crucial for fetching container logs dynamically.*
- `diagnosticreports` (get, list, watch, create, update, patch, delete)
- `diagnosticreports/status` (get, update, patch)