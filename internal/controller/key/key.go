// Package key contains the controller for Key resources
package key

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
	errNotKey       = "managed resource is not a Key custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errCreateKey    = "cannot create key"
	errDeleteKey    = "cannot delete key"
	errGetKey       = "cannot get key"
)

// Setup adds a controller that reconciles Key managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.KeyGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.KeyGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube: mgr.GetClient(),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.Key{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Key)
	if !ok {
		return nil, errors.New(errNotKey)
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
	cr, ok := mg.(*v1alpha1.Key)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotKey)
	}

	// If we don't have an access key ID, the resource doesn't exist
	if cr.Status.AtProvider.AccessKeyID == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Try to get the key from Garage
	key, err := e.client.GetKey(ctx, cr.Status.AtProvider.AccessKeyID)
	if err != nil {
		// Key doesn't exist
		cr.Status.AtProvider.AccessKeyID = ""
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	cr.Status.AtProvider.AccessKeyID = key.AccessKeyID
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Key)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotKey)
	}

	cr.SetConditions(xpv1.Creating())

	req := &garage.CreateKeyRequest{
		Name: cr.Spec.ForProvider.Name,
	}

	key, err := e.client.CreateKey(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateKey)
	}

	cr.Status.AtProvider.AccessKeyID = key.AccessKeyID

	// Return connection details (access key ID and secret)
	connDetails := managed.ConnectionDetails{
		"accessKeyId":     []byte(key.AccessKeyID),
		"secretAccessKey": []byte(key.SecretAccessKey),
	}

	return managed.ExternalCreation{
		ConnectionDetails: connDetails,
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// Keys are immutable for now
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Key)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotKey)
	}

	cr.SetConditions(xpv1.Deleting())

	if cr.Status.AtProvider.AccessKeyID == "" {
		return managed.ExternalDelete{}, nil
	}

	return managed.ExternalDelete{}, errors.Wrap(e.client.DeleteKey(ctx, cr.Status.AtProvider.AccessKeyID), errDeleteKey)
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}
