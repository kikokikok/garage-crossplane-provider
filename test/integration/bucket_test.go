//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/kikokikok/provider-garage/apis"
	"github.com/kikokikok/provider-garage/apis/v1alpha1"
	"github.com/kikokikok/provider-garage/apis/v1beta1"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Test Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../package/crds"},
		ErrorIfCRDPathMissing: false,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = apis.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Bucket Controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	Context("When creating a Bucket", func() {
		It("Should create the Bucket resource successfully", func() {
			ctx := context.Background()

			// Create a namespace for testing
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-bucket-namespace",
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create ProviderConfig secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "garage-creds",
					Namespace: "default",
				},
				StringData: map[string]string{
					"credentials": `{"endpoint":"http://localhost:3903","adminToken":"test-token"}`,
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			// Create ProviderConfig
			pc := &v1beta1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-provider-config",
				},
				Spec: v1beta1.ProviderConfigSpec{
					Credentials: v1beta1.ProviderCredentials{
						Source: "Secret",
						CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
							SecretRef: &xpv1.SecretKeySelector{
								SecretReference: xpv1.SecretReference{
									Name:      "garage-creds",
									Namespace: "default",
								},
								Key: "credentials",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pc)).Should(Succeed())

			// Create Bucket
			globalAlias := "test-integration-bucket"
			bucket := &v1alpha1.Bucket{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-bucket",
					Namespace: "test-bucket-namespace",
				},
				Spec: v1alpha1.BucketSpec{
					ForProvider: v1alpha1.BucketParameters{
						GlobalAlias: &globalAlias,
					},
				},
			}
			bucket.Spec.ProviderConfigReference = &xpv1.Reference{Name: "test-provider-config"}

			Expect(k8sClient.Create(ctx, bucket)).Should(Succeed())

			// Verify bucket was created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      "test-bucket",
					Namespace: "test-bucket-namespace",
				}, bucket)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
