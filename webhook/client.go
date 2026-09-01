package webhook

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// NewClient creates an authenticated Micetro API client.
func (c *MicetroDNSProviderSolver) NewClient(cfg *MicetroDNSProviderConfig) (*Client, error) {
	httpClient := &http.Client{}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30
	}
	httpClient.Timeout = time.Duration(timeout) * time.Second

	if cfg.CABundleRef != nil {
		caBundle, err := c.GetCABundle(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed loading CA bundle: %v", err)
		}
		certPool := x509.NewCertPool()
		if ok := certPool.AppendCertsFromPEM(caBundle); !ok {
			return nil, fmt.Errorf("failed to parse CA bundle certificates")
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{
			RootCAs: certPool,
		}
		httpClient.Transport = transport
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	maps.Copy(headers, cfg.Headers)

	baseURL := strings.TrimRight(cfg.Host, "/") + DefaultMicetroAPIBasePath

	client := &Client{
		BaseURL:    baseURL,
		Headers:    headers,
		httpClient: httpClient,
	}

	username, password, err := c.GetAuthSecret(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed reading auth credentials: %v", err)
	}

	if cfg.AuthSecretRef.Type == "token" {
		client.token = password
	} else {
		if err := client.authenticate(username, password); err != nil {
			return nil, fmt.Errorf("failed authenticating with Micetro API: %v", err)
		}
	}

	return client, nil
}

func (client *Client) authenticate(username, password string) (retErr error) {
	authReq := micetroAuthRequest{
		LoginName: username,
		Password:  password,
	}

	body, err := json.Marshal(authReq)
	if err != nil {
		return fmt.Errorf("failed marshaling auth request: %v", err)
	}

	url := client.BaseURL + DefaultMicetroAPIAuthPath
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed creating auth request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("error closing response body: %v", closeErr)
		}
	}()

	if err := checkResponse(resp); err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	var authResp micetroAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed decoding auth response: %v", err)
	}

	if authResp.Session == "" {
		return fmt.Errorf("no session token returned from authentication")
	}

	client.token = authResp.Session
	return nil
}

func (client *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	url := client.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed marshaling request body: %v", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+client.token)
	for k, v := range client.Headers {
		req.Header.Set(k, v)
	}

	return client.httpClient.Do(req)
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unexpected status %d (failed to read response body: %v)", resp.StatusCode, err)
	}

	var apiErr micetroAPIError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
		return &apiErr
	}

	return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
}

// FindZone finds a DNS zone by name in Micetro.
func (client *Client) FindZone(zoneName string, dnsViewRef string) (_ *micetroZone, retErr error) {
	filterName := strings.TrimSuffix(zoneName, ".")

	path := "/dnsZones?filter=" + filterName
	if dnsViewRef != "" {
		path += "&dnsViewRef=" + dnsViewRef
	}

	resp, err := client.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed listing zones: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("error closing response body: %v", closeErr)
		}
	}()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var result micetroZonesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed decoding zones response: %v", err)
	}

	for i := range result.DNSZones {
		z := &result.DNSZones[i]
		if z.Name == zoneName || z.Name == filterName {
			return z, nil
		}
	}

	return nil, fmt.Errorf("zone %q not found in Micetro", zoneName)
}

// FindViewRef resolves a DNS view name to its object reference.
func (client *Client) FindViewRef(viewName string) (_ string, retErr error) {
	if viewName == "" {
		return "", nil
	}

	path := "/dnsViews?filter=" + viewName

	resp, err := client.doRequest("GET", path, nil)
	if err != nil {
		return "", fmt.Errorf("failed listing views: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("error closing response body: %v", closeErr)
		}
	}()

	if err := checkResponse(resp); err != nil {
		return "", err
	}

	var result micetroViewsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed decoding views response: %v", err)
	}

	for _, v := range result.DNSViews {
		if v.Name == viewName {
			return v.Ref, nil
		}
	}

	return "", fmt.Errorf("DNS view %q not found in Micetro", viewName)
}

// CreateTXTRecord creates a TXT DNS record in the specified zone.
func (client *Client) CreateTXTRecord(zoneRef, recordName, recordData string, ttl int) (retErr error) {
	path := "/dnsZones/" + zoneRef + "/dnsRecords"

	reqBody := micetroCreateRecordRequest{
		DNSRecord: micetroRecord{
			Name:    recordName,
			Type:    "TXT",
			Data:    recordData,
			TTL:     strconv.Itoa(ttl),
			Enabled: true,
			Comment: "cert-manager DNS-01 challenge",
		},
		SaveComment: "cert-manager ACME DNS-01 challenge record",
	}

	resp, err := client.doRequest("POST", path, reqBody)
	if err != nil {
		return fmt.Errorf("failed creating TXT record: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("error closing response body: %v", closeErr)
		}
	}()

	return checkResponse(resp)
}

// FindTXTRecord finds a specific TXT record by name and content in a zone.
// Returns the record ref if found, empty string if not found.
func (client *Client) FindTXTRecord(zoneRef, zoneName, recordFQDN, recordContent string) (_ string, retErr error) {
	relName := strings.TrimSuffix(recordFQDN, zoneName)
	relName = strings.TrimSuffix(relName, ".")

	path := "/dnsZones/" + zoneRef + "/dnsRecords?filter=" + relName

	resp, err := client.doRequest("GET", path, nil)
	if err != nil {
		return "", fmt.Errorf("failed listing zone records: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("error closing response body: %v", closeErr)
		}
	}()

	if err := checkResponse(resp); err != nil {
		return "", err
	}

	var result micetroRecordsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed decoding records response: %v", err)
	}

	for _, r := range result.DNSRecords {
		if r.Type == "TXT" && r.Data == recordContent {
			recFQDN := r.Name + "." + zoneName
			if recFQDN == recordFQDN || r.Name == recordFQDN {
				return r.Ref, nil
			}
		}
	}

	return "", nil
}

// DeleteRecord deletes a DNS record by its object reference.
func (client *Client) DeleteRecord(recordRef string) (retErr error) {
	path := "/dnsRecords/" + recordRef

	resp, err := client.doRequest("DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed deleting record: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("error closing response body: %v", closeErr)
		}
	}()

	return checkResponse(resp)
}
