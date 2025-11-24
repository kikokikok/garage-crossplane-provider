// Package key contains the configuration for Garage key resources
package key

import "github.com/crossplane/upjet/pkg/config"

// Configure customizes the Garage key resources
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("garage_key", func(r *config.Resource) {
		// Set the short group for the API group
		r.ShortGroup = "key"
		
		// Set the Kind for the CRD
		r.Kind = "Key"
		
		// Set a description for the resource
		r.Description = "Key is a managed resource that represents a Garage access key"
		
		// Mark sensitive fields that should be stored in connection secrets
		r.Sensitive.AdditionalConnectionDetailsFn = func(attr map[string]any) (map[string][]byte, error) {
			conn := map[string][]byte{}
			if v, ok := attr["access_key_id"].(string); ok {
				conn["access_key_id"] = []byte(v)
			}
			if v, ok := attr["secret_access_key"].(string); ok {
				conn["secret_access_key"] = []byte(v)
			}
			return conn, nil
		}
	})
	
	p.AddResourceConfigurator("garage_key_access", func(r *config.Resource) {
		// Set the short group for the API group
		r.ShortGroup = "key"
		
		// Set the Kind for the CRD
		r.Kind = "KeyAccess"
		
		// Set a description for the resource
		r.Description = "KeyAccess is a managed resource that represents Garage key access permissions to a bucket"
		
		// Set references to establish relationships with other resources
		r.References["key_id"] = config.Reference{
			Type: "Key",
		}
		r.References["bucket_id"] = config.Reference{
			Type:              "github.com/kikokikok/provider-garage/apis/bucket/v1alpha1.Bucket",
			RefFieldName:      "BucketIDRef",
			SelectorFieldName: "BucketIDSelector",
		}
	})
}
