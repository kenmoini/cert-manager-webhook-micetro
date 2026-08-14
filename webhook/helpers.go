package webhook


// IsAllowedZone checks if the webhook is allowed to edit the given zone, per
// AllowedZones setting. All zones allowed if AllowedZones is empty (the default setting)
func (cfg micetroDNSProviderConfig) IsAllowedZone(zone string) bool {
	// If no allowed zones are specified, all zones are allowed
	if len(cfg.AllowedZones) == 0 {
		return true
	}

	// Check if the zone is in the list of allowed zones, or is a subdomain of an allowed zone
	for _, allowed := range cfg.AllowedZones {
		if zone == allowed || strings.HasSuffix(zone, "."+allowed) {
			return true
		}
	}

	// If the zone is not in the list of allowed zones, return false
	return false
}