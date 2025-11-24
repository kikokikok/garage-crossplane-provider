// Package v1alpha1 contains API Schema definitions for the garage v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=garage.crossplane.io
package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "garage.crossplane.io"
	Version = "v1alpha1"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// Bucket type metadata.
var (
	BucketKind             = reflect.TypeOf(Bucket{}).Name()
	BucketGroupKind        = schema.GroupKind{Group: Group, Kind: BucketKind}.String()
	BucketKindAPIVersion   = BucketKind + "." + GroupVersion.String()
	BucketGroupVersionKind = GroupVersion.WithKind(BucketKind)
)

// Key type metadata.
var (
	KeyKind             = reflect.TypeOf(Key{}).Name()
	KeyGroupKind        = schema.GroupKind{Group: Group, Kind: KeyKind}.String()
	KeyKindAPIVersion   = KeyKind + "." + GroupVersion.String()
	KeyGroupVersionKind = GroupVersion.WithKind(KeyKind)
)

// KeyAccess type metadata.
var (
	KeyAccessKind             = reflect.TypeOf(KeyAccess{}).Name()
	KeyAccessGroupKind        = schema.GroupKind{Group: Group, Kind: KeyAccessKind}.String()
	KeyAccessKindAPIVersion   = KeyAccessKind + "." + GroupVersion.String()
	KeyAccessGroupVersionKind = GroupVersion.WithKind(KeyAccessKind)
)

func init() {
	SchemeBuilder.Register(&Bucket{}, &BucketList{})
	SchemeBuilder.Register(&Key{}, &KeyList{})
	SchemeBuilder.Register(&KeyAccess{}, &KeyAccessList{})
}
