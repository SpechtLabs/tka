package tailscale

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
		ct.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	who, err := t.lc.WhoIs(ctx, req.RemoteAddr)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting WhoIs")
		ct.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting WhoIs", "internal_error": err.Error()})
		return
	}

	// not sure if this is the right thing to do...
	userName, _, _ := strings.Cut(who.UserProfile.LoginName, "@")
	n := who.Node.View()
	if n.IsTagged() {
		otelzap.L().ErrorContext(ctx, "tagged nodes not (yet) supported")
		ct.JSON(http.StatusBadRequest, gin.H{"error": "tagged nodes not (yet) supported"})
		return
	}

	rules, err := tailcfg.UnmarshalCapJSON[capRule](who.CapMap, t.capName)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error unmarshaling capability")
		ct.JSON(http.StatusBadRequest, gin.H{"error": "Error unmarshaling capability", "internal_error": err.Error()})
		return
	}

	if len(rules) == 0 {
		otelzap.L().ErrorContext(ctx, "No capability rule found")
		ct.JSON(http.StatusBadRequest, gin.H{"error": "No capability rule found"})
		return
	}

	if len(rules) > 1 {
		// TODO(cedi): unsure what to do when having more than one cap...
		otelzap.L().ErrorContext(ctx, "More than one capability rule found")
		ct.JSON(http.StatusBadRequest, gin.H{"error": "More than one capability rule found"})
		return
	}

	now := time.Now()
	role := rules[0].Role
	period, err := time.ParseDuration(rules[0].Period)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
		ct.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing duration", "internal_error": err.Error()})
		return
	}
	until := now.Add(period)

	otelzap.L().InfoContext(ctx,
		"User is now logged in and has assigned a role",
		zap.String("user", userName),
		zap.String("role", role),
		zap.String("now", now.Format(time.RFC3339)),
		zap.String("period", period.String()),
		zap.String("until", until.Format(time.RFC3339)),
	)

	ct.JSON(http.StatusOK, gin.H{"user": userName, "role": role, "until": until.Format(time.RFC3339)})
}
