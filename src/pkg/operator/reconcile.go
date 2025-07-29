package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/spechtlabs/go-otel-utils/otelzap"
)

type SignInOperation int

const (
	SignInOperationProvision SignInOperation = iota
	SignInOperationDeprovision
	SignInOperationNOP
)

// +kubebuilder:rbac:groups=tka.specht-labs.de,resources=TkaSignin,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tka.specht-labs.de,resources=TkaSignin/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tka.specht-labs.de,resources=TkaSignin/finalizers,verbs=update

func (t *KubeOperator) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	startTime := time.Now()
	defer func() {
		reconcilerDuration.WithLabelValues("user", req.Name, req.Namespace).Observe(float64(time.Since(startTime).Microseconds()))
	}()

	ctx, span := t.tracer.Start(ctx, "KubeOperator.Reconcile")
	defer span.End()

	c := t.mgr.GetClient()

	// Grab and process the signin object first
	signIn := &v1alpha1.TkaSignin{}
	if err := c.Get(ctx, req.NamespacedName, signIn); err != nil || signIn == nil {
		if k8serrors.IsNotFound(err) {
			otelzap.L().Info("signin deleted", zap.String("name", req.Name), zap.String("namespace", req.Namespace))
			return reconcile.Result{}, nil
		}

		otelzap.L().WithError(err).Error("failed to get tka signin", zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		return reconcile.Result{}, err
	}

	validDuration, err := t.processSignIn(ctx, signIn)
	if err != nil {
		otelzap.L().WithError(err.Cause()).Error(err.Error(), zap.String("username", signIn.Spec.Username))
		return reconcile.Result{}, fmt.Errorf("%s", err.Display())
	}

	return reconcile.Result{RequeueAfter: validDuration}, nil
}

func (t *KubeOperator) processSignIn(ctx context.Context, signIn *v1alpha1.TkaSignin) (time.Duration, humane.Error) {
	op, duration := getAction(signIn)
	switch op {
	case SignInOperationProvision:
		if err := t.signInUser(ctx, signIn); err != nil {
			return duration, humane.Wrap(err, "failed to sign in user")
		}

	case SignInOperationDeprovision:
		if err := t.LogOutUser(ctx, signIn.Spec.Username); err != nil {
			return duration, humane.Wrap(err, "failed to log out user")
		}
	case SignInOperationNOP:
		otelzap.L().Debug("signin operation nop", zap.String("username", signIn.Spec.Username))
	}

	return duration, nil
}

func getAction(signIn *v1alpha1.TkaSignin) (SignInOperation, time.Duration) {
	validUntil, err := time.Parse(time.RFC3339, signIn.Spec.ValidUntil)
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to parse validUntil")
		return SignInOperationNOP, time.Duration(0)
	}

	now := time.Now()

	// If signin is stale, remove
	if validUntil.After(now.UTC()) {
		return SignInOperationDeprovision, time.Duration(0)
	}

	// If a new signin is not yet provisioned - use the reconciler loop to deploy the SA and CRB
	if signIn.Status.Provisioned == false {
		return SignInOperationProvision, validUntil.Sub(now.UTC())
	}

	return SignInOperationNOP, time.Duration(0)
}
