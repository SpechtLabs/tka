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

	now := time.Now()
	role := rules[0].Role
	period, err := time.ParseDuration(rules[0].Period)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
		ct.JSON(http.StatusInternalServerError, NewErrorResponse("Error parsing duration", err))
		return
	}

	if period < 10*time.Minute {
		err := humane.New("`period` may not specify a duration less than 10 minutesD",
			fmt.Sprintf("Specify a period greater than 10 minutes in your tailscale ACL for user %s", userName),
		)
		otelzap.L().WithError(err).ErrorContext(ctx, "Invalid capRule")
		ct.JSON(http.StatusUnprocessableEntity, NewErrorResponse("Invalid capRule", err))
		return
	}

	// TODO(cedi): revert to real period
	until := now.Add(period)

	if err := t.operator.SignInUser(ctx, userName, role, until); err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error signing in user")
		ct.JSON(http.StatusInternalServerError, FromHumaneError(err))
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
