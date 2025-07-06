package operator

import (
	"context"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
)

func (t *KubeOperator) SignInUser(ctx context.Context, userName, role string, validUntil time.Time) humane.Error {
	ctx, span := t.tracer.Start(ctx, "KubeOperator.SignInUser")
	defer span.End()

	//client := t.mgr.GetClient()
	//scheme := t.mgr.GetScheme()

	// TODO
	// 1. Create Service Account
	// 2. get kubernetes version
	// 3. if version >= 1.30 then generate token for Service Account
	// 4. create cluster-role-binding
	// 5. ensure cleanup-job happens once time is done for

	// Notes:
	// Attach the validUntil value to the ServiceAccount && Cluster-Role-Binding

	return nil
}
