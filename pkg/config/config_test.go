package config

import (
	"os"
	"testing"
)

func TestLoadConfiguration(t *testing.T) {
	os.Setenv("DEBUG", "true")
	os.Setenv("KUBECONFIG", "/path/to/kubeconfig")
	os.Setenv("NAMESPACE", "default")
	os.Setenv("INGRESS_NAMESPACE", "ingress-nginx")
	os.Setenv("LABEL_SELECTOR", "app.kubernetes.io/name=ingress-nginx")

	LoadConfiguration()

	if !CFG.Debug {
		t.Errorf("Expected Debug to be true, got %v", CFG.Debug)
	}
	if CFG.KubeConfig != "/path/to/kubeconfig" {
		t.Errorf("Expected KubeConfig to be '/path/to/kubeconfig', got %v", CFG.KubeConfig)
	}
	if CFG.Namespace != "default" {
		t.Errorf("Expected Namespace to be 'default', got %v", CFG.Namespace)
	}
	if CFG.IngressNamespace != "ingress-nginx" {
		t.Errorf("Expected IngressNamespace to be 'ingress-nginx', got %v", CFG.IngressNamespace)
	}
	if CFG.IngressControllerLabel != "app.kubernetes.io/name=ingress-nginx" {
		t.Errorf("Expected IngressControllerLabel to be 'app.kubernetes.io/name=ingress-nginx', got %v", CFG.IngressControllerLabel)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	os.Setenv("TEST_ENV", "value")
	defer os.Unsetenv("TEST_ENV")

	value := getEnvOrDefault("TEST_ENV", "default")
	if value != "value" {
		t.Errorf("Expected 'value', got %v", value)
	}

	value = getEnvOrDefault("NON_EXISTENT_ENV", "default")
	if value != "default" {
		t.Errorf("Expected 'default', got %v", value)
	}
}

func TestParseEnvBool(t *testing.T) {
	os.Setenv("BOOL_ENV", "true")
	defer os.Unsetenv("BOOL_ENV")

	value := parseEnvBool("BOOL_ENV", false)
	if !value {
		t.Errorf("Expected true, got %v", value)
	}

	value = parseEnvBool("NON_EXISTENT_BOOL_ENV", true)
	if !value {
		t.Errorf("Expected true, got %v", value)
	}
}

func TestValidateRequiredConfig(t *testing.T) {
	CFG.Namespace = ""
	err := ValidateRequiredConfig()
	if err == nil {
		t.Error("Expected error for missing Namespace, got nil")
	}

	CFG.Namespace = "default"
	CFG.IngressNamespace = ""
	err = ValidateRequiredConfig()
	if err == nil {
		t.Error("Expected error for missing IngressNamespace, got nil")
	}

	CFG.IngressNamespace = "ingress-nginx"
	CFG.IngressControllerLabel = ""
	err = ValidateRequiredConfig()
	if err == nil {
		t.Error("Expected error for missing IngressControllerLabel, got nil")
	}
}
