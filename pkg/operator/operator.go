// Package operator provides Kubernetes operator functionality for the TKA service.
// This package implements the Kubernetes controller that manages TKASignIn custom
// resources and provisions user credentials within the cluster. It handles the
// lifecycle of authentication credentials and integrates with the Kubernetes API.
package operator

import (
	"context"

	"github.com/go-logr/zapr"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/api/v1alpha1"
	"github.com/spechtlabs/tka/pkg/client/k8s"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/internal/utils"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

var scheme = runtime.NewScheme()

type KubeOperator struct {
	mgr    ctrl.Manager
	tracer trace.Tracer
	client k8s.TkaClient
}

func newControllerManagedBy() (ctrl.Manager, humane.Error) {
	// GetConfigOrDie creates a *rest.Config for talking to a Kubernetes API tailscale.
	// If --kubeconfig.go is set, will use the kubeconfig.go file at that location.  Otherwise will assume running
	// in cluster and use the cluster provided kubeconfig.go.
	//
	// The returned `*rest.Config` has client-side ratelimting disabled as we can rely on API priority and
	// fairness. Set its QPS to a value equal or bigger than 0 to re-enable it.
	//
	// It also applies saner defaults for QPS and burst based on the Kubernetes
	// controller manager defaults (20 QPS, 30 burst)
	//
	// Config precedence:
	//
	// * --kubeconfig.go flag pointing at a file
	//
	// * KUBECONFIG environment variable pointing at a file
	//
	// * In-cluster config if running in cluster
	//
	// * $HOME/.kube/config if exists.

	// If we run in-cluster then we also do leader election.
	// But for local debugging, that's not needed
	inCluster := isInCluster()
	leaderElectionNamespace := ""
	if !inCluster {
		leaderElectionNamespace = "default"
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		HealthProbeBindAddress:  "0",
		LeaderElection:          inCluster,
		LeaderElectionNamespace: leaderElectionNamespace,
		LeaderElectionID:        "controller.tka.specht-labs.de",
		Metrics: server.Options{
			BindAddress: "0",
		},
	})
	if err != nil {
		return nil, humane.Wrap(err, "failed to create manager")
	}

	return mgr, nil
}

func newKubeOperator(mgr ctrl.Manager, clientOpts k8s.ClientOptions) (*KubeOperator, humane.Error) {
	op := &KubeOperator{
		mgr:    mgr,
		tracer: otel.Tracer("tka_controller"),
		client: k8s.NewTkaClient(mgr.GetClient(), otel.Tracer("tka_controller"), mgr.GetConfig(), clientOpts),
	}

	if err := ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.TkaSignin{}).Named("TkaSignin").Complete(op); err != nil {
		return nil, humane.Wrap(err, "failed to register controller manager")
	}

	return op, nil
}

func NewK8sOperator(clientOpts k8s.ClientOptions) (*KubeOperator, humane.Error) {
	// Register the schemes
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, humane.Wrap(err, "failed to add clientgoscheme to scheme")
	}

	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, humane.Wrap(err, "failed to add v1alpha1 to scheme")
	}

	ctrl.SetLogger(zapr.NewLogger(otelzap.L().Logger))

	mgr, err := newControllerManagedBy()
	if err != nil {
		return nil, err
	}

	if ok, err := utils.IsK8sVerAtLeast(mgr.GetConfig(), 1, 24); err != nil {
		return nil, err
	} else if !ok {
		return nil, humane.New("k8s version must be at least 1.24")
	}

	op, err := newKubeOperator(mgr, clientOpts)
	if err != nil {
		otelzap.L().WithError(err).Error("failed to create kube operator")
		return nil, err
	}

	return op, nil
}

func (t *KubeOperator) Start(ctx context.Context) humane.Error {
	if err := t.mgr.Start(ctx); err != nil {
		otelzap.L().WithError(err).Error("failed to start manager")
		return humane.Wrap(err, "failed to start manager")
	}

	return nil
}

func (t *KubeOperator) GetClient() k8s.TkaClient {
	return t.client
}
