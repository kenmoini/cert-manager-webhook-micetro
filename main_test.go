package main

import (
	"testing"

	acmetest "github.com/cert-manager/cert-manager/test/acme"

	micetrowebhook "github.com/kenmoini/cert-manager-webhook-micetro/webhook"
)

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.
	//

	// Uncomment the below fixture when implementing your custom DNS provider
	//fixture := acmetest.NewFixture(&micetrowebhook.MicetroDNSProviderSolver{},
	//	acmetest.SetResolvedZone(zone),
	//	acmetest.SetAllowAmbientCredentials(false),
	//	acmetest.SetManifestPath("testdata/micetro-solver"),
	//	acmetest.SetBinariesPath("_test/kubebuilder/bin"),
	//)
	solver := micetrowebhook.New("59351")
	fixture := acmetest.NewFixture(solver,
		acmetest.SetResolvedZone("example.com."),
		acmetest.SetManifestPath("testdata/micetro-solver"),
		acmetest.SetDNSServer("127.0.0.1:59351"),
		acmetest.SetUseAuthoritative(false),
	)
	//need to uncomment and  RunConformance delete runBasic and runExtended once https://github.com/cert-manager/cert-manager/pull/4835 is merged
	//fixture.RunConformance(t)
	fixture.RunBasic(t)
	fixture.RunExtended(t)

}
