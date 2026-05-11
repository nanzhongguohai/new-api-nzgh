package system_setting

import "github.com/QuantumNous/new-api/setting/config"

// OIDCProviderSettings configures new-api as an OIDC Authorization Server (Provider).
type OIDCProviderSettings struct {
	Enabled bool `json:"enabled"`
}

var defaultOIDCProviderSettings = OIDCProviderSettings{}

func init() {
	config.GlobalConfig.Register("oidc_provider", &defaultOIDCProviderSettings)
}

func GetOIDCProviderSettings() *OIDCProviderSettings {
	return &defaultOIDCProviderSettings
}
