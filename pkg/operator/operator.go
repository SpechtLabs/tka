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
	"github.com/spechtlabs/tka/pkg/service/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/internal/utils"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

var scheme = runtime.NewScheme()

// KubeOperator is the Kubernetes controller that manages TkaSignin custom resources.
// It handles provisioning and deprovisioning of user credentials based on sign-in requests.
type KubeOperator struct {
	mgr    ctrl.Manager
	tracer trace.Tracer
	client k8s.TkaClient
}

func getConfigOrDie() *rest.Config {
	config, err := ctrl.GetConfig()
	if err != nil {
		herr := humane.Wrap(err, "Failed to get Kubernetes config",
			"If --kubeconfig.go is set, will use the kubeconfig.go file at that location. Otherwise will assume running in cluster and use the cluster provided kubeconfig.go.",
			"Check the config precedence: 1) --kubeconfig.go flag pointing at a file 2) KUBECONFIG environment variable pointing at a file 3) In-cluster config if running in cluster 4) $HOME/.kube/config if exists.",
		)

		otelzap.L().WithError(herr).Fatal("Failed to get Kubernetes config")
	}

	return config
}

func newControllerManagedBy() (ctrl.Manager, humane.Error) {
	// If we run in-cluster then we also do leader election.
	// But for local debugging, that's not needed
	inCluster := isInCluster()
	leaderElectionNamespace := ""
	if !inCluster {
		leaderElectionNamespace = "default"
	}

	mgr, err := ctrl.NewManager(getConfigOrDie(), ctrl.Options{
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

func newKubeOperator(mgr ctrl.Manager, clusterInfo *models.TkaClusterInfo, clientOpts k8s.ClientOptions) (*KubeOperator, humane.Error) {
	op := &KubeOperator{
		mgr:    mgr,
		tracer: otel.Tracer("tka_controller"),
		client: k8s.NewTkaClient(mgr.GetClient(), clusterInfo, clientOpts),
	}

	if err := ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.TkaSignin{}).Named("TkaSignin").Complete(op); err != nil {
		return nil, humane.Wrap(err, "failed to register controller manager")
	}

	return op, nil
}

// NewK8sOperator creates and initializes a new KubeOperator with the provided
// cluster information and client configuration options.
func NewK8sOperator(clusterInfo *models.TkaClusterInfo, clientOpts k8s.ClientOptions) (*KubeOperator, humane.Error) {
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

	if ok, err := utils.IsK8sVerAtLeast(1, 24); err != nil {
		return nil, err
	} else if !ok {
		return nil, humane.New("k8s version must be at least 1.24")
	}

	op, err := newKubeOperator(mgr, clusterInfo, clientOpts)
	if err != nil {
		return nil, err
	}

	return op, nil
}

func (t *KubeOperator) Start(ctx context.Context) humane.Error {
	if err := t.mgr.Start(ctx); err != nil {
		return humane.Wrap(err, "failed to start manager")
	}

	return nil
}

func (t *KubeOperator) GetClient() k8s.TkaClient {
	return t.client
}
