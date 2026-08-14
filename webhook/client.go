package webhook
import (
	"bytes"
	"fmt"
	"net/http"
	"crypto/tls"
	"crypto/x509"
	"golang.org/x/exp/maps"
)

// NewClient creates the HTTP Client 
func (c *MicetroDNSProviderSolver) NewClient (cfg *MicetroDNSProviderConfig) *Client {

	// Create the HTTP client
	httpClient := &http.Client{}

	// Get the Authentication credentials
	username, password, err := c.GetAuthSecret(cfg)
	if err != nil {
		return &Client{}
	}

	// Check if we have a CA bundle to use for the TLS connection
	caBundle := []byte{}
	if cfg.CABundleRef != nil {
		caBundle, err = c.GetCABundle(cfg)
		if err != nil {
			return &Client{}
		}

		certPool := x509.NewCertPool()
		if ok := certPool.AppendCertsFromPEM(caBundle); !ok {
			return &Client{}
		}

		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{
			RootCAs: certPool,
		}

		httpClient.Transport = transport
	}

	// Add request headers
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	maps.Copy(headers, cfg.Headers)

	var jsonAuthStr = []byte(fmt.Sprintf(`{"loginName":"%s", "password":"%s"}`, username, password))
	var url = fmt.Sprintf("%s/api/v1/authentication/login", cfg.Host)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonAuthStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	return &Client{}
}