package webhook

import (
	"net/http"
	"k8s.io/client-go/kubernetes"
)

// MicetroDNSProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
//
// It must implement the [webhook.Solver] interface:
// https://pkg.go.dev/github.com/cert-manager/cert-manager/pkg/acme/webhook#Solver
type MicetroDNSProviderSolver struct {
	// If a Kubernetes 'clientset' is needed, you must:
	// 1. uncomment the additional `client` field in this structure below
	// 2. uncomment the "k8s.io/client-go/kubernetes" import at the top of the file
	// 3. uncomment the relevant code in the Initialize method below
	// 4. ensure your webhook's service account has the required RBAC role
	//    assigned to it for interacting with the Kubernetes APIs you need.
	client *kubernetes.Clientset
}

// MicetroDNSProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
// This typically includes references to Secret resources containing DNS
// provider credentials, in cases where a 'multi-tenant' DNS solver is being
// created.
// If you do *not* require per-issuer or per-certificate configuration to be
// provided to your webhook, you can skip decoding altogether in favour of
// using CLI flags or similar to provide configuration.
// You should not include sensitive information here. If credentials need to
// be used by your provider here, you should reference a Kubernetes Secret
// resource and fetch these credentials using a Kubernetes clientset.
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

	// TTL is the time-to-live value in seconds of the inserted DNS records.
	// Default is 60 seconds if not specified.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3600
	// +kubebuilder:default=60
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Format=int32
	// +optional
	TTL int `json:"ttl"`

	// Timeout is the timeout value in seconds for requests to the Micetro API.
	// Default is 30 seconds if not specified.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=300
	// +kubebuilder:default=30
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Format=int32
	// +optional
	Timeout int `json:"timeout"`
}

// micetroAuthSecretRef contains the reference information for the Kubernetes
// secret which contains the Micetro API credentials.
type micetroAuthSecretRef struct {
	// Namespace is the namespace of the secret containing the Micetro API credentials.
	Namespace string `json:"namespace"`

	// Name is the name of the secret containing the Micetro API credentials.
	Name string `json:"name"`

	// Type is the type of the secret containing the Micetro API credentials.
	// Valid options are "basic" and "token".
	// If "basic" is specified, the secret must contain the keys "username" and "password".
	// If "token" is specified, the secret must contain the key "token".
	// +kubebuilder:validation:Enum=basic;token
	Type string `json:"type"`
}

// micetroCABundleConfigMapRef contains the reference information for the Kubernetes
// config map which contains the Micetro CA bundle.
type micetroCABundleConfigMapRef struct {
	// Namespace is the namespace of the config map containing the Micetro CA bundle.
	Namespace string `json:"namespace"`

	// Name is the name of the config map containing the Micetro CA bundle.
	Name string `json:"name"`
	
	// Key is the key of the config map containing the Micetro CA bundle.
	// If not specified, the default key "ca-bundle.crt" will be used.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=string
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9._-]+$`
	// +kubebuilder:default="ca-bundle.crt"
	// +optional
	Key string `json:"key"`
}

// Client configuration structure
type Client struct {
	BaseURL string
	Headers map[string]string

	httpClient *http.Client
	apiKey     *string
}