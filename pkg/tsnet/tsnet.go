// Package tsnet provides a Tailscale server implementation.
// This is a wrapper around the tsnet.Server type to provide better testability
package tsnet

import (
	"context"
	"net"

	"github.com/sierrasoftworks/humane-errors-go"
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	ts "tailscale.com/tsnet"
)

type BackendState string

const (
	BackendStateNoState          BackendState = "NoState"
	BackendStateNeedsLogin       BackendState = "NeedsLogin"
	BackendStateNeedsMachineAuth BackendState = "NeedsMachineAuth"
	BackendStateStopped          BackendState = "Stopped"
	BackendStateStarting         BackendState = "Starting"
	BackendStateRunning          BackendState = "Running"
)

type Server struct {
	s  *ts.Server       // embedded tsnet.Server
	st *ipnstate.Status // Connection status
}

func NewServer(hostname string) *Server {
	return &Server{
		s:  &ts.Server{Hostname: hostname},
		st: &ipnstate.Status{BackendState: string(BackendStateNoState)},
	}
}

func (a *Server) Hostname() string {
	return a.s.Hostname
}

func (a *Server) Up(ctx context.Context) (*ipnstate.Status, error) {
	st, err := a.s.Up(ctx)
	if err != nil {
		return nil, humane.Wrap(err, "failed to connect to tailscale", "check (debug) logs for more details")
	}
	a.st = st
	return st, nil
}

func (a *Server) Listen(network, addr string) (net.Listener, error) {
	return a.s.Listen(network, addr)
}

func (a *Server) ListenTLS(network, addr string) (net.Listener, error) {
	return a.s.ListenTLS(network, addr)
}

func (a *Server) ListenFunnel(network, addr string) (net.Listener, error) {
	return a.s.ListenFunnel(network, addr)
}

func (a *Server) LocalWhoIs() (WhoIsResolver, error) {
	lc, err := a.s.LocalClient()
	if err != nil {
		return nil, err
	}
	return &localWhoIsResolver{lc: lc}, nil
}

func (a *Server) SetDir(dir string)                 { a.s.Dir = dir }
func (a *Server) SetLogf(logf func(string, ...any)) { a.s.Logf = logf }

func (a *Server) GetPeerState() *ipnstate.PeerStatus {
	return a.st.Self
}

// IsConnected reports whether the server is connected to the Tailscale network.
// Returns true only when the backend state is "Running".
func (a *Server) IsConnected() bool {
	if a.st == nil {
		return false
	}

	return a.st.BackendState == "Running"
}

// BackendState returns the current Tailscale backend state.
// Possible values: "NoState", "NeedsLogin", "NeedsMachineAuth", "Stopped",
// "Starting", "Running".
func (a *Server) BackendState() BackendState {
	if a.st == nil {
		return BackendStateNoState
	}

	return BackendState(a.st.BackendState)
}

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
