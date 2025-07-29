package tailscale

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"go.uber.org/zap"
	"tailscale.com/tailcfg"
)

func (t *TKAServer) login(ct *gin.Context) {
	req := ct.Request

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.login")
	defer span.End()

	// This URL is visited by the user who is being authenticated. If they are
	// visiting the URL over Funnel, that means they are not part of the
	// tailnet that they are trying to be authenticated for.
	if IsFunnelRequest(ct.Request) {
		otelzap.L().ErrorContext(ctx, "Unauthorized request from Funnel")
		ct.JSON(http.StatusBadGateway, ErrorResponse{Error: "Unauthorized funnel request"})
		return
	}

	who, err := t.lc.WhoIs(ctx, req.RemoteAddr)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting WhoIs")
		ct.JSON(http.StatusBadGateway, ErrorResponse{Error: "Error getting WhoIs", Cause: err.Error()})
		return
	}

	// not sure if this is the right thing to do...
	userName, _, _ := strings.Cut(who.UserProfile.LoginName, "@")
	n := who.Node.View()
	if n.IsTagged() {
		otelzap.L().ErrorContext(ctx, "tagged nodes not (yet) supported")
		ct.JSON(http.StatusForbidden, ErrorResponse{Error: "tagged nodes not (yet) supported"})
		return
	}

	rules, err := tailcfg.UnmarshalCapJSON[capRule](who.CapMap, t.capName)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error unmarshaling capability")
		ct.JSON(http.StatusForbidden, ErrorResponse{Error: "Error unmarshaling capability in tailscale ACL", Cause: err.Error()})
		return
	}

	if len(rules) == 0 {
		otelzap.L().ErrorContext(ctx, "No capability rule found for user. Assuming unauthorized.")
		ct.JSON(http.StatusForbidden, ErrorResponse{Error: "User not authorized in tailscale ACL"})
		return
	}

	if len(rules) > 1 {
		// TODO(cedi): unsure what to do when having more than one cap...
		otelzap.L().ErrorContext(ctx, "More than one capability rule found")
		ct.JSON(http.StatusBadRequest, ErrorResponse{Error: "More than one capability rule found"})
		return
	}

	now := time.Now()
	role := rules[0].Role
	period, err := time.ParseDuration(rules[0].Period)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
		ct.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Error parsing duration", Cause: err.Error()})
		return
	}
	until := now.Add(period)

	if err := t.operator.SignInUser(ctx, userName, role, until); err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error signing in user")
		ct.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Error signing in user", Cause: err.Error()})
		return
	}

	otelzap.L().InfoContext(ctx,
		"User login request was successful and is now awaiting the provisioning of the Kubernetes credentials",
		zap.String("user", userName),
		zap.String("role", role),
		zap.String("now", now.Format(time.RFC3339)),
		zap.String("period", period.String()),
		zap.String("until", until.Format(time.RFC3339)),
	)

	ct.JSON(http.StatusAccepted, UserLoginResponse{Username: userName, Role: role, Until: until.Format(time.RFC3339)})
}

type UserLoginResponse struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Until    string `json:"until"`
}

func (e ErrorResponse) AsHumane() humane.Error {
	return humane.Wrap(fmt.Errorf("%s", e.Cause), e.Error)
}

type ErrorResponse struct {
	Error string `json:"error"`
	Cause string `json:"internal_error,omitempty"`
}
