package main

import (
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	micetrowebhook "github.com/kenmoini/cert-manager-webhook-micetro/webhook"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		&micetrowebhook.MicetroDNSProviderSolver{},
	)
}

var _ webhook.Solver = (*micetrowebhook.MicetroDNSProviderSolver)(nil)