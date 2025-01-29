package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// AppConfig defines the structure for application configuration loaded from environment variables.
type AppConfig struct {
	Debug                  bool   `json:"debug"`                  // Enable debug logging
	KubeConfig             string `json:"kubeConfig"`             // Path to kubeconfig file
	Namespace              string `json:"namespace"`              // Namespace for filtering logs
	IngressNamespace       string `json:"ingressNamespace"`       // Namespace of ingress-nginx controllers
	IngressControllerLabel string `json:"ingressControllerLabel"` // Label selector for ingress-nginx pods
	DefaultLogFormat       bool   `json:"defaultLogFormat"`       // Use NGINX default log format
}

// CFG is the global configuration instance.
var CFG AppConfig

// LoadConfiguration loads the configuration from environment variables and validates it.
func LoadConfiguration() {
	// Load environment variables with defaults
	CFG.Debug = parseEnvBool("DEBUG", false)
	CFG.KubeConfig = getEnvOrDefault("KUBECONFIG", "")
	CFG.Namespace = getEnvOrDefault("NAMESPACE", "")
	CFG.IngressNamespace = getEnvOrDefault("INGRESS_NAMESPACE", "ingress-nginx")
	CFG.IngressControllerLabel = getEnvOrDefault("LABEL_SELECTOR", "app.kubernetes.io/name=ingress-nginx")
	CFG.DefaultLogFormat = parseEnvBool("DEFAULT_LOG_FORMAT", true)

	// Validate the loaded configuration
	if err := ValidateRequiredConfig(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Log loaded configuration in debug mode
	if CFG.Debug {
		log.Printf("Configuration Loaded: %+v", CFG)
	}
}

// getEnvOrDefault retrieves the value of an environment variable or returns a default value if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Environment variable %s not set. Using default: %s", key, defaultValue)
	return defaultValue
}

// parseEnvBool parses an environment variable as a boolean, supporting common truthy/falsy values.
func parseEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Printf("Environment variable %s not set. Using default: %t", key, defaultValue)
		return defaultValue
	}

	// Normalize and parse the value
	value = strings.ToLower(value)
	switch value {
	case "1", "true", "yes", "on", "enabled":
		return true
	case "0", "false", "no", "off", "disabled":
		return false
	default:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Error parsing %s as bool: %v. Using default value: %t", key, err, defaultValue)
			return defaultValue
		}
		return boolValue
	}
}

// ValidateRequiredConfig ensures all required configuration values are set.
func ValidateRequiredConfig() error {
	if CFG.Namespace == "" {
		return fmt.Errorf("NAMESPACE is required but not set")
	}
	if CFG.IngressNamespace == "" {
		return fmt.Errorf("INGRESS_NAMESPACE is required but not set")
	}
	if CFG.IngressControllerLabel == "" {
		return fmt.Errorf("LABEL_SELECTOR is required but not set")
	}
	return nil
}
