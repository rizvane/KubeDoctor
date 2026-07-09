# 🩺 KubeDoctor

**KubeDoctor** est un Opérateur de Self-Healing et de Diagnostic pour Kubernetes et OpenShift, conçu pour la production et développé en Go avec Kubebuilder / Operator SDK.

Lorsque des problèmes surviennent dans un cluster Kubernetes, analyser les logs et identifier la cause première peut être fastidieux. KubeDoctor automatise ce processus : il agit comme un architecte silencieux, surveillant votre cluster pour détecter les Pods en échec, récupérant automatiquement les logs et diagnostiquant la cause du problème. Il propose des solutions concrètes, y compris des **correctifs sous forme de patchs YAML**, pour résoudre l'incident.

Il s'intègre également parfaitement avec OpenAI (ou d'autres LLM) en tant que solution de secours pour analyser les erreurs non classifiées.

---

## 🏗️ Architecture

1. **Le Contrôleur (Manager)**
   - Surveille les événements des `corev1.Pod` à travers le cluster.
   - Se déclenche lorsqu'il détecte des états d'échec spécifiques : `CrashLoopBackOff`, `OOMKilled`, `ImagePullBackOff`, `CreateContainerConfigError`, ou des codes de sortie non nuls.
2. **Récupération des Logs & Analyse Diagnostique**
   - Récupère les 50 dernières lignes de logs en utilisant un flux (stream) natif `kubernetes.Clientset`.
   - Transmet les logs au **Moteur de Remédiation Intelligent**.
   - Génère une Ressource Personnalisée (CRD `DiagnosticReport`) contenant les informations du Pod, les logs capturés, la raison de l'échec, et une recommandation spécifique / correctif YAML.
3. **Moteur de Remédiation Intelligent**
   - **Heuristiques Locales :** Fait correspondre les erreurs standards (OOMKilled, problèmes de Pull d'image) et génère des correctifs YAML (ex: modification des limites de mémoire).
   - **Fallback IA (Secours) :** Si la variable `OPENAI_API_KEY` est présente, le moteur nettoie les données sensibles (IPs, Emails, Tokens) via un Anonymiseur et demande une solution générée par l'IA.
4. **Dashboard (GKE / Natif Kubernetes)**
   - Une interface utilisateur légère codée en Go qui affiche directement les objets `DiagnosticReport`, idéale pour les développeurs souhaitant suivre les problèmes sur GKE avant une éventuelle migration vers OpenShift.

---

## 🚀 Démarrage Rapide / Instructions d'Installation

### 1. Prérequis
- **Un Cluster Kubernetes** (ex: GKE, Minikube, Kind)
- **kubectl** configuré pour communiquer avec votre cluster.
- **Go 1.21+** installé localement.
- **Operator SDK** (v1.34+)

### 2. Installer les CRD (Custom Resource Definitions)
Appliquez la CRD `DiagnosticReport` sur votre cluster :

```bash
make manifests
make install
```

*(Vous devriez voir le message `customresourcedefinition.apiextensions.k8s.io/diagnosticreports.diagnostics.rizvane.com created`)*

### 3. Lancer l'Opérateur Localement
Vous pouvez lancer le contrôleur localement (en dehors du cluster) pour effectuer des tests. Il utilisera votre configuration `~/.kube/config` actuelle.

```bash
# Optionnel : Configurer la clé OpenAI pour les recommandations de secours gérées par l'IA
export OPENAI_API_KEY="sk-votre-cle..."

make run
```

### 4. Déployer sur le Cluster
Pour déployer l'opérateur en tant que Pod à l'intérieur de votre cluster :

```bash
# Construire l'image Docker
make docker-build docker-push IMG=<votre-registre>/kubedoctor:v0.1.0

# Déployer l'opérateur sur le cluster
make deploy IMG=<votre-registre>/kubedoctor:v0.1.0
```

---

## 📊 Le Dashboard KubeDoctor

KubeDoctor est fourni avec un tableau de bord web léger et portable, conçu spécifiquement pour du Kubernetes natif (comme GKE).

### Lancer le Dashboard Localement
Assurez-vous d'être authentifié sur votre cluster (`KUBECONFIG`), puis lancez :

```bash
go build -o bin/dashboard dashboard/main.go
./bin/dashboard
```

Accédez au tableau de bord sur `http://localhost:8082`.

### Déployer le Dashboard sur Kubernetes
Vous pouvez conteneuriser le fichier `dashboard/main.go` et le déployer derrière un `Service` Kubernetes standard (NodePort/LoadBalancer) ou une `Ingress` pour offrir à vos développeurs une vue en temps réel de tous les incidents du cluster.

---

## 🧠 Détails du Moteur de Recommandation

### Débogage Standard (Sans LLM)
Le cœur de KubeDoctor ne nécessite pas de LLM. Il est doté d'un moteur heuristique robuste. Voici quelques exemples de corrections automatiques :

* **OOMKilled (Code de Sortie 137)**
  * *Diagnostic :* Le conteneur a dépassé sa limite de mémoire.
  * *Patch :* Recommande un extrait YAML pour ajuster `resources.limits.memory` et `resources.requests.memory`.
* **ImagePullBackOff**
  * *Diagnostic :* L'image Docker est introuvable ou l'accès est refusé.
  * *Patch :* Recommande l'ajout de `imagePullSecrets` à la spécification du Pod.
* **Connection Refused (Connexion Refusée)**
  * *Diagnostic :* Échec réseau.
  * *Patch :* Conseille de vérifier les définitions de Service, les ports, et les NetworkPolicies.
* **ConfigMaps/Secrets Manquants (CreateContainerConfigError)**
  * *Diagnostic :* Configurations référencées manquantes.
  * *Patch :* Rappelle de vérifier les `volumeMounts` et la présence du Secret.

### Débogage IA (Fallback / Secours)
Si l'application plante pour une raison inconnue, le moteur d'IA prend le relais.
* Configurez la variable d'environnement `OPENAI_API_KEY` sur le Pod de l'opérateur.
* **La Sécurité Avant Tout :** L'opérateur supprime les adresses IP, les adresses email et les chaînes `Bearer <token>` des logs avant toute transmission à l'API externe.

---

## 🛠️ Génération du Bundle OLM (OperatorHub)

Pour les intégrations avec OpenShift et OperatorHub, KubeDoctor est configuré pour générer un bundle OLM (Operator Lifecycle Manager).

```bash
make bundle
```
Cela génère le CSV (ClusterServiceVersion) dans `bundle/manifests/app.clusterserviceversion.yaml`, rendant KubeDoctor prêt pour la production et distribuable sur des catalogues d'entreprise.

---

## 🛡️ Permissions RBAC
L'opérateur demande automatiquement les permissions suivantes via les tags Kubebuilder :
- `pods` (get, list, watch)
- `pods/log` (get) - *Crucial pour récupérer dynamiquement les logs des conteneurs.*
- `diagnosticreports` (get, list, watch, create, update, patch, delete)
- `diagnosticreports/status` (get, update, patch)