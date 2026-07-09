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
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	diagnosticsv1alpha1 "github.com/rizvane/KubeDoctor/api/v1alpha1"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get
//+kubebuilder:rbac:groups=diagnostics.rizvane.com,resources=diagnosticreports,verbs=get;list;watch;create;update;patch;delete

func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// We only care about pods that are in a failed state or have crashing containers
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodRunning && isPodHealthy(&pod) {
		return ctrl.Result{}, nil
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		state := containerStatus.State
		if state.Waiting != nil && (state.Waiting.Reason == "CrashLoopBackOff" || state.Waiting.Reason == "CreateContainerError") ||
			state.Terminated != nil && state.Terminated.ExitCode != 0 {

			reason := getErrorReason(containerStatus)

			// Check if we already have a DiagnosticReport for this pod to prevent spamming
			reportName := fmt.Sprintf("diag-%s-%s", pod.Name, containerStatus.Name)

			var existingReport diagnosticsv1alpha1.DiagnosticReport
			err := r.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: reportName}, &existingReport)
			if err == nil {
				// Report already exists, skip
				continue
			} else if !errors.IsNotFound(err) {
				log.Error(err, "Failed to get DiagnosticReport")
				return ctrl.Result{}, err
			}

			log.Info("Detected failing pod container, creating DiagnosticReport", "Pod", pod.Name, "Container", containerStatus.Name, "Reason", reason)

			// Fetch logs using kubernetes.Clientset
			logs, err := r.getPodLogs(ctx, pod.Namespace, pod.Name, containerStatus.Name)
			if err != nil {
				log.Error(err, "Failed to fetch pod logs")
				logs = "Failed to fetch logs: " + err.Error()
			}

			// Generate Recommendation
			recommendation := GenerateRecommendation(logs, reason)

			// Create DiagnosticReport
			report := &diagnosticsv1alpha1.DiagnosticReport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      reportName,
					Namespace: pod.Namespace,
					Labels: map[string]string{
						"pod":       pod.Name,
						"container": containerStatus.Name,
					},
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(&pod, corev1.SchemeGroupVersion.WithKind("Pod")),
					},
				},
				Spec: diagnosticsv1alpha1.DiagnosticReportSpec{
					Namespace:     pod.Namespace,
					PodName:       pod.Name,
					ContainerName: containerStatus.Name,
				},
				Status: diagnosticsv1alpha1.DiagnosticReportStatus{
					Phase:          "Completed",
					Reason:         reason,
					Logs:           logs,
					Recommendation: recommendation,
				},
			}

			if err := r.Create(ctx, report); err != nil {
				log.Error(err, "Failed to create DiagnosticReport")
				return ctrl.Result{}, err
			}

			// We need to update the status subresource after creation
			report.Status = diagnosticsv1alpha1.DiagnosticReportStatus{
				Phase:          "Completed",
				Reason:         reason,
				Logs:           logs,
				Recommendation: recommendation,
			}
			if err := r.Status().Update(ctx, report); err != nil {
				log.Error(err, "Failed to update DiagnosticReport status")
				return ctrl.Result{}, err
			}

			log.Info("Successfully created DiagnosticReport", "Name", reportName)
		}
	}

	return ctrl.Result{}, nil
}

func isPodHealthy(pod *corev1.Pod) bool {
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil || (status.State.Terminated != nil && status.State.Terminated.ExitCode != 0) {
			return false
		}
	}
	return true
}

func getErrorReason(status corev1.ContainerStatus) string {
	if status.State.Waiting != nil {
		return status.State.Waiting.Reason
	}
	if status.State.Terminated != nil {
		return status.State.Terminated.Reason
	}
	return "Unknown"
}

func (r *PodReconciler) getPodLogs(ctx context.Context, namespace, podName, containerName string) (string, error) {
	tailLines := int64(50)
	req := r.Clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
		Previous:  true, // Fetch previous container logs if it crashed
	})

	podLogs, err := req.Stream(ctx)
	if err != nil {
		// Try without Previous if it fails (e.g. no previous container)
		req = r.Clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
			Container: containerName,
			TailLines: &tailLines,
		})
		podLogs, err = req.Stream(ctx)
		if err != nil {
			return "", err
		}
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
