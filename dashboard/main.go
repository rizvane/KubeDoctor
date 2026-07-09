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

func main() {
	config, err := getKubeConfig()
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating dynamic client: %s", err.Error())
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleDashboard(w, r, client)
	})

	log.Println("Starting KubeDoctor Dashboard on :8082")
	if err := http.ListenAndServe(":8082", nil); err != nil {
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

	if err := tmpl.Execute(w, reports); err != nil {
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
