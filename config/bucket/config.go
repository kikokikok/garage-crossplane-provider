// Package bucket contains the configuration for Garage bucket resources
package bucket

import "github.com/crossplane/upjet/pkg/config"

// Configure customizes the Garage bucket resource
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("garage_bucket", func(r *config.Resource) {
		// Set the short group for the API group
		r.ShortGroup = "bucket"
		
		// Set the Kind for the CRD
		r.Kind = "Bucket"
		
		// Set a description for the resource
		r.Description = "Bucket is a managed resource that represents a Garage S3 bucket"
	})
}
