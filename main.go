package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/supporttools/ReflowIngressLogs/pkg/config"
	"github.com/supporttools/ReflowIngressLogs/pkg/k8s"
	"github.com/supporttools/ReflowIngressLogs/pkg/logging"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

var (
	logger    = logging.SetupLogging()
	clientset *kubernetes.Clientset
	podLogMap sync.Map // Tracks active log stream goroutines
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

	// Channel to signal when to stop goroutines
	stopCh := make(chan struct{})
	defer close(stopCh)

	// Start watching for pod events
	go watchIngressPods(clientset, stopCh)

	// Handle graceful shutdown
	gracefulShutdown(stopCh)
}

func watchIngressPods(clientset *kubernetes.Clientset, stopCh chan struct{}) {
	logger.Info("Watching ingress controller pods for changes...")

	watcher, err := clientset.CoreV1().Pods(config.CFG.IngressNamespace).Watch(context.TODO(), metav1.ListOptions{
		LabelSelector: config.CFG.IngressControllerLabel,
	})
	if err != nil {
		logger.Fatalf("Error setting up pod watcher: %v", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-stopCh:
			logger.Info("Stopping pod watcher...")
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				logger.Warn("Pod watcher channel closed, restarting...")
				time.Sleep(2 * time.Second) // Prevent tight loops
				go watchIngressPods(clientset, stopCh)
				return
			}

			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				continue
			}

			switch event.Type {
			case watch.Added, watch.Modified:
				// Start log streaming if the pod is not already being watched
				if _, exists := podLogMap.Load(pod.Name); !exists {
					logger.Infof("Starting log stream for new/updated pod: %s", pod.Name)
					go streamPodLogs(clientset, pod, config.CFG.Namespace, stopCh)
				}
			case watch.Deleted:
				// Stop log streaming for deleted pods
				logger.Infof("Pod deleted: %s, stopping log stream.", pod.Name)
				podLogMap.Delete(pod.Name)
			}
		}
	}
}

func streamPodLogs(clientset *kubernetes.Clientset, pod *v1.Pod, namespace string, stopCh chan struct{}) {
	logger.Infof("Streaming logs for pod: %s", pod.Name)
	podLogMap.Store(pod.Name, true) // Mark pod as being streamed

	defer func() {
		podLogMap.Delete(pod.Name) // Cleanup when log streaming stops
	}()

	for {
		select {
		case <-stopCh:
			logger.Infof("Stopping log stream for pod: %s", pod.Name)
			return
		default:
			req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{Follow: true})
			readCloser, err := req.Stream(context.TODO())
			if err != nil {
				logger.Errorf("Error streaming logs for pod %s: %v", pod.Name, err)
				time.Sleep(5 * time.Second) // Retry after a short delay
				continue
			}

			scanner := bufio.NewScanner(readCloser)
			filter := fmt.Sprintf(" [%s-", namespace) // Filter pattern for the namespace

			for scanner.Scan() {
				select {
				case <-stopCh:
					logger.Infof("Stopping log stream for pod: %s", pod.Name)
					readCloser.Close()
					return
				default:
					line := scanner.Text()
					if strings.Contains(line, filter) {
						logger.Infof("[%s] %s", pod.Name, line)
					}
				}
			}

			if err := scanner.Err(); err != nil {
				logger.Errorf("Error reading logs for pod %s: %v", pod.Name, err)
			}

			readCloser.Close()
			time.Sleep(5 * time.Second) // Prevent tight retry loops
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

	// Wait a moment for cleanup
	logger.Info("Shutting down ReflowIngressLogs gracefully.")
}
