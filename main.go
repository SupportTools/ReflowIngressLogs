package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/supporttools/ReflowIngressLogs/pkg/config"
	"github.com/supporttools/ReflowIngressLogs/pkg/k8s"
	"github.com/supporttools/ReflowIngressLogs/pkg/logging"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	logger    = logging.SetupLogging()
	clientset *kubernetes.Clientset
)

func main() {
	logger.Info("Starting ReflowIngressLogs")

	// Load and validate configuration
	logger.Info("Loading configuration...")
	config.LoadConfiguration()
	if err := config.ValidateRequiredConfig(); err != nil {
		logger.Fatalf("Configuration validation failed: %v", err)
	}
	logger.Info("Configuration loaded and validated successfully.")

	// Connect to Kubernetes
	logger.Info("Connecting to Kubernetes...")
	var err error
	clientset, err = k8s.ConnectToK8s()
	if err != nil {
		logger.Fatalf("Failed to connect to Kubernetes: %v", err)
	}
	logger.Info("Connected to Kubernetes successfully.")

	// Get ingress-nginx-controller pods
	pods, err := clientset.CoreV1().Pods(config.CFG.IngressNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: config.CFG.IngressControllerLabel,
	})
	if err != nil {
		logger.Errorf("Error listing pods: %v", err)
		os.Exit(1)
	}

	if len(pods.Items) == 0 {
		logger.Warn("No ingress controller pods found.")
		return
	}

	// Channel to signal when to stop goroutines
	stopCh := make(chan struct{})
	defer close(stopCh)

	// Stream logs for each pod
	for _, pod := range pods.Items {
		go func(pod v1.Pod) {
			streamPodLogs(clientset, pod, config.CFG.Namespace, stopCh)
		}(pod)
	}

	// Handle graceful shutdown
	gracefulShutdown(stopCh)
}

func streamPodLogs(clientset *kubernetes.Clientset, pod v1.Pod, namespace string, stopCh chan struct{}) {
	logger.Infof("Streaming logs for pod: %s", pod.Name)

	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{
		Follow: true,
	})

	readCloser, err := req.Stream(context.TODO())
	if err != nil {
		logger.Errorf("Error streaming logs for pod %s: %v", pod.Name, err)
		return
	}
	defer readCloser.Close()

	scanner := bufio.NewScanner(readCloser)
	filter := fmt.Sprintf(" [%s-", namespace) // Filter pattern for the namespace

	for {
		select {
		case <-stopCh:
			logger.Infof("Stopping log stream for pod: %s", pod.Name)
			return
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, filter) {
					logger.Infof("[%s] %s", pod.Name, line)
				}
			} else if err := scanner.Err(); err != nil {
				logger.Errorf("Error reading logs for pod %s: %v", pod.Name, err)
				return
			}
		}
	}
}

func gracefulShutdown(stopCh chan struct{}) {
	// Create a channel to listen for OS signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	<-signalCh
	logger.Info("Shutdown signal received. Cleaning up...")

	// Signal all goroutines to stop
	close(stopCh)

	// Wait a moment for cleanup (if needed)
	logger.Info("Shutting down ReflowIngressLogs gracefully.")
}
