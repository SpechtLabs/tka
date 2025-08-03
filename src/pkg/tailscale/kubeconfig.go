package tailscale

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/operator"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"tailscale.com/tailcfg"
)

func (t *TKAServer) getKubeconfig(ct *gin.Context) {
	req := ct.Request

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.getKubeconfig")
	defer span.End()

	// This URL is visited by the user who is being authenticated. If they are
	// visiting the URL over Funnel, that means they are not part of the
	// tailnet that they are trying to be authenticated for.
	if IsFunnelRequest(ct.Request) {
		otelzap.L().ErrorContext(ctx, "Unauthorized request from Funnel")
		ct.JSON(http.StatusForbidden, NewErrorResponse("Unauthorized request from Funnel", nil))
		return
	}

	who, err := t.lc.WhoIs(ctx, req.RemoteAddr)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting WhoIs")
		ct.JSON(http.StatusInternalServerError, NewErrorResponse("Error getting WhoIs", err))
		return
	}

	// not sure if this is the right thing to do...
	userName, _, _ := strings.Cut(who.UserProfile.LoginName, "@")
	n := who.Node.View()
	if n.IsTagged() {
		otelzap.L().ErrorContext(ctx, "tagged nodes not (yet) supported")
		ct.JSON(http.StatusBadRequest, NewErrorResponse("tagged nodes not (yet) supported", nil))
		return
	}

	rules, err := tailcfg.UnmarshalCapJSON[capRule](who.CapMap, t.capName)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error unmarshaling capability")
		ct.JSON(http.StatusBadRequest, FromHumaneError(humane.Wrap(err, "Error unmarshaling tailscale capability map", "Check the syntax of your tailscale ACL for user "+userName+".")))
		return
	}

	if len(rules) == 0 {
		otelzap.L().ErrorContext(ctx, "No capability rule found for user. Assuming unauthorized.")
		ct.JSON(http.StatusForbidden, NewErrorResponse("User not authorized", nil))
		return
	}

	if len(rules) > 1 {
		// TODO(cedi): unsure what to do when having more than one cap...
		otelzap.L().ErrorContext(ctx, "More than one capability rule found")
		ct.JSON(http.StatusBadRequest, FromHumaneError(humane.New("More than one capability rule found", "Please ensure that you only have one capability rule for your user.", "If you have more than one, please contact the administrator of this system.")))
		return
	}

	if kubecfg, err := t.operator.GetKubeconfig(ctx, userName); err != nil || kubecfg == nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting kubeconfig")

		if errors.Is(err, operator.NotReadyYetError) {
			ct.JSON(http.StatusProcessing, FromHumaneError(err))
			return
		}

		if err.Cause() != nil && k8serrors.IsNotFound(err.Cause()) {
			ct.JSON(http.StatusUnauthorized, FromHumaneError(err))
			return
		} else {
			ct.JSON(http.StatusInternalServerError, FromHumaneError(err))
			return
		}
	} else {
		ct.JSON(http.StatusOK, *kubecfg)
		return
	}
}
