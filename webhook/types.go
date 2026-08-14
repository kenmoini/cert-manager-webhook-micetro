package webhook


// micetroDNSProviderConfig is a structure that is used to decode into when
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
type micetroDNSProviderConfig struct {
	// Version is the version of the Micetro API.  Must be specified.
	// Valid options are "26.1", "25.2", and "25.1".
	// +kubebuilder:validation:Enum=26.1;25.2;25.1
	Version string `json:"version"`

	// Host is the Base URL (e.g. https://dns.example.ca) of the Micetro API.
	Host string `json:"host"`

	// Headers are additional headers added to requests to the
	// Micetro API server.
	Headers map[string]string `json:"headers"`

	// CABundle is a PEM encoded CA bundle which will be used in
	// certificate validation when connecting to the Micetro server.
	//
	// When left blank, the default system store will be used.
	//
	// +optional
	CABundle []byte `json:"caBundle"`

	// AllowedZones is the list of zones that may be edited. If the list is
	// empty, all zones are permitted.
	AllowedZones []string `json:"allowedZones"`

	// AuthSecretRef contains the reference information for the Kubernetes
	// secret which contains the Micetro API credentials.
	AuthSecretRef *micetroAuthSecretRef `json:"authSecretRef"`

	// TTL is the time-to-live value of the inserted DNS records.
	//
	// +optional
	TTL int `json:"ttl"`

	// Timeout is the timeout value for requests to the Micetro API.
	// The value is specified in seconds.
	//
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