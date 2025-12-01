// Package bucket contains the controller for Bucket resources
package bucket

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
	errNotBucket    = "managed resource is not a Bucket custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errCreateBucket = "cannot create bucket"
	errDeleteBucket = "cannot delete bucket"
)

// Setup adds a controller that reconciles Bucket managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.BucketGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.BucketGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube: mgr.GetClient(),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.Bucket{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Bucket)
	if !ok {
		return nil, errors.New(errNotBucket)
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

	return &external{client: garageClient}, nil
}

type external struct {
	client *garage.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Bucket)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBucket)
	}

	var bucket *garage.Bucket
	var err error

	// Try to find by ID first
	if cr.Status.AtProvider.ID != "" {
		bucket, err = e.client.GetBucket(ctx, cr.Status.AtProvider.ID)
		if err != nil {
			// Bucket doesn't exist by ID, clear the ID and try by alias
			cr.Status.AtProvider.ID = ""
		}
	}

	// If no ID or ID lookup failed, try by globalAlias
	if bucket == nil && cr.Spec.ForProvider.GlobalAlias != nil && *cr.Spec.ForProvider.GlobalAlias != "" {
		bucket, err = e.client.GetBucketByAlias(ctx, *cr.Spec.ForProvider.GlobalAlias)
		if err != nil {
			// Bucket doesn't exist
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
	}

	if bucket == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	cr.Status.AtProvider.ID = bucket.ID
	cr.Status.AtProvider.GlobalAliases = bucket.GlobalAliases
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Bucket)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBucket)
	}

	cr.SetConditions(xpv1.Creating())

	req := &garage.CreateBucketRequest{}
	if cr.Spec.ForProvider.GlobalAlias != nil {
		req.GlobalAlias = cr.Spec.ForProvider.GlobalAlias
	}

	bucket, err := e.client.CreateBucket(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateBucket)
	}

	cr.Status.AtProvider.ID = bucket.ID
	cr.Status.AtProvider.GlobalAliases = bucket.GlobalAliases

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Bucket)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotBucket)
	}

	cr.SetConditions(xpv1.Deleting())

	if cr.Status.AtProvider.ID == "" {
		return managed.ExternalDelete{}, nil
	}

	return managed.ExternalDelete{}, errors.Wrap(e.client.DeleteBucket(ctx, cr.Status.AtProvider.ID), errDeleteBucket)
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}
