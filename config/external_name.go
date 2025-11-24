// Package config contains the external name configurations for Garage resources
package config

import "github.com/crossplane/upjet/pkg/config"

// ExternalNameConfigs defines external names for Garage resources.
// External names are used to uniquely identify resources in the external system (Garage).
var ExternalNameConfigs = map[string]config.ExternalName{
	// Garage bucket resource - uses provider-assigned identifier
	"garage_bucket": config.IdentifierFromProvider,
	
	// Garage key resource - uses provider-assigned identifier
	"garage_key": config.IdentifierFromProvider,
	
	// Garage key access resource - uses provider-assigned identifier
	"garage_key_access": config.IdentifierFromProvider,
}
