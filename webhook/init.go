package webhook

import (
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func (c *MicetroDNSProviderSolver) init(config *apiextensionsv1.JSON, namespace string) (*Client, *MicetroDNSProviderConfig, error) {
	// Load and validate the configuration
	cfg, err := loadConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed parsing provider config: %v", err)
	}

	if err := c.validate(cfg); err != nil {
		return nil, nil, fmt.Errorf("failed validating config: %v", err)
	}

	return c.NewClient(cfg), cfg, nil
}