package paths

// sharedPaths groups path getters for resources shared between users of a given
// machine
type sharedPaths struct{}

// AdminSettingsJSON returns the path to admin-settings.json to override and lock settings.
func (p sharedPaths) AdminSettingsJSON() string {
	return p.Dir(adminSettingsJSONPath)
}

// LicenseFiles returns the paths to license.enc and license.pub.
func (p sharedPaths) LicenseFiles() (licenseEnc, licensePub string) {
	return p.Dir(licenseEncPath), p.Dir(licensePubPath)
}

// InstallSettingsJSON returns the path to install-settings.json to override initial settings.
func (p sharedPaths) InstallSettingsJSON() string {
	return p.Dir(installSettingsJSONPath)
}

// RegistryJSON returns the path to registry.json for restricting image pulls.
func (p sharedPaths) RegistryJSON() string {
	return p.Dir(registryJSONPath)
}

// AccessJSON returns the path to access.json for Registry Access Management.
func (p sharedPaths) AccessJSON() string {
	return p.Dir(accessJSONPath)
}
