package k8s

import (
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
)

// Defaults are used when no configuration is provided via OperatorOptions.
const (
	DefaultNamespace       = "tka-dev"
	DefaultClusterName     = "tka-cluster"
	DefaultContextPrefix   = "tka-context-"
	DefaultUserEntryPrefix = "tka-user-"

	// MinSigninValidity is the minimum validity period for a token in Kubernetes. This minimum period is enforced by the Kubernetes API.
	MinSigninValidity = 10 * time.Minute
)

var NotReadyYetError = humane.New("Not ready yet", "Please wait for the TKA signin to be ready")
