package key

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	"github.com/kikokikok/provider-garage/apis/v1alpha1"
	"github.com/kikokikok/provider-garage/pkg/garage"
)

// keyClient interface to allow mocking
type keyClient interface {
	CreateKey(ctx context.Context, req *garage.CreateKeyRequest) (*garage.Key, error)
	GetKey(ctx context.Context, accessKeyID string) (*garage.Key, error)
	GetKeyByName(ctx context.Context, name string) (*garage.Key, error)
	DeleteKey(ctx context.Context, accessKeyID string) error
}

type mockKeyClient struct {
	MockCreateKey    func(ctx context.Context, req *garage.CreateKeyRequest) (*garage.Key, error)
	MockGetKey       func(ctx context.Context, accessKeyID string) (*garage.Key, error)
	MockGetKeyByName func(ctx context.Context, name string) (*garage.Key, error)
	MockDeleteKey    func(ctx context.Context, accessKeyID string) error
}

func (m *mockKeyClient) CreateKey(ctx context.Context, req *garage.CreateKeyRequest) (*garage.Key, error) {
	return m.MockCreateKey(ctx, req)
}

func (m *mockKeyClient) GetKey(ctx context.Context, accessKeyID string) (*garage.Key, error) {
	return m.MockGetKey(ctx, accessKeyID)
}

func (m *mockKeyClient) GetKeyByName(ctx context.Context, name string) (*garage.Key, error) {
	return m.MockGetKeyByName(ctx, name)
}

func (m *mockKeyClient) DeleteKey(ctx context.Context, accessKeyID string) error {
	return m.MockDeleteKey(ctx, accessKeyID)
}

// Mock external for testing
type mockExternal struct {
	client keyClient
}

func (e *mockExternal) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Key)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotKey)
	}

	var key *garage.Key

	// The AccessKeyID is stored in the external-name annotation (persisted by Crossplane after Create)
	// and also in Status.AtProvider.AccessKeyID (which may be stale due to Crossplane's behavior)
	externalName := meta.GetExternalName(cr)

	// Try to find by external-name annotation first (this is the most reliable after Create)
	if externalName != "" && externalName != cr.Name {
		// external-name is set to the AccessKeyID
		key, _ = e.client.GetKey(ctx, externalName)
	}

	// Fallback to Status.AtProvider.AccessKeyID
	if key == nil && cr.Status.AtProvider.AccessKeyID != "" {
		key, _ = e.client.GetKey(ctx, cr.Status.AtProvider.AccessKeyID)
		if key == nil {
			// Key doesn't exist by ID - it was deleted externally
			cr.Status.AtProvider.AccessKeyID = ""
		}
	}

	// If we still haven't found the key, try to find it by name.
	// This handles the case where the key was created in Garage but the controller
	// crashed before the external-name annotation could be saved.
	if key == nil && cr.Spec.ForProvider.Name != "" {
		key, _ = e.client.GetKeyByName(ctx, cr.Spec.ForProvider.Name)
		if key != nil {
			// Update external name to the found ID so future lookups are faster
			meta.SetExternalName(cr, key.AccessKeyID)
		}
	}

	// If no key found, this is a fresh resource
	if key == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	cr.Status.AtProvider.AccessKeyID = key.AccessKeyID
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
		// Return connection details on observe so secret gets created for existing keys
		ConnectionDetails: managed.ConnectionDetails{
			"accessKeyId": []byte(key.AccessKeyID),
		},
	}, nil
}

func (e *mockExternal) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
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

	// Store the AccessKeyID in external-name annotation
	meta.SetExternalName(cr, key.AccessKeyID)

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

func (e *mockExternal) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (e *mockExternal) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Key)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotKey)
	}

	cr.SetConditions(xpv1.Deleting())

	// Try to get the AccessKeyID from status first, then from external-name annotation
	accessKeyID := cr.Status.AtProvider.AccessKeyID
	if accessKeyID == "" {
		externalName := meta.GetExternalName(cr)
		if externalName != "" && externalName != cr.Name {
			accessKeyID = externalName
		}
	}

	if accessKeyID == "" {
		return managed.ExternalDelete{}, nil
	}

	return managed.ExternalDelete{}, errors.Wrap(e.client.DeleteKey(ctx, accessKeyID), errDeleteKey)
}

func (e *mockExternal) Disconnect(ctx context.Context) error {
	return nil
}

