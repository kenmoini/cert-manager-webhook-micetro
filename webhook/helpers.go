package webhook

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsAllowedZone checks if the webhook is allowed to edit the given zone, per
// AllowedZones setting. All zones allowed if AllowedZones is empty (the default setting)
func (cfg MicetroDNSProviderConfig) IsAllowedZone(zone string) bool {
	// If no allowed zones are specified, all zones are allowed
	if len(cfg.AllowedZones) == 0 {
		return true
	}

	// Check if the zone is in the list of allowed zones, or is a subdomain of an allowed zone
	for _, allowed := range cfg.AllowedZones {
		if zone == allowed || strings.HasSuffix(zone, "."+allowed) {
			return true
		}
	}

	// If the zone is not in the list of allowed zones, return false
	return false
}

// GetAuthSecret gets the Micetro API credentials from the Kubernetes secret specified in AuthSecretRef
func (c *MicetroDNSProviderSolver) GetAuthSecret(cfg *MicetroDNSProviderConfig) (string, string, error) {

	// Check that the AuthSecretRef is specified
	if cfg.AuthSecretRef == nil {
		return "", "", fmt.Errorf("authSecretRef is not specified")
	}

	// Check that the namespace, name, and type are specified
	if cfg.AuthSecretRef.Namespace == "" || cfg.AuthSecretRef.Name == "" || cfg.AuthSecretRef.Type == "" {
		return "", "", fmt.Errorf("authSecretRef is missing required fields")
	}

	// Load the API auth secret
	sec, err := c.client.CoreV1().Secrets(cfg.AuthSecretRef.Namespace).Get(context.TODO(), cfg.AuthSecretRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed loading API auth secret %s/%s: %v", cfg.AuthSecretRef.Namespace, cfg.AuthSecretRef.Name, err)
	}
	// Read the API Auth from the secret
	if cfg.AuthSecretRef.Type == "basic" {
		usernameBytes, ok := sec.Data["username"]
		if !ok {
			return "", "", fmt.Errorf("username not found in secret \"%s/%s\"", cfg.AuthSecretRef.Namespace, cfg.AuthSecretRef.Name)
		}
		passwordBytes, ok := sec.Data["password"]
		if !ok {
			return "", "", fmt.Errorf("password not found in secret \"%s/%s\"", cfg.AuthSecretRef.Namespace, cfg.AuthSecretRef.Name)
		}
		return string(usernameBytes), string(passwordBytes), nil
	}
	if cfg.AuthSecretRef.Type == "token" {
		tokenBytes, ok := sec.Data["token"]
		if !ok {
			return "", "", fmt.Errorf("token not found in secret \"%s/%s\"", cfg.AuthSecretRef.Namespace, cfg.AuthSecretRef.Name)
		}
		return "", string(tokenBytes), nil
	}

	return "", "", fmt.Errorf("unsupported authSecretRef type \"%s\"", cfg.AuthSecretRef.Type)
}

// GetCABundle gets the Micetro CA bundle from the Kubernetes config map specified in CABundleRef
func (c *MicetroDNSProviderSolver) GetCABundle(cfg *MicetroDNSProviderConfig) ([]byte, error) {

	// Check that the CABundleRef is specified
	if cfg.CABundleRef == nil {
		return nil, fmt.Errorf("caBundleRef is not specified")
	}

	// Check that the namespace, name, and key are specified
	if cfg.CABundleRef.Namespace == "" || cfg.CABundleRef.Name == "" || cfg.CABundleRef.Key == "" {
		return nil, fmt.Errorf("caBundleRef is missing required fields")
	}

	// Load the CA bundle config map
	cm, err := c.client.CoreV1().ConfigMaps(cfg.CABundleRef.Namespace).Get(context.TODO(), cfg.CABundleRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed loading CA bundle config map %s/%s: %v", cfg.CABundleRef.Namespace, cfg.CABundleRef.Name, err)
	}

	// Read the CA bundle from the config map
	caBundleBytes, ok := cm.Data[cfg.CABundleRef.Key]
	if !ok {
		return nil, fmt.Errorf("key \"%s\" not found in config map \"%s/%s\"", cfg.CABundleRef.Key, cfg.CABundleRef.Namespace, cfg.CABundleRef.Name)
	}

	return []byte(caBundleBytes), nil
}