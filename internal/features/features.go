// Package features contains feature flag setup for the provider.
package features

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
)

// Setup adds feature controllers to the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	// Feature flags will be set up here if needed
	return nil
}
