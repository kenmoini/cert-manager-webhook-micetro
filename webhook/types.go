package webhook

import (
	"fmt"
	"net/http"

	"k8s.io/client-go/kubernetes"
)

// MicetroDNSProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
//
// It must implement the [webhook.Solver] interface:
// https://pkg.go.dev/github.com/cert-manager/cert-manager/pkg/acme/webhook#Solver
type MicetroDNSProviderSolver struct {
	client *kubernetes.Clientset
}

// MicetroDNSProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
type MicetroDNSProviderConfig struct {
	// Version is the version of the Micetro API.  Must be specified.
	// Valid options are "26.1", "25.2", and "25.1".
	// +kubebuilder:validation:Enum=26.1;25.2;25.1
	Version string `json:"version"`

	// Host is the Base URL (e.g. https://dns.example.ca) of the Micetro API.
	Host string `json:"host"`

	// AuthSecretRef contains the reference information for the Kubernetes
	// secret which contains the Micetro API credentials.
	AuthSecretRef *micetroAuthSecretRef `json:"authSecretRef"`

	// Headers are additional headers added to requests to the
	// Micetro API server.
	// +optional
	Headers map[string]string `json:"headers"`

	// CABundleRef contains the reference information for the Kubernetes config map which contains the Micetro CA bundle.
	// If not specified, the system CA bundle will be used.
	// +optional
	CABundleRef *micetroCABundleConfigMapRef `json:"caBundleRef"`

	// AllowedZones is the list of zones that may be edited. If the list is
	// empty, all zones are permitted.
	// +optional
	AllowedZones []string `json:"allowedZones"`

	// DNSViewRef is an optional DNS view name to scope zone lookups.
	// If not specified, the default view is used.
	// +optional
	DNSViewRef string `json:"dnsViewRef,omitempty"`

	// TTL is the time-to-live value in seconds of the inserted DNS records.
	// Default is 60 seconds if not specified.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3600
	// +kubebuilder:default=60
	// +optional
	TTL int `json:"ttl"`

	// Timeout is the timeout value in seconds for requests to the Micetro API.
	// Default is 30 seconds if not specified.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=300
	// +kubebuilder:default=30
	// +optional
	Timeout int `json:"timeout"`
}

// micetroAuthSecretRef contains the reference information for the Kubernetes
// secret which contains the Micetro API credentials.
type micetroAuthSecretRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	// Type is "basic" (username/password) or "token" (pre-existing session token).
	// +kubebuilder:validation:Enum=basic;token
	Type string `json:"type"`
}

// micetroCABundleConfigMapRef contains the reference information for the Kubernetes
// config map which contains the Micetro CA bundle.
type micetroCABundleConfigMapRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	// +kubebuilder:default="ca-bundle.crt"
	// +optional
	Key string `json:"key"`
}

// Client is the Micetro REST API client.
type Client struct {
	BaseURL    string
	Headers    map[string]string
	httpClient *http.Client
	token      string
}

// --- Micetro API request/response types (internal) ---

type micetroAuthRequest struct {
	LoginName string `json:"loginName"`
	Password  string `json:"password"`
}

type micetroAuthResponse struct {
	Session string `json:"session"`
}

type micetroAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *micetroAPIError) Error() string {
	return fmt.Sprintf("micetro API error (code %d): %s", e.Code, e.Message)
}

type micetroZone struct {
	Ref        string `json:"ref"`
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	DNSViewRef string `json:"dnsViewRef,omitempty"`
}

type micetroZonesResponse struct {
	DNSZones     []micetroZone `json:"dnsZones"`
	TotalResults int           `json:"totalResults"`
}

type micetroRecord struct {
	Ref        string `json:"ref,omitempty"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Data       string `json:"data"`
	TTL        string `json:"ttl,omitempty"`
	Comment    string `json:"comment,omitempty"`
	Enabled    bool   `json:"enabled,omitempty"`
	DNSZoneRef string `json:"dnsZoneRef,omitempty"`
}

type micetroRecordsResponse struct {
	DNSRecords   []micetroRecord `json:"dnsRecords"`
	TotalResults int             `json:"totalResults"`
}

type micetroCreateRecordRequest struct {
	DNSRecord   micetroRecord `json:"dnsRecord"`
	SaveComment string        `json:"saveComment,omitempty"`
}

type micetroView struct {
	Ref  string `json:"ref"`
	Name string `json:"name"`
}

type micetroViewsResponse struct {
	DNSViews     []micetroView `json:"dnsViews"`
	TotalResults int           `json:"totalResults"`
}
