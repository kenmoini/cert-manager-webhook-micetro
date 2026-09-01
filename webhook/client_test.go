package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(handler http.Handler) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := &Client{
		BaseURL:    server.URL,
		httpClient: server.Client(),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}
	return client, server
}

func TestClient_Authenticate_Success(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, DefaultMicetroAPIAuthPath)

		var req micetroAuthRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "admin", req.LoginName)
		assert.Equal(t, "secret", req.Password)

		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(micetroAuthResponse{Session: "test-session-token"})
		require.NoError(t, err)
	}))
	defer server.Close()

	err := client.authenticate("admin", "secret")
	require.NoError(t, err)
	assert.Equal(t, "test-session-token", client.token)
}

func TestClient_Authenticate_InvalidCredentials(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		err := json.NewEncoder(w).Encode(micetroAPIError{Code: 401, Message: "Invalid credentials"})
		require.NoError(t, err)
	}))
	defer server.Close()

	err := client.authenticate("bad", "creds")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestClient_FindZone_ExactMatch(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/dnsZones")
		assert.Contains(t, r.URL.RawQuery, "filter=example.com")

		err := json.NewEncoder(w).Encode(micetroZonesResponse{
			DNSZones: []micetroZone{
				{Ref: "DNSZones/1", Name: "example.com.", Type: "Primary"},
				{Ref: "DNSZones/2", Name: "other-example.com.", Type: "Primary"},
			},
			TotalResults: 2,
		})
		require.NoError(t, err)
	}))
	defer server.Close()
	client.token = "test-token"

	zone, err := client.FindZone("example.com.", "")
	require.NoError(t, err)
	assert.Equal(t, "DNSZones/1", zone.Ref)
	assert.Equal(t, "example.com.", zone.Name)
}

func TestClient_FindZone_NotFound(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(micetroZonesResponse{
			DNSZones:     []micetroZone{},
			TotalResults: 0,
		})
		require.NoError(t, err)
	}))
	defer server.Close()
	client.token = "test-token"

	_, err := client.FindZone("missing.com.", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestClient_FindZone_WithView(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "dnsViewRef=DNSViews/1")

		err := json.NewEncoder(w).Encode(micetroZonesResponse{
			DNSZones:     []micetroZone{{Ref: "DNSZones/5", Name: "example.com.", Type: "Primary"}},
			TotalResults: 1,
		})
		require.NoError(t, err)
	}))
	defer server.Close()
	client.token = "test-token"

	zone, err := client.FindZone("example.com.", "DNSViews/1")
	require.NoError(t, err)
	assert.Equal(t, "DNSZones/5", zone.Ref)
}

func TestClient_FindViewRef_Found(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/dnsViews")

		err := json.NewEncoder(w).Encode(micetroViewsResponse{
			DNSViews:     []micetroView{{Ref: "DNSViews/1", Name: "internal"}},
			TotalResults: 1,
		})
		require.NoError(t, err)
	}))
	defer server.Close()
	client.token = "test-token"

	ref, err := client.FindViewRef("internal")
	require.NoError(t, err)
	assert.Equal(t, "DNSViews/1", ref)
}

func TestClient_FindViewRef_Empty(t *testing.T) {
	client := &Client{}
	ref, err := client.FindViewRef("")
	require.NoError(t, err)
	assert.Equal(t, "", ref)
}

func TestClient_CreateTXTRecord_Success(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/dnsZones/DNSZones/1/dnsRecords")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req micetroCreateRecordRequest
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)
		assert.Equal(t, "_acme-challenge.sub.example.com.", req.DNSRecord.Name)
		assert.Equal(t, "TXT", req.DNSRecord.Type)
		assert.Equal(t, "challenge-token", req.DNSRecord.Data)
		assert.Equal(t, "60", req.DNSRecord.TTL)

		w.WriteHeader(http.StatusCreated)
		_, err = w.Write([]byte(`{"ref":"DNSRecords/42"}`))
		require.NoError(t, err)
	}))
	defer server.Close()
	client.token = "test-token"

	err := client.CreateTXTRecord("DNSZones/1", "_acme-challenge.sub.example.com.", "challenge-token", 60)
	require.NoError(t, err)
}

func TestClient_CreateTXTRecord_Error(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		err := json.NewEncoder(w).Encode(micetroAPIError{Code: 400, Message: "Invalid record data"})
		require.NoError(t, err)
	}))
	defer server.Close()
	client.token = "test-token"

	err := client.CreateTXTRecord("DNSZones/1", "test.example.com.", "data", 60)
	assert.Error(t, err)
}

func TestClient_FindTXTRecord_Found(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/dnsZones/DNSZones/1/dnsRecords")

		err := json.NewEncoder(w).Encode(micetroRecordsResponse{
			DNSRecords: []micetroRecord{
				{Ref: "DNSRecords/10", Name: "_acme-challenge.sub", Type: "TXT", Data: "wrong-token"},
				{Ref: "DNSRecords/11", Name: "_acme-challenge.sub", Type: "TXT", Data: "correct-token"},
				{Ref: "DNSRecords/12", Name: "_acme-challenge.sub", Type: "A", Data: "1.2.3.4"},
			},
			TotalResults: 3,
		})
		require.NoError(t, err)
	}))
	defer server.Close()
	client.token = "test-token"

	ref, err := client.FindTXTRecord("DNSZones/1", "example.com.", "_acme-challenge.sub.example.com.", "correct-token")
	require.NoError(t, err)
	assert.Equal(t, "DNSRecords/11", ref)
}

func TestClient_FindTXTRecord_NotFound(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(micetroRecordsResponse{
			DNSRecords:   []micetroRecord{},
			TotalResults: 0,
		})
		require.NoError(t, err)
	}))
	defer server.Close()
	client.token = "test-token"

	ref, err := client.FindTXTRecord("DNSZones/1", "example.com.", "_acme-challenge.example.com.", "token")
	require.NoError(t, err)
	assert.Equal(t, "", ref)
}

func TestClient_DeleteRecord_Success(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "/dnsRecords/DNSRecords/42")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client.token = "test-token"

	err := client.DeleteRecord("DNSRecords/42")
	require.NoError(t, err)
}

func TestClient_DeleteRecord_NotFound(t *testing.T) {
	client, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		err := json.NewEncoder(w).Encode(micetroAPIError{Code: 404, Message: "Object not found"})
		require.NoError(t, err)
	}))
	defer server.Close()
	client.token = "test-token"

	err := client.DeleteRecord("DNSRecords/999")
	assert.Error(t, err)
}
