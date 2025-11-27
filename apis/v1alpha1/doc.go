// Package v1alpha1 contains API Schema definitions for the garage v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=garage.crossplane.io
//
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../hack/boilerplate.go.txt paths=./...
//go:generate go run github.com/crossplane/crossplane-tools/cmd/angryjet@v0.0.0-20251017183449-dd4517244339 generate-methodsets --header-file=../../hack/boilerplate.go.txt --filename-managed=zz_generated.managed.go .
package v1alpha1
