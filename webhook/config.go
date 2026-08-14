package webhook

import (
	"fmt"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"encoding/json"
)

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (*MicetroDNSProviderConfig, error) {
	cfg := &MicetroDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

// validate ensures the needed fields are present in the configuration and returns an error if any are missing.
func (c *MicetroDNSProviderSolver) validate(cfg *MicetroDNSProviderConfig) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	
	if cfg.Host == "" {
		return fmt.Errorf("host is required in the configuration")
	}

	if cfg.AuthSecretRef == nil {
		return fmt.Errorf("authSecretRef is required in the configuration")
	}

	if cfg.AuthSecretRef.Namespace == "" || cfg.AuthSecretRef.Name == "" || cfg.AuthSecretRef.Type == "" {
		return fmt.Errorf("authSecretRef must have namespace, name, and type specified")
	}

	if cfg.Version == "" {
		return fmt.Errorf("version is required in the configuration")
	}

	return nil
}
