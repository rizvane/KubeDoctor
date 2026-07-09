package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var gvr = schema.GroupVersionResource{
	Group:    "diagnostics.rizvane.com",
	Version:  "v1alpha1",
	Resource: "diagnosticreports",
}

// Report struct to bind with the Go template
type Report struct {
	Namespace      string
	PodName        string
	ContainerName  string
	Phase          string
	Reason         string
	Logs           string
	Recommendation string
	CreationTime   string
}

// DashboardConfig holds the configuration for the UI
type DashboardConfig struct {
	Title       string
	RefreshRate string
}

// Translation holds localized strings for the UI
type Translation struct {
	Subtitle     string
	NoData       string
	Pod          string
	Container    string
	Reason       string
	Created      string
	CapturedLogs string
}

func main() {
	config, err := getKubeConfig()
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating dynamic client: %s", err.Error())
	}

	port := os.Getenv("DASHBOARD_PORT")
	if port == "" {
		port = "8082"
	}

	// Serve static assets (like the logo)
	fs := http.FileServer(http.Dir("dashboard/assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleDashboard(w, r, client)
	})

	log.Printf("Starting KubeDoctor Dashboard on :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleDashboard(w http.ResponseWriter, r *http.Request, client dynamic.Interface) {
	reportsList, err := client.Resource(gvr).Namespace("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		http.Error(w, "Failed to list DiagnosticReports: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var reports []Report
	for _, item := range reportsList.Items {
		spec, ok := item.Object["spec"].(map[string]interface{})
		if !ok {
			continue
		}
		status, ok := item.Object["status"].(map[string]interface{})
		if !ok {
			status = map[string]interface{}{}
		}

		reports = append(reports, Report{
			Namespace:      getString(spec, "namespace"),
			PodName:        getString(spec, "podName"),
			ContainerName:  getString(spec, "containerName"),
			Phase:          getString(status, "phase"),
			Reason:         getString(status, "reason"),
			Logs:           getString(status, "logs"),
			Recommendation: getString(status, "recommendation"),
			CreationTime:   item.GetCreationTimestamp().String(),
		})
	}

	tmpl, err := template.ParseFiles("dashboard/index.html")
	if err != nil {
		http.Error(w, "Failed to parse template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	lang := os.Getenv("LANGUAGE")
	var t Translation
	if lang == "fr" || lang == "FR" {
		t = Translation{
			Subtitle:     "Rapports de Diagnostic et d'Auto-Guérison",
			NoData:       "Aucun rapport de diagnostic trouvé. Votre cluster est en parfaite santé ! 🎉",
			Pod:          "Pod",
			Container:    "Conteneur",
			Reason:       "Raison",
			Created:      "Créé le",
			CapturedLogs: "Logs capturés :",
		}
	} else {
		t = Translation{
			Subtitle:     "Diagnostics & Self-Healing Reports",
			NoData:       "No Diagnostic Reports found. Your cluster is healthy! 🎉",
			Pod:          "Pod",
			Container:    "Container",
			Reason:       "Reason",
			Created:      "Created",
			CapturedLogs: "Captured Logs:",
		}
	}

	uiConfig := DashboardConfig{
		Title:       getEnvOrDefault("DASHBOARD_TITLE", "KubeDoctor Dashboard"),
		RefreshRate: getEnvOrDefault("DASHBOARD_REFRESH_RATE_SECONDS", "30"),
	}

	data := struct {
		Config  DashboardConfig
		Reports []Report
		T       Translation
	}{
		Config:  uiConfig,
		Reports: reports,
		T:       t,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
	}
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getEnvOrDefault(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

func getKubeConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}
