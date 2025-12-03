// Package keyaccess contains the controller for KeyAccess resources
package keyaccess

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	v1 "github.com/kikokikok/provider-garage/apis/v1"
	"github.com/kikokikok/provider-garage/apis/v1alpha1"
	"github.com/kikokikok/provider-garage/pkg/garage"
)

const (
	errNotKeyAccess    = "managed resource is not a KeyAccess custom resource"
	errTrackPCUsage    = "cannot track ProviderConfig usage"
	errGetPC           = "cannot get ProviderConfig"
	errGetCreds        = "cannot get credentials"
	errGrantAccess     = "cannot grant key access"
	errRevokeAccess    = "cannot revoke key access"
	errResolveBucket   = "cannot resolve bucket reference"
	errResolveKey      = "cannot resolve key reference"
)

// Setup adds a controller that reconciles KeyAccess managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.KeyAccessGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.KeyAccessGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube: mgr.GetClient(),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.KeyAccess{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.KeyAccess)
	if !ok {
		return nil, errors.New(errNotKeyAccess)
	}

	pc := &v1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	creds := struct {
		Endpoint   string `json:"endpoint"`
		AdminToken string `json:"adminToken"`
	}{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &creds); err != nil {
			return nil, errors.Wrap(err, errGetCreds)
		}
	}

	endpoint := creds.Endpoint
	if pc.Spec.Endpoint != nil && *pc.Spec.Endpoint != "" {
		endpoint = *pc.Spec.Endpoint
	}

	garageClient := garage.NewClient(endpoint, creds.AdminToken)

	return &external{client: garageClient, kube: c.kube}, nil
}

type external struct {
	client *garage.Client
	kube   client.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.KeyAccess)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotKeyAccess)
	}

	// Resolve bucket ID
	bucketID, err := e.resolveBucketID(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errResolveBucket)
	}

	// Resolve access key ID
	accessKeyID, err := e.resolveAccessKeyID(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errResolveKey)
	}

	if bucketID == "" || accessKeyID == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Check if the key has access to the bucket
	bucket, err := e.client.GetBucket(ctx, bucketID)
	if err != nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Look for the key in bucket's key list
	var hasAccess bool
	for _, k := range bucket.Keys {
		if k.AccessKeyID == accessKeyID {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	cr.Status.AtProvider.BucketID = bucketID
	cr.Status.AtProvider.AccessKeyID = accessKeyID
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.KeyAccess)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotKeyAccess)
	}

	cr.SetConditions(xpv1.Creating())

	// Resolve bucket ID
	bucketID, err := e.resolveBucketID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errResolveBucket)
	}

	// Resolve access key ID
	accessKeyID, err := e.resolveAccessKeyID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errResolveKey)
	}

	req := &garage.GrantKeyAccessRequest{
		BucketID:    bucketID,
		AccessKeyID: accessKeyID,
	}
	req.Permissions.Read = cr.Spec.ForProvider.Permissions.Read
	req.Permissions.Write = cr.Spec.ForProvider.Permissions.Write
	req.Permissions.Owner = cr.Spec.ForProvider.Permissions.Owner

	_, err = e.client.GrantKeyAccess(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGrantAccess)
	}

	cr.Status.AtProvider.BucketID = bucketID
	cr.Status.AtProvider.AccessKeyID = accessKeyID

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// KeyAccess permissions can be updated by re-granting
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.KeyAccess)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotKeyAccess)
	}

	cr.SetConditions(xpv1.Deleting())

	if cr.Status.AtProvider.BucketID == "" || cr.Status.AtProvider.AccessKeyID == "" {
		return managed.ExternalDelete{}, nil
	}

	req := &garage.RevokeKeyAccessRequest{
		BucketID:    cr.Status.AtProvider.BucketID,
		AccessKeyID: cr.Status.AtProvider.AccessKeyID,
	}

	_, err := e.client.RevokeKeyAccess(ctx, req)
	return managed.ExternalDelete{}, errors.Wrap(err, errRevokeAccess)
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}

// resolveBucketID resolves the bucket ID from direct value or reference
func (e *external) resolveBucketID(ctx context.Context, cr *v1alpha1.KeyAccess) (string, error) {
	if cr.Spec.ForProvider.BucketID != nil && *cr.Spec.ForProvider.BucketID != "" {
		return *cr.Spec.ForProvider.BucketID, nil
	}

	if cr.Spec.ForProvider.BucketIDRef != nil {
		bucket := &v1alpha1.Bucket{}
		err := e.kube.Get(ctx, types.NamespacedName{
			Name:      cr.Spec.ForProvider.BucketIDRef.Name,
			Namespace: cr.Namespace,
		}, bucket)
		if err != nil {
			return "", err
		}
		// Check if the Bucket has been reconciled and has an ID
		if bucket.Status.AtProvider.ID == "" {
			return "", errors.New("referenced Bucket has not been reconciled yet (ID is empty)")
		}
		return bucket.Status.AtProvider.ID, nil
	}

	return "", nil
}

// resolveAccessKeyID resolves the access key ID from direct value or reference
func (e *external) resolveAccessKeyID(ctx context.Context, cr *v1alpha1.KeyAccess) (string, error) {
	if cr.Spec.ForProvider.AccessKeyID != nil && *cr.Spec.ForProvider.AccessKeyID != "" {
		return *cr.Spec.ForProvider.AccessKeyID, nil
	}

	if cr.Spec.ForProvider.AccessKeyIDRef != nil {
		key := &v1alpha1.Key{}
		err := e.kube.Get(ctx, types.NamespacedName{
			Name:      cr.Spec.ForProvider.AccessKeyIDRef.Name,
			Namespace: cr.Namespace,
		}, key)
		if err != nil {
			return "", err
		}
		// Check if the Key has been reconciled and has an AccessKeyID
		if key.Status.AtProvider.AccessKeyID == "" {
			return "", errors.New("referenced Key has not been reconciled yet (AccessKeyID is empty)")
		}
		return key.Status.AtProvider.AccessKeyID, nil
	}

	return "", nil
}
