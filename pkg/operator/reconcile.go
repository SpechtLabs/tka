package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/spechtlabs/tka/api/v1alpha1"
	"github.com/spechtlabs/tka/pkg/client/k8s"
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
	if err := c.Get(ctx, req.NamespacedName, signIn); err != nil {
		if k8serrors.IsNotFound(err) {
			otelzap.L().Info("signin deleted", zap.String("name", req.Name), zap.String("namespace", req.Namespace))
			return reconcile.Result{}, nil
		}

		otelzap.L().WithError(err).Error("failed to get tka signin", zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		return reconcile.Result{}, err
	}

	op, validDuration := getAction(signIn)
	switch op {
	case SignInOperationProvision:
		if err := t.signInUser(ctx, signIn); err != nil {
			otelzap.L().WithError(err).Error("Failed to sign in user", zap.String("username", signIn.Spec.Username))
			return reconcile.Result{}, fmt.Errorf("%s", err.Display())
		}

	case SignInOperationDeprovision:
		if err := t.signOutUser(ctx, signIn); err != nil {
			otelzap.L().WithError(err).Error("Failed to sign out user", zap.String("username", signIn.Spec.Username))
			return reconcile.Result{}, fmt.Errorf("%s", err.Display())
		}

	case SignInOperationNOP:
		otelzap.L().Debug("signin operation nop", zap.String("username", signIn.Spec.Username))

	default:
		otelzap.L().Warn("unknown signin operation", zap.String("username", signIn.Spec.Username))
	}

	return reconcile.Result{RequeueAfter: validDuration}, nil
}

func getAction(signIn *v1alpha1.TkaSignin) (SignInOperation, time.Duration) {
	validity, err := time.ParseDuration(signIn.Spec.ValidityPeriod)
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to parse validity period")
		return SignInOperationNOP, time.Duration(0)
	}

	// If a new signin is not yet provisioned - use the reconciler loop to deploy the SA and CRB
	if !signIn.Status.Provisioned {
		return SignInOperationProvision, validity
	}

	// If SignIn is expired
	validUntil, err := time.Parse(time.RFC3339, signIn.Status.ValidUntil)
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to parse validUntil")
		return SignInOperationNOP, time.Duration(0)
	}

	if time.Now().UTC().After(validUntil.UTC()) {
		return SignInOperationDeprovision, time.Duration(0)
	}

	// If user extended the login
	var signedInAtStr string
	if signedIn, ok := signIn.Annotations[k8s.LastAttemptedSignIn]; ok {
		signedInAtStr = signedIn
	} else {
		signedInAtStr = signIn.Status.SignedInAt
	}

	signedInAt, err := time.Parse(time.RFC3339, signedInAtStr)
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to parse signedInAt")
		return SignInOperationNOP, time.Duration(0)
	}

	signedInUntilDuration, err := time.ParseDuration(signIn.Spec.ValidityPeriod)
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to parse signedInAt")
		return SignInOperationNOP, time.Duration(0)
	}

	statusValidUntil := signedInAt.Add(signedInUntilDuration)
	if !statusValidUntil.Equal(validUntil) {
		otelzap.L().Debug("User extended their login validity", zap.String("username", signIn.Spec.Username))
		return SignInOperationProvision, time.Duration(0)
	}

	return SignInOperationNOP, time.Duration(0)
}
