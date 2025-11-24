package bucket

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	"github.com/kikokikok/provider-garage/apis/v1alpha1"
	"github.com/kikokikok/provider-garage/pkg/garage"
)

// garageClient interface to allow mocking
type garageClient interface {
	CreateBucket(ctx context.Context, req *garage.CreateBucketRequest) (*garage.Bucket, error)
	GetBucket(ctx context.Context, bucketID string) (*garage.Bucket, error)
	UpdateBucket(ctx context.Context, req *garage.UpdateBucketRequest) (*garage.Bucket, error)
	DeleteBucket(ctx context.Context, bucketID string) error
}

type mockGarageClient struct {
	MockCreateBucket func(ctx context.Context, req *garage.CreateBucketRequest) (*garage.Bucket, error)
	MockGetBucket    func(ctx context.Context, bucketID string) (*garage.Bucket, error)
	MockUpdateBucket func(ctx context.Context, req *garage.UpdateBucketRequest) (*garage.Bucket, error)
	MockDeleteBucket func(ctx context.Context, bucketID string) error
}

func (m *mockGarageClient) CreateBucket(ctx context.Context, req *garage.CreateBucketRequest) (*garage.Bucket, error) {
	return m.MockCreateBucket(ctx, req)
}

func (m *mockGarageClient) GetBucket(ctx context.Context, bucketID string) (*garage.Bucket, error) {
	return m.MockGetBucket(ctx, bucketID)
}

func (m *mockGarageClient) UpdateBucket(ctx context.Context, req *garage.UpdateBucketRequest) (*garage.Bucket, error) {
	return m.MockUpdateBucket(ctx, req)
}

func (m *mockGarageClient) DeleteBucket(ctx context.Context, bucketID string) error {
	return m.MockDeleteBucket(ctx, bucketID)
}

// Mock external for testing
type mockExternal struct {
	client garageClient
}

func (e *mockExternal) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Bucket)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBucket)
	}

	if cr.Status.AtProvider.ID == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	bucket, err := e.client.GetBucket(ctx, cr.Status.AtProvider.ID)
	if err != nil {
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

func (e *mockExternal) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
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

func (e *mockExternal) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (e *mockExternal) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Bucket)
	if !ok {
		return errors.New(errNotBucket)
	}

	cr.SetConditions(xpv1.Deleting())

	if cr.Status.AtProvider.ID == "" {
		return nil
	}

	return errors.Wrap(e.client.DeleteBucket(ctx, cr.Status.AtProvider.ID), errDeleteBucket)
}

func TestObserve(t *testing.T) {
	type fields struct {
		client garageClient
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
		"BucketDoesNotExist": {
			reason: "Should return ResourceExists=false when bucket ID is empty",
			fields: fields{
				client: &mockGarageClient{},
			},
			args: args{
				mg: &v1alpha1.Bucket{
					Status: v1alpha1.BucketStatus{
						AtProvider: v1alpha1.BucketObservation{
							ID: "",
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
		"BucketExistsAndUpToDate": {
			reason: "Should return ResourceExists=true when bucket exists",
			fields: fields{
				client: &mockGarageClient{
					MockGetBucket: func(ctx context.Context, bucketID string) (*garage.Bucket, error) {
						return &garage.Bucket{
							ID:            "bucket-123",
							GlobalAliases: []string{"test-bucket"},
						}, nil
					},
				},
			},
			args: args{
				mg: &v1alpha1.Bucket{
					Status: v1alpha1.BucketStatus{
						AtProvider: v1alpha1.BucketObservation{
							ID: "bucket-123",
						},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
			},
		},
		"BucketGetError": {
			reason: "Should return ResourceExists=false when get fails",
			fields: fields{
				client: &mockGarageClient{
					MockGetBucket: func(ctx context.Context, bucketID string) (*garage.Bucket, error) {
						return nil, errors.New("not found")
					},
				},
			},
			args: args{
				mg: &v1alpha1.Bucket{
					Status: v1alpha1.BucketStatus{
						AtProvider: v1alpha1.BucketObservation{
							ID: "bucket-123",
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

func TestCreate(t *testing.T) {
	type fields struct {
		client garageClient
	}

	type args struct {
		mg resource.Managed
	}

	type want struct {
		o   managed.ExternalCreation
		err error
	}

	globalAlias := "test-bucket"

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"SuccessfulCreate": {
			reason: "Should successfully create a bucket",
			fields: fields{
				client: &mockGarageClient{
					MockCreateBucket: func(ctx context.Context, req *garage.CreateBucketRequest) (*garage.Bucket, error) {
						return &garage.Bucket{
							ID:            "bucket-123",
							GlobalAliases: []string{"test-bucket"},
						}, nil
					},
				},
			},
			args: args{
				mg: &v1alpha1.Bucket{
					Spec: v1alpha1.BucketSpec{
						ForProvider: v1alpha1.BucketParameters{
							GlobalAlias: &globalAlias,
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: nil,
			},
		},
		"CreateError": {
			reason: "Should return error when create fails",
			fields: fields{
				client: &mockGarageClient{
					MockCreateBucket: func(ctx context.Context, req *garage.CreateBucketRequest) (*garage.Bucket, error) {
						return nil, errors.New("create failed")
					},
				},
			},
			args: args{
				mg: &v1alpha1.Bucket{
					Spec: v1alpha1.BucketSpec{
						ForProvider: v1alpha1.BucketParameters{
							GlobalAlias: &globalAlias,
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errors.New("create failed"), errCreateBucket),
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

func TestDelete(t *testing.T) {
	type fields struct {
		client garageClient
	}

	type args struct {
		mg resource.Managed
	}

	type want struct {
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"SuccessfulDelete": {
			reason: "Should successfully delete a bucket",
			fields: fields{
				client: &mockGarageClient{
					MockDeleteBucket: func(ctx context.Context, bucketID string) error {
						return nil
					},
				},
			},
			args: args{
				mg: &v1alpha1.Bucket{
					Status: v1alpha1.BucketStatus{
						AtProvider: v1alpha1.BucketObservation{
							ID: "bucket-123",
						},
					},
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteNonExistentBucket": {
			reason: "Should not error when deleting bucket with no ID",
			fields: fields{
				client: &mockGarageClient{},
			},
			args: args{
				mg: &v1alpha1.Bucket{
					Status: v1alpha1.BucketStatus{
						AtProvider: v1alpha1.BucketObservation{
							ID: "",
						},
					},
				},
			},
			want: want{
				err: nil,
			},
		},
		"DeleteError": {
			reason: "Should return error when delete fails",
			fields: fields{
				client: &mockGarageClient{
					MockDeleteBucket: func(ctx context.Context, bucketID string) error {
						return errors.New("delete failed")
					},
				},
			},
			args: args{
				mg: &v1alpha1.Bucket{
					Status: v1alpha1.BucketStatus{
						AtProvider: v1alpha1.BucketObservation{
							ID: "bucket-123",
						},
					},
				},
			},
			want: want{
				err: errors.Wrap(errors.New("delete failed"), errDeleteBucket),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &mockExternal{client: tc.fields.client}
			err := e.Delete(context.Background(), tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Delete(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}
