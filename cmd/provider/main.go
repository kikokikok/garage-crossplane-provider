// Package main is the entry point for the Garage Crossplane provider
package main

import (
	"os"
	"path/filepath"
	"time"

	kingpin "github.com/alecthomas/kingpin/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/feature"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"

	"github.com/kikokikok/provider-garage/apis"
	"github.com/kikokikok/provider-garage/internal/controller/bucket"
)

func main() {
	var (
		app              = kingpin.New(filepath.Base(os.Args[0]), "Garage Crossplane Provider").DefaultEnvars()
		debug            = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncInterval     = app.Flag("sync", "Sync interval for controllers.").Short('s').Default("1h").Duration()
		pollInterval     = app.Flag("poll", "Poll interval for managed resources.").Default("1m").Duration()
		leaderElection   = app.Flag("leader-election", "Use leader election for controllers.").Short('l').Default("false").Envar("LEADER_ELECTION").Bool()
		maxReconcileRate = app.Flag("max-reconcile-rate", "Maximum rate of reconciliation per controller.").Default("10").Int()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithValues("provider", "garage"))
	ctrl.SetLogger(zl)

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Cache: cache.Options{
			SyncPeriod: syncInterval,
		},
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-provider-garage",
		LeaseDuration:    func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:    func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add Garage APIs to scheme")

	o := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
		Features:                &feature.Flags{},
	}

	kingpin.FatalIfError(bucket.Setup(mgr, o), "Cannot setup Bucket controller")

	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
