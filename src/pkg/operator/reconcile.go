package operator

import (
	"context"
	"time"

	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	"go.uber.org/zap"

	rbacv1 "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"go.opentelemetry.io/otel/attribute"
)

func (t *KubeOperator) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	startTime := time.Now()
	defer func() {
		reconcilerDuration.WithLabelValues("tka", req.Name, req.Namespace).Observe(float64(time.Since(startTime).Microseconds()))
	}()

	ctx, span := t.tracer.Start(ctx, "KubeOperator.Reconcile")
	defer span.End()

	client := t.mgr.GetClient()
	scheme := t.mgr.GetScheme()

	tkaConfig := &v1alpha1.TKA{}
	if err := client.Get(ctx, req.NamespacedName, tkaConfig); err != nil {
		otelzap.L().WithError(err).Error("failed to get tka config")
		return reconcile.Result{}, err
	}

	clusterRoles := tkaConfig.Spec.AdditionalClusterRole
	span.SetAttributes(attribute.Int("extra_cluster_roles", len(clusterRoles)))
	for _, clusterRole := range clusterRoles {
		cr := &rbacv1.ClusterRole{}
		if err := client.Get(ctx, runtimeclient.ObjectKeyFromObject(&clusterRole), cr); err == nil {
			otelzap.L().Info("ClusterRole with this name already exists, skipping", zap.String("name", clusterRole.Name))
			continue
		}

		if err := ctrl.SetControllerReference(tkaConfig, &clusterRole, scheme); err != nil {
			otelzap.L().WithError(err).Error("Failed to set controller reference")
			continue
		}

		if err := client.Create(ctx, &clusterRole); err != nil {
			otelzap.L().WithError(err).Error("Failed to create cluster role")
			continue
		}
	}

	return reconcile.Result{}, nil
}