func TestKeyObserve(t *testing.T) {
	type fields struct {
		client keyClient
	}

	type args struct {
		mg resource.Managed
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"KeyDoesNotExist": {
			reason: "Should return ResourceExists=false when key is not found",
			fields: fields{
				client: &mockKeyClient{
					MockGetKeyByName: func(ctx context.Context, name string) (*garage.Key, error) {
						return nil, errors.New("not found")
					},
				},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-key",
					},
					Spec: v1alpha1.KeySpec{
						ForProvider: v1alpha1.KeyParameters{
							Name: "test-key",
						},
					},
					Status: v1alpha1.KeyStatus{
						AtProvider: v1alpha1.KeyObservation{
							AccessKeyID: "",
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: false,
				},
				err: nil,
			},
		},
		"KeyExistsByExternalName": {
			reason: "Should return ResourceExists=true when key exists and is found by external-name annotation",
			fields: fields{
				client: &mockKeyClient{
					MockGetKey: func(ctx context.Context, accessKeyID string) (*garage.Key, error) {
						if accessKeyID == "GK123456" {
							return &garage.Key{
								AccessKeyID: "GK123456",
								Name:        "test-key",
							}, nil
						}
						return nil, errors.New("not found")
					},
				},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-key",
						Annotations: map[string]string{
							"crossplane.io/external-name": "GK123456",
						},
					},
					Spec: v1alpha1.KeySpec{
						ForProvider: v1alpha1.KeyParameters{
							Name: "test-key",
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
					ConnectionDetails: managed.ConnectionDetails{
						"accessKeyId": []byte("GK123456"),
					},
				},
				err: nil,
			},
		},
		"KeyExistsByStatusID": {
			reason: "Should return ResourceExists=true when key is found by Status.AtProvider.AccessKeyID",
			fields: fields{
				client: &mockKeyClient{
					MockGetKey: func(ctx context.Context, accessKeyID string) (*garage.Key, error) {
						if accessKeyID == "GK789" {
							return &garage.Key{
								AccessKeyID: "GK789",
								Name:        "test-key",
							}, nil
						}
						return nil, errors.New("not found")
					},
				},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-key",
					},
					Spec: v1alpha1.KeySpec{
						ForProvider: v1alpha1.KeyParameters{
							Name: "test-key",
						},
					},
					Status: v1alpha1.KeyStatus{
						AtProvider: v1alpha1.KeyObservation{
							AccessKeyID: "GK789",
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
					ConnectionDetails: managed.ConnectionDetails{
						"accessKeyId": []byte("GK789"),
					},
				},
				err: nil,
			},
		},
		"KeyExistsByName_RecoveryScenario": {
			reason: "Should return ResourceExists=true and adopt key when found by name (recovery scenario after crash)",
			fields: fields{
				client: &mockKeyClient{
					MockGetKeyByName: func(ctx context.Context, name string) (*garage.Key, error) {
						if name == "waves-engine-key" {
							return &garage.Key{
								AccessKeyID: "GK_RECOVERED",
								Name:        "waves-engine-key",
							}, nil
						}
						return nil, errors.New("not found")
					},
				},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "waves-engine-key",
						// external-name defaults to resource name when not set
						Annotations: map[string]string{
							"crossplane.io/external-name": "waves-engine-key",
						},
					},
					Spec: v1alpha1.KeySpec{
						ForProvider: v1alpha1.KeyParameters{
							Name: "waves-engine-key",
						},
					},
					Status: v1alpha1.KeyStatus{
						AtProvider: v1alpha1.KeyObservation{
							AccessKeyID: "", // Empty - simulating lost state
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
					ConnectionDetails: managed.ConnectionDetails{
						"accessKeyId": []byte("GK_RECOVERED"),
					},
				},
				err: nil,
			},
		},
		"KeyDeletedExternally": {
			reason: "Should return ResourceExists=false when key was deleted externally",
			fields: fields{
				client: &mockKeyClient{
					MockGetKey: func(ctx context.Context, accessKeyID string) (*garage.Key, error) {
						return nil, errors.New("not found")
					},
					MockGetKeyByName: func(ctx context.Context, name string) (*garage.Key, error) {
						return nil, errors.New("not found")
					},
				},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "deleted-key",
						Annotations: map[string]string{
							"crossplane.io/external-name": "GK_DELETED",
						},
					},
					Spec: v1alpha1.KeySpec{
						ForProvider: v1alpha1.KeyParameters{
							Name: "deleted-key",
						},
					},
					Status: v1alpha1.KeyStatus{
						AtProvider: v1alpha1.KeyObservation{
							AccessKeyID: "GK_DELETED",
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: false,
				},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &mockExternal{client: tc.fields.client}
			got, err := e.Observe(context.Background(), tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestKeyCreate(t *testing.T) {
	type fields struct {
		client keyClient
	}

	type args struct {
		mg resource.Managed
	}

	type want struct {
		o   managed.ExternalCreation
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"SuccessfulCreate": {
			reason: "Should successfully create a key and return connection details",
			fields: fields{
				client: &mockKeyClient{
					MockCreateKey: func(ctx context.Context, req *garage.CreateKeyRequest) (*garage.Key, error) {
						return &garage.Key{
							AccessKeyID:     "GK123456",
							Name:            "test-key",
							SecretAccessKey: "secret123",
						}, nil
					},
				},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-key",
					},
					Spec: v1alpha1.KeySpec{
						ForProvider: v1alpha1.KeyParameters{
							Name: "test-key",
						},
					},
				},
			},
			want: want{
				o: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						"accessKeyId":     []byte("GK123456"),
						"secretAccessKey": []byte("secret123"),
					},
				},
				err: nil,
			},
		},
		"CreateError": {
			reason: "Should return error when create fails",
			fields: fields{
				client: &mockKeyClient{
					MockCreateKey: func(ctx context.Context, req *garage.CreateKeyRequest) (*garage.Key, error) {
						return nil, errors.New("create failed")
					},
				},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-key",
					},
					Spec: v1alpha1.KeySpec{
						ForProvider: v1alpha1.KeyParameters{
							Name: "test-key",
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errors.New("create failed"), errCreateKey),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &mockExternal{client: tc.fields.client}
			got, err := e.Create(context.Background(), tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Create(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Create(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestKeyDelete(t *testing.T) {
	type fields struct {
		client keyClient
	}

	type args struct {
		mg resource.Managed
	}

	type want struct {
		o   managed.ExternalDelete
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"SuccessfulDelete": {
			reason: "Should successfully delete a key using Status.AtProvider.AccessKeyID",
			fields: fields{
				client: &mockKeyClient{
					MockDeleteKey: func(ctx context.Context, accessKeyID string) error {
						return nil
					},
				},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-key",
					},
					Status: v1alpha1.KeyStatus{
						AtProvider: v1alpha1.KeyObservation{
							AccessKeyID: "GK123456",
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
		"DeleteUsingExternalName": {
			reason: "Should successfully delete a key using external-name annotation when status is empty",
			fields: fields{
				client: &mockKeyClient{
					MockDeleteKey: func(ctx context.Context, accessKeyID string) error {
						if accessKeyID != "GK_EXT_NAME" {
							return errors.New("wrong key ID")
						}
						return nil
					},
				},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-key",
						Annotations: map[string]string{
							"crossplane.io/external-name": "GK_EXT_NAME",
						},
					},
					Status: v1alpha1.KeyStatus{
						AtProvider: v1alpha1.KeyObservation{
							AccessKeyID: "", // Empty status
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
		"DeleteNonExistentKey": {
			reason: "Should not error when deleting key with no ID and no external-name",
			fields: fields{
				client: &mockKeyClient{},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-key",
						// external-name defaults to resource name
						Annotations: map[string]string{
							"crossplane.io/external-name": "test-key",
						},
					},
					Status: v1alpha1.KeyStatus{
						AtProvider: v1alpha1.KeyObservation{
							AccessKeyID: "",
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
		"DeleteError": {
			reason: "Should return error when delete fails",
			fields: fields{
				client: &mockKeyClient{
					MockDeleteKey: func(ctx context.Context, accessKeyID string) error {
						return errors.New("delete failed")
					},
				},
			},
			args: args{
				mg: &v1alpha1.Key{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-key",
					},
					Status: v1alpha1.KeyStatus{
						AtProvider: v1alpha1.KeyObservation{
							AccessKeyID: "GK123456",
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.Wrap(errors.New("delete failed"), errDeleteKey),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &mockExternal{client: tc.fields.client}
			got, err := e.Delete(context.Background(), tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Delete(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Delete(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}
