// Package tailscale provides adapters for integrating with Tailscale's tsnet package.
// This file contains adapter implementations that bridge our interfaces with tsnet types.
package tailscale

import (
	"context"
	"net"

	"github.com/sierrasoftworks/humane-errors-go"
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

// tsnetAdapter implements TSNet by delegating to *tsnet.Server.
// This adapter allows us to use tsnet.Server through our TSNet interface.
type tsnetAdapter struct{ s *tsnet.Server }

func (a *tsnetAdapter) Up(ctx context.Context) (*ipnstate.Status, error) {
	return a.s.Up(ctx)
}

func (a *tsnetAdapter) Listen(network, addr string) (net.Listener, error) {
	return a.s.Listen(network, addr)
}

func (a *tsnetAdapter) ListenTLS(network, addr string) (net.Listener, error) {
	return a.s.ListenTLS(network, addr)
}

func (a *tsnetAdapter) ListenFunnel(network, addr string) (net.Listener, error) {
	return a.s.ListenFunnel(network, addr)
}

func (a *tsnetAdapter) LocalWhoIs() (WhoIsResolver, error) {
	lc, err := a.s.LocalClient()
	if err != nil {
		return nil, err
	}
	return &localWhoIsResolver{lc: lc}, nil
}

func (a *tsnetAdapter) SetDir(dir string)                 { a.s.Dir = dir }
func (a *tsnetAdapter) SetLogf(logf func(string, ...any)) { a.s.Logf = logf }

// localWhoIsResolver adapts *local.Client to our WhoIsResolver interface.
// This adapter bridges Tailscale's local client with our interface.
type localWhoIsResolver struct{ lc *local.Client }

func (l *localWhoIsResolver) WhoIs(ctx context.Context, remoteAddr string) (*WhoIsInfo, humane.Error) {
	who, err := l.lc.WhoIs(ctx, remoteAddr)
	if err != nil {
		return nil, humane.Wrap(err, "failed to get WhoIs", "check (debug) logs for more details")
	}
	return &WhoIsInfo{
		LoginName: who.UserProfile.LoginName,
		CapMap:    who.CapMap,
		Tags:      who.Node.Tags,
	}, nil
}
