package controller

import (
	"context"
	"time"

	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TKAReconciler struct {
	client client.Client

	scheme *runtime.Scheme
	tracer trace.Tracer
}

func NewTKAReconciler(client client.Client, scheme *runtime.Scheme) *TKAReconciler {
	return &TKAReconciler{
		client: client,
		scheme: scheme,
		tracer: otel.Tracer("tka_controller"),
	}
}

// SetupWithManager sets up the controller with the Manager.
func (t *TKAReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.TKA{}).
		Named("tailscale-k8s-auth").
		Complete(t)
}

func (t *TKAReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	startTime := time.Now()
	defer func() {
		reconcilerDuration.WithLabelValues("tka", req.Name, req.Namespace).Observe(float64(time.Since(startTime).Microseconds()))
	}()

	//TODO implement me
	panic("implement me")
}
