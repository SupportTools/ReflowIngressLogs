// pkg/k8s/k8s.go
package k8s

import (
	"fmt"
	"os"

	"github.com/supporttools/ReflowIngressLogs/pkg/config"
	"github.com/supporttools/ReflowIngressLogs/pkg/logging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var log = logging.SetupLogging()

// ConnectToK8s connects to a Kubernetes cluster by checking the environment and configuration settings.
func ConnectToK8s() (*kubernetes.Clientset, error) {
	var kubeConfig *rest.Config
	var err error

	// Attempt to use in-cluster configuration
	log.Debug("Attempting to connect using in-cluster configuration...")
	kubeConfig, err = rest.InClusterConfig()
	if err == nil {
		log.Info("Successfully obtained in-cluster configuration.")
		clientset, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			log.Errorf("Failed to create Kubernetes client from in-cluster config: %v", err)
			return nil, err
		}
		log.Debug("Successfully created Kubernetes client using in-cluster configuration.")
		return clientset, nil
	}
	log.Warnf("In-cluster configuration failed: %v. Attempting to use KUBECONFIG.", err)

	// Check if KUBECONFIG is set in the environment
	envKubeConfig := os.Getenv("KUBECONFIG")
	if envKubeConfig != "" {
		log.Debugf("Environment variable KUBECONFIG is set: %s", envKubeConfig)
	}

	// Attempt to use KUBECONFIG from config or environment
	cfgKubeConfig := config.CFG.KubeConfig
	if cfgKubeConfig == "" && envKubeConfig != "" {
		cfgKubeConfig = envKubeConfig
	}

	if cfgKubeConfig != "" {
		log.Debugf("Using KUBECONFIG: %s", cfgKubeConfig)
		log.Debug("Attempting to build configuration from KUBECONFIG...")
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", cfgKubeConfig)
		if err == nil {
			log.Info("Successfully loaded configuration from KUBECONFIG.")
			clientset, err := kubernetes.NewForConfig(kubeConfig)
			if err != nil {
				log.Errorf("Failed to create Kubernetes client from KUBECONFIG: %v", err)
				return nil, err
			}
			log.Debug("Successfully created Kubernetes client using KUBECONFIG.")
			return clientset, nil
		}
		log.Errorf("Failed to load configuration from KUBECONFIG (%s): %v", cfgKubeConfig, err)
	} else {
		log.Debug("No KUBECONFIG path is provided.")
	}

	// All connection attempts failed
	log.Error("All attempts to configure Kubernetes client failed. Ensure the environment or KUBECONFIG is set correctly.")
	return nil, fmt.Errorf("failed to configure Kubernetes client: %v", err)
}
