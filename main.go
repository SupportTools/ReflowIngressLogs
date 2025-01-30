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
				logger.Warn("Pod watcher channel closed unexpectedly, restarting...")
				time.Sleep(2 * time.Second) // Prevent tight loops
				go watchIngressPods(clientset, stopCh)
				return
			}

			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				logger.Warn("Received unexpected object type in pod watcher event")
				continue
			}

			logger.Debugf("Received pod event: %s for pod %s", event.Type, pod.Name)

			switch event.Type {
			case watch.Added, watch.Modified:
				if _, exists := podLogMap.Load(pod.Name); !exists {
					logger.Infof("Starting log stream for new/updated pod: %s", pod.Name)
					go streamPodLogs(clientset, pod, config.CFG.Namespace, stopCh)
				}
			case watch.Deleted:
				logger.Infof("Pod deleted: %s, stopping log stream.", pod.Name)
				podLogMap.Delete(pod.Name)
			default:
				logger.Warnf("Unhandled event type: %s for pod %s", event.Type, pod.Name)
			}
		}
	}
}

func streamPodLogs(clientset *kubernetes.Clientset, pod *v1.Pod, namespace string, stopCh chan struct{}) {
	logger.Infof("Streaming logs for pod: %s", pod.Name)
	podLogMap.Store(pod.Name, true)

	defer func() {
		podLogMap.Delete(pod.Name)
		logger.Infof("Stopped log streaming for pod: %s", pod.Name)
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
				logger.Errorf("Error streaming logs for pod %s: %v. Retrying in 5 seconds...", pod.Name, err)
				time.Sleep(5 * time.Second)
				continue
			}

			logger.Infof("Successfully started log stream for pod: %s", pod.Name)

			scanner := bufio.NewScanner(readCloser)
			var filter string
			if config.CFG.DefaultLogFormat {
				logger.Debugf("Using default log format for pod: %s", pod.Name)
				filter = fmt.Sprintf(" [%s-", namespace)
			} else {
				logger.Debugf("Using custom log format for pod: %s", pod.Name)
				filter = fmt.Sprintf(" [namespace: %s", namespace)
			}

			logger.Debugf("Filtering logs for pod %s using pattern: %s", pod.Name, filter)

			for scanner.Scan() {
				select {
				case <-stopCh:
					logger.Infof("Stopping log stream for pod: %s", pod.Name)
					readCloser.Close()
					return
				default:
					line := scanner.Text()
					if strings.Contains(line, filter) {
						fmt.Println(line)
					}
				}
			}

			if err := scanner.Err(); err != nil {
				logger.Errorf("Error reading logs for pod %s: %v", pod.Name, err)
			}

			readCloser.Close()
			logger.Warnf("Log stream closed unexpectedly for pod %s. Restarting in 5 seconds...", pod.Name)
			time.Sleep(5 * time.Second)
		}
	}
}

func gracefulShutdown(stopCh chan struct{}) {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	<-signalCh
	logger.Info("Shutdown signal received. Cleaning up...")

	close(stopCh)

	logger.Info("Shutting down ReflowIngressLogs gracefully.")
}
