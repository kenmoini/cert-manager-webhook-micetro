package webhook

import (
	"fmt"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func (c *MicetroDNSProviderSolver) Name() string {
	return "micetro-solver"
}

func (c *MicetroDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	client, cfg, err := c.init(ch.Config, ch.ResourceNamespace)
	if err != nil {
		return fmt.Errorf("failed to initialize provider: %v", err)
	}

	viewRef := ""
	if cfg.DNSViewRef != "" {
		viewRef, err = client.FindViewRef(cfg.DNSViewRef)
		if err != nil {
			return fmt.Errorf("failed to find DNS view %q: %v", cfg.DNSViewRef, err)
		}
	}

	zone, err := client.FindZone(ch.ResolvedZone, viewRef)
	if err != nil {
		return fmt.Errorf("failed to find zone %q in Micetro: %v", ch.ResolvedZone, err)
	}

	recordRef, err := client.FindTXTRecord(zone.Ref, zone.Name, ch.ResolvedFQDN, ch.Key)
	if err != nil {
		return fmt.Errorf("failed to find TXT record %q in zone %q: %v", ch.ResolvedFQDN, zone.Name, err)
	}

	if recordRef == "" {
		return nil
	}

	if err := client.DeleteRecord(recordRef); err != nil {
		return fmt.Errorf("failed to delete TXT record %q (ref: %s): %v", ch.ResolvedFQDN, recordRef, err)
	}

	return nil
}

func (c *MicetroDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	c.client = cl
	return nil
}

func (c *MicetroDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	client, cfg, err := c.init(ch.Config, ch.ResourceNamespace)
	if err != nil {
		return fmt.Errorf("failed to initialize provider: %v", err)
	}

	if !cfg.IsAllowedZone(ch.ResolvedZone) {
		return fmt.Errorf("zone %s is not in the allowed zones list %v", ch.ResolvedZone, cfg.AllowedZones)
	}

	viewRef := ""
	if cfg.DNSViewRef != "" {
		viewRef, err = client.FindViewRef(cfg.DNSViewRef)
		if err != nil {
			return fmt.Errorf("failed to find DNS view %q: %v", cfg.DNSViewRef, err)
		}
	}

	zone, err := client.FindZone(ch.ResolvedZone, viewRef)
	if err != nil {
		return fmt.Errorf("failed to find zone %q in Micetro: %v", ch.ResolvedZone, err)
	}

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = 60
	}

	if err := client.CreateTXTRecord(zone.Ref, ch.ResolvedFQDN, ch.Key, ttl); err != nil {
		return fmt.Errorf("failed to create TXT record %q in zone %q: %v", ch.ResolvedFQDN, zone.Name, err)
	}

	return nil
}
