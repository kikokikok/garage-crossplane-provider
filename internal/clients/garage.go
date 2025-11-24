// Package clients contains the Terraform setup builder for the Garage provider
package clients

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/upjet/pkg/terraform"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kikokikok/provider-garage/apis/v1beta1"
)

const (
	// Garage provider configuration keys
	garageEndpoint   = "garage_endpoint"
	garageAdminToken = "garage_admin_token"

	// Terraform provider details
	terraformProviderSource  = "deuxfleurs-org/garage"
	terraformProviderVersion = "0.9.0"
	
	// Error messages
	errGetProviderConfig = "cannot get referenced ProviderConfig"
	errTrackUsage        = "cannot track ProviderConfig usage"
	errExtractSecret     = "cannot extract credentials from secret"
	errUnmarshalSecret   = "cannot unmarshal credentials secret"
)

// TerraformSetupBuilder builds Terraform setup for Garage provider
func TerraformSetupBuilder(version, providerSource, providerVersion string) terraform.SetupFn {
	return func(ctx context.Context, c client.Client, mg resource.Managed) (terraform.Setup, error) {
		ps := terraform.Setup{
			Version: version,
			Requirement: terraform.ProviderRequirement{
				Source:  providerSource,
				Version: providerVersion,
			},
		}

		// Get ProviderConfig reference
		configRef := mg.GetProviderConfigReference()
		if configRef == nil {
			return ps, errors.New("no providerConfigRef provided")
		}

		// Read the ProviderConfig
		pc := &v1beta1.ProviderConfig{}
		if err := c.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
			return ps, errors.Wrap(err, errGetProviderConfig)
		}

		// Note: Usage tracking is handled by Crossplane's core controllers.
		// We simply reference the ProviderConfig here for credential extraction.

		// Extract credentials from the referenced secret
		cd := pc.Spec.Credentials
		data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c, cd.CommonCredentialSelectors)
		if err != nil {
			return ps, errors.Wrap(err, errExtractSecret)
		}

		// Parse credentials
		creds := map[string]string{}
		if err := json.Unmarshal(data, &creds); err != nil {
			return ps, errors.Wrap(err, errUnmarshalSecret)
		}

		// Set provider configuration
		ps.Configuration = map[string]any{}
		
		// Set Garage endpoint
		if v, ok := creds[garageEndpoint]; ok && v != "" {
			ps.Configuration[garageEndpoint] = v
		} else {
			return ps, fmt.Errorf("missing required credential: %s", garageEndpoint)
		}
		
		// Set Garage admin token
		if v, ok := creds[garageAdminToken]; ok && v != "" {
			ps.Configuration[garageAdminToken] = v
		} else {
			return ps, fmt.Errorf("missing required credential: %s", garageAdminToken)
		}

		return ps, nil
	}
}
