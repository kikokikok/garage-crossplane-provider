// Package v1alpha1 contains API Schema definitions for the garage v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=garage.crossplane.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// BucketSpec defines the desired state of Bucket
type BucketSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       BucketParameters `json:"forProvider"`
}

// BucketParameters are the configurable fields of a Bucket.
type BucketParameters struct {
	// GlobalAlias is the global alias for the bucket (S3 bucket name)
	// +optional
	GlobalAlias *string `json:"globalAlias,omitempty"`

	// LocalAlias is a local alias for the bucket
	// +optional
	LocalAlias *LocalAlias `json:"localAlias,omitempty"`

	// Quotas for the bucket
	// +optional
	Quotas *BucketQuotas `json:"quotas,omitempty"`
}

// LocalAlias represents a local alias for a bucket
type LocalAlias struct {
	// AccessKeyID is the access key ID to associate the alias with
	AccessKeyID string `json:"accessKeyId"`
	// Alias is the local alias name
	Alias string `json:"alias"`
}

// BucketQuotas represents quotas for a bucket
type BucketQuotas struct {
	// MaxSize is the maximum size in bytes
	// +optional
	MaxSize *int64 `json:"maxSize,omitempty"`
	// MaxObjects is the maximum number of objects
	// +optional
	MaxObjects *int64 `json:"maxObjects,omitempty"`
}

// BucketStatus represents the observed state of a Bucket.
type BucketStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          BucketObservation `json:"atProvider,omitempty"`
}

// BucketObservation are the observable fields of a Bucket.
type BucketObservation struct {
	// ID is the unique identifier of the bucket
	ID string `json:"id,omitempty"`
	// GlobalAliases are the global aliases of the bucket
	GlobalAliases []string `json:"globalAliases,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,garage}

// Bucket is a managed resource that represents a Garage bucket.
type Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketSpec   `json:"spec"`
	Status BucketStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BucketList contains a list of Bucket
type BucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bucket `json:"items"`
}

// KeySpec defines the desired state of Key
type KeySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       KeyParameters `json:"forProvider"`
}

// KeyParameters are the configurable fields of a Key.
type KeyParameters struct {
	// Name is the name of the key
	Name string `json:"name"`

	// Permissions for the key
	// +optional
	Permissions *KeyPermissions `json:"permissions,omitempty"`
}

// KeyPermissions represents global permissions for a key
type KeyPermissions struct {
	// CreateBucket allows the key to create buckets
	CreateBucket bool `json:"createBucket"`
}

// KeyStatus represents the observed state of a Key.
type KeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          KeyObservation `json:"atProvider,omitempty"`
}

// KeyObservation are the observable fields of a Key.
type KeyObservation struct {
	// AccessKeyID is the access key ID
	AccessKeyID string `json:"accessKeyId,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ACCESS_KEY_ID",type="string",JSONPath=".status.atProvider.accessKeyId"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,garage}

// Key is a managed resource that represents a Garage access key.
type Key struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeySpec   `json:"spec"`
	Status KeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeyList contains a list of Key
type KeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Key `json:"items"`
}

// KeyAccessSpec defines the desired state of KeyAccess
type KeyAccessSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       KeyAccessParameters `json:"forProvider"`
}

// KeyAccessParameters are the configurable fields of a KeyAccess.
type KeyAccessParameters struct {
	// BucketID is the ID of the bucket
	// +optional
	BucketID *string `json:"bucketId,omitempty"`

	// BucketIDRef is a reference to a Bucket to retrieve its ID
	// +optional
	BucketIDRef *xpv1.Reference `json:"bucketIdRef,omitempty"`

	// BucketIDSelector selects a reference to a Bucket
	// +optional
	BucketIDSelector *xpv1.Selector `json:"bucketIdSelector,omitempty"`

	// AccessKeyID is the access key ID
	// +optional
	AccessKeyID *string `json:"accessKeyId,omitempty"`

	// AccessKeyIDRef is a reference to a Key to retrieve its access key ID
	// +optional
	AccessKeyIDRef *xpv1.Reference `json:"accessKeyIdRef,omitempty"`

	// AccessKeyIDSelector selects a reference to a Key
	// +optional
	AccessKeyIDSelector *xpv1.Selector `json:"accessKeyIdSelector,omitempty"`

	// Permissions for the key on the bucket
	Permissions KeyAccessPermissions `json:"permissions"`
}

