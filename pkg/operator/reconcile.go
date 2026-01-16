package operator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/api/v1alpha1"
	"github.com/spechtlabs/tka/pkg/client/k8s"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// SignInOperation represents the type of action to perform during reconciliation.
type SignInOperation int

// SignInOperation constants define the possible reconciliation actions.
const (
	// SignInOperationProvision creates or updates user credentials.
	SignInOperationProvision SignInOperation = iota
	// SignInOperationDeprovision removes user credentials.
	SignInOperationDeprovision
	// SignInOperationNOP indicates no operation is needed.
	SignInOperationNOP
)

// reconcileEvent captures all context for a single reconciliation wide event.
type reconcileEvent struct {
	name       string
	namespace  string
	username   string
	operation  string
	success    bool
	err        error
	requeueIn  time.Duration
	durationMs int64
}

// +kubebuilder:rbac:groups=tka.specht-labs.de,resources=TkaSignin,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tka.specht-labs.de,resources=TkaSignin/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tka.specht-labs.de,resources=TkaSignin/finalizers,verbs=update

func (t *KubeOperator) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	startTime := time.Now()
	ctx, span := t.tracer.Start(ctx, "KubeOperator.Reconcile")

	// Initialize wide event context
	event := &reconcileEvent{
		name:      req.Name,
		namespace: req.Namespace,
		operation: "unknown",
		success:   true,
	}

	// Emit wide event data to span attributes and log at the end
	defer func() {
		event.durationMs = time.Since(startTime).Milliseconds()
		reconcilerDuration.WithLabelValues("user", req.Name, req.Namespace).Observe(float64(time.Since(startTime).Microseconds()))

		// Set span attributes for wide event data
		span.SetAttributes(
			attribute.String("reconcile.name", event.name),
			attribute.String("reconcile.namespace", event.namespace),
			attribute.String("reconcile.username", event.username),
			attribute.String("reconcile.operation", event.operation),
			attribute.Bool("reconcile.success", event.success),
			attribute.Int64("reconcile.duration_ms", event.durationMs),
		)

		if event.requeueIn > 0 {
			span.SetAttributes(attribute.Int64("reconcile.requeue_in_ms", event.requeueIn.Milliseconds()))
		}

		if event.err != nil {
			span.SetStatus(codes.Error, event.err.Error())
			span.RecordError(event.err)
			otelzap.L().WithError(event.err).ErrorContext(ctx, "reconcile completed with error")
		}

		span.End()
	}()

	c := t.mgr.GetClient()

	// Grab and process the signin object first
	signIn := &v1alpha1.TkaSignin{}
	if err := c.Get(ctx, req.NamespacedName, signIn); err != nil {
		if k8serrors.IsNotFound(err) {
			signIn = &v1alpha1.TkaSignin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      req.Name,
					Namespace: req.Namespace,
				},
				Spec: v1alpha1.TkaSigninSpec{
					Username: strings.TrimPrefix(req.Name, k8s.DefaultUserEntryPrefix),
				},
			}
			event.username = signIn.Spec.Username
			event.operation = "deprovision_not_found"

		if err := t.signOutUser(ctx, signIn); err != nil {
			event.success = false
			event.err = err
			return reconcile.Result{}, fmt.Errorf("failed to deprovision deleted signin %s: %w", req.Name, err)
		}
		return reconcile.Result{}, nil
	}

	event.success = false
	event.err = err
	event.operation = "get_signin_failed"
	return reconcile.Result{}, fmt.Errorf("failed to get signin %s: %w", req.NamespacedName, err)
}

	event.username = signIn.Spec.Username

	op, validDuration := getAction(signIn, span)
	event.requeueIn = validDuration

	switch op {
	case SignInOperationProvision:
		event.operation = "provision"
		if err := t.signInUser(ctx, signIn); err != nil {
			event.success = false
			event.err = err
			return reconcile.Result{}, fmt.Errorf("failed to provision signin %s: %w", signIn.Name, err)
		}

	case SignInOperationDeprovision:
		event.operation = "deprovision"
		if err := t.signOutUser(ctx, signIn); err != nil {
			event.success = false
			event.err = err
			return reconcile.Result{}, fmt.Errorf("failed to deprovision signin %s: %w", signIn.Name, err)
		}

	case SignInOperationNOP:
		event.operation = "nop"

	default:
		event.operation = "unknown"
	}

	return reconcile.Result{RequeueAfter: validDuration}, nil
}

func getAction(signIn *v1alpha1.TkaSignin, span trace.Span) (SignInOperation, time.Duration) {
	validity, err := time.ParseDuration(signIn.Spec.ValidityPeriod)
	if err != nil {
		span.AddEvent("parse_validity_period_failed")
		return SignInOperationNOP, time.Duration(0)
	}

	// If a new signin is not yet provisioned - use the reconciler loop to deploy the SA and CRB
	if !signIn.Status.Provisioned {
		span.AddEvent("not_provisioned")
		return SignInOperationProvision, validity
	}

	// If SignIn is expired
	validUntil, err := time.Parse(time.RFC3339, signIn.Status.ValidUntil)
	if err != nil {
		span.AddEvent("parse_valid_until_failed")
		return SignInOperationNOP, time.Duration(0)
	}

	if time.Now().UTC().After(validUntil.UTC()) {
		span.AddEvent("signin_expired")
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
		span.AddEvent("parse_signed_in_at_failed")
		return SignInOperationNOP, time.Duration(0)
	}

	signedInUntilDuration, err := time.ParseDuration(signIn.Spec.ValidityPeriod)
	if err != nil {
		span.AddEvent("parse_validity_duration_failed")
		return SignInOperationNOP, time.Duration(0)
	}

	statusValidUntil := signedInAt.Add(signedInUntilDuration)
	if !statusValidUntil.Equal(validUntil) {
		span.AddEvent("login_extended")
		return SignInOperationProvision, time.Duration(0)
	}

	return SignInOperationNOP, time.Duration(0)
}