// KeyAccessPermissions represents permissions for a key on a bucket
type KeyAccessPermissions struct {
	// Read permission
	Read bool `json:"read"`
	// Write permission
	Write bool `json:"write"`
	// Owner permission
	Owner bool `json:"owner"`
}

// KeyAccessStatus represents the observed state of a KeyAccess.
type KeyAccessStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          KeyAccessObservation `json:"atProvider,omitempty"`
}

// KeyAccessObservation are the observable fields of a KeyAccess.
type KeyAccessObservation struct {
	// BucketID is the ID of the bucket
	BucketID string `json:"bucketId,omitempty"`
	// AccessKeyID is the access key ID
	AccessKeyID string `json:"accessKeyId,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="BUCKET",type="string",JSONPath=".status.atProvider.bucketId"
// +kubebuilder:printcolumn:name="KEY",type="string",JSONPath=".status.atProvider.accessKeyId"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,garage}

// KeyAccess is a managed resource that represents access permissions for a key on a bucket.
type KeyAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeyAccessSpec   `json:"spec"`
	Status KeyAccessStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeyAccessList contains a list of KeyAccess
type KeyAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeyAccess `json:"items"`
}

// GetCondition of this Bucket.
func (mg *Bucket) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetDeletionPolicy of this Bucket.
func (mg *Bucket) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// GetManagementPolicies of this Bucket.
func (mg *Bucket) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this Bucket.
func (mg *Bucket) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

// GetPublishConnectionDetailsTo of this Bucket.
func (mg *Bucket) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return mg.Spec.PublishConnectionDetailsTo
}

// GetWriteConnectionSecretToReference of this Bucket.
func (mg *Bucket) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this Bucket.
func (mg *Bucket) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetDeletionPolicy of this Bucket.
func (mg *Bucket) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

// SetManagementPolicies of this Bucket.
func (mg *Bucket) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this Bucket.
func (mg *Bucket) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

// SetPublishConnectionDetailsTo of this Bucket.
func (mg *Bucket) SetPublishConnectionDetailsTo(r *xpv1.PublishConnectionDetailsTo) {
	mg.Spec.PublishConnectionDetailsTo = r
}

// SetWriteConnectionSecretToReference of this Bucket.
func (mg *Bucket) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}

// Similar methods for Key
func (mg *Key) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

func (mg *Key) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

func (mg *Key) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

func (mg *Key) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

func (mg *Key) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return mg.Spec.PublishConnectionDetailsTo
}

func (mg *Key) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

func (mg *Key) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

func (mg *Key) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

func (mg *Key) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

func (mg *Key) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

func (mg *Key) SetPublishConnectionDetailsTo(r *xpv1.PublishConnectionDetailsTo) {
	mg.Spec.PublishConnectionDetailsTo = r
}

func (mg *Key) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}

// Similar methods for KeyAccess
func (mg *KeyAccess) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

func (mg *KeyAccess) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

func (mg *KeyAccess) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

func (mg *KeyAccess) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

func (mg *KeyAccess) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return mg.Spec.PublishConnectionDetailsTo
}

func (mg *KeyAccess) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

func (mg *KeyAccess) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

func (mg *KeyAccess) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

func (mg *KeyAccess) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

func (mg *KeyAccess) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

func (mg *KeyAccess) SetPublishConnectionDetailsTo(r *xpv1.PublishConnectionDetailsTo) {
	mg.Spec.PublishConnectionDetailsTo = r
}

func (mg *KeyAccess) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}

// GroupVersionKind returns the GroupVersionKind for Bucket
func (mg *Bucket) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   GroupVersion.Group,
		Version: GroupVersion.Version,
		Kind:    "Bucket",
	}
}

// GroupVersionKind returns the GroupVersionKind for Key
func (mg *Key) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   GroupVersion.Group,
		Version: GroupVersion.Version,
		Kind:    "Key",
	}
}

// GroupVersionKind returns the GroupVersionKind for KeyAccess
func (mg *KeyAccess) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   GroupVersion.Group,
		Version: GroupVersion.Version,
		Kind:    "KeyAccess",
	}
}
