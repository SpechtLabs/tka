#!/bin/bash

# Demo script for the new Gossip TUI
# This script creates a tmux session with 5 panes running gossip nodes

echo "Starting Gossip TUI Demo with tmux..."
echo "This will create a tmux session with 5 gossip nodes"
echo ""

# Check if tmux is available
if ! command -v tmux &>/dev/null; then
  echo "Error: tmux is not installed. Please install tmux first."
  exit 1
fi

# Kill any existing session with the same name
SESSION_NAME="gossip-demo"
tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true

# Kill any existing processes on the ports we need
echo "Cleaning up any existing processes on ports 8080-8083, 8089..."
for port in 8080 8081 8082 8083 8089; do
  echo "Cleaning port $port..."
  lsof -ti:$port | xargs kill -9 2>/dev/null || true
done
sleep 2

# Double-check that ports are free
echo "Verifying ports are free..."
for port in 8080 8081 8082 8083 8089; do
  if lsof -i:$port >/dev/null 2>&1; then
    echo "Port $port is still in use!"
    lsof -i:$port
  else
    echo "Port $port is free"
  fi
done

# Create a new tmux session with a window
tmux new-session -d -s "$SESSION_NAME" -n "gossip"

echo "Setting up tmux panes..."

# Create 5 panes - keep it simple with sequential splits
echo "Creating 5 panes..."

# Create 4 more panes (starting with 1, we'll have 5 total)
tmux split-window -h -t "$SESSION_NAME:gossip"
tmux split-window -v -t "$SESSION_NAME:gossip"
tmux split-window -h -t "$SESSION_NAME:gossip"
tmux split-window -v -t "$SESSION_NAME:gossip"

# Arrange in a tiled layout
tmux select-layout -t "$SESSION_NAME:gossip" tiled

# Debug: Show available panes
echo "Available panes (should be 5):"
tmux list-panes -t "$SESSION_NAME:gossip" -F "#{pane_index}: #{pane_width}x#{pane_height} [#{pane_id}]"

echo "Starting gossip nodes..."

# Get the absolute path to the project root
PROJECT_ROOT=$(cd "$(dirname "$0")/.." && pwd)

# Start each node in its respective pane
echo "Starting server node on port 8080 (pane 1)..."
echo "Project root: $PROJECT_ROOT"

# Pane 1: Server
tmux select-pane -t "$SESSION_NAME:gossip.1"

tmux send-keys -t "$SESSION_NAME:gossip.1" "set -x OTEL_ENVIRONMENT \"dev\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.1" "set -x OTEL_EXPORTER_OTLP_ENDPOINT \"localhost:4317\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.1" "set -x OTEL_EXPORTER_OTLP_INSECURE \"true\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.1" "set -x OTEL_LOG_LEVEL \"debug\"" Enter
sleep 0.5
tmux send-keys -t "$SESSION_NAME:gossip.1" "cd '$PROJECT_ROOT'" Enter
sleep 0.5
tmux send-keys -t "$SESSION_NAME:gossip.1" "go run ./cmd/gossip serve 'server' --listen-port 8080 --gossip-interval 5s --gossip-factor 2" Enter

echo "Waiting for server to start up..."
sleep 5

# Verify server is listening
echo "Verifying server is listening on port 8080..."
if lsof -i:8080 >/dev/null 2>&1; then
  echo "✓ Server is listening on port 8080"
else
  echo "✗ Server is NOT listening on port 8080"
  echo "Check pane 1 for errors."
fi

# Pane 2: Client 1
echo "Starting client 1 on port 8081 (pane 2)..."
tmux select-pane -t "$SESSION_NAME:gossip.2"
tmux send-keys -t "$SESSION_NAME:gossip.2" "set -x OTEL_ENVIRONMENT \"dev\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.2" "set -x OTEL_EXPORTER_OTLP_ENDPOINT \"localhost:4317\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.2" "set -x OTEL_EXPORTER_OTLP_INSECURE \"true\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.2" "set -x OTEL_LOG_LEVEL \"debug\"" Enter
sleep 0.5
tmux send-keys -t "$SESSION_NAME:gossip.2" "cd '$PROJECT_ROOT'" Enter
sleep 0.5
tmux send-keys -t "$SESSION_NAME:gossip.2" "go run ./cmd/gossip client 'client-1' --listen-port 8081 --server-addr localhost:8080 --gossip-interval 5s --gossip-factor 2" Enter
sleep 2

# Pane 3: Client 2
echo "Starting client 2 on port 8082 (pane 3)..."
tmux select-pane -t "$SESSION_NAME:gossip.3"
tmux send-keys -t "$SESSION_NAME:gossip.3" "set -x OTEL_ENVIRONMENT \"dev\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.3" "set -x OTEL_EXPORTER_OTLP_ENDPOINT \"localhost:4317\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.3" "set -x OTEL_EXPORTER_OTLP_INSECURE \"true\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.3" "set -x OTEL_LOG_LEVEL \"debug\"" Enter
sleep 0.5
tmux send-keys -t "$SESSION_NAME:gossip.3" "cd '$PROJECT_ROOT'" Enter
sleep 0.5
tmux send-keys -t "$SESSION_NAME:gossip.3" "go run ./cmd/gossip client 'client-2' --listen-port 8082 --server-addr localhost:8080 --gossip-interval 5s --gossip-factor 2" Enter
sleep 2

# Pane 4: Client 3
echo "Starting client 3 on port 8083 (pane 4)..."
tmux select-pane -t "$SESSION_NAME:gossip.4"
tmux send-keys -t "$SESSION_NAME:gossip.4" "set -x OTEL_ENVIRONMENT \"dev\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.4" "set -x OTEL_EXPORTER_OTLP_ENDPOINT \"localhost:4317\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.4" "set -x OTEL_EXPORTER_OTLP_INSECURE \"true\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.4" "set -x OTEL_LOG_LEVEL \"debug\"" Enter
sleep 0.5
tmux send-keys -t "$SESSION_NAME:gossip.4" "cd '$PROJECT_ROOT'" Enter
sleep 0.5
tmux send-keys -t "$SESSION_NAME:gossip.4" "go run ./cmd/gossip client 'client-3' --listen-port 8083 --server-addr localhost:8080 --gossip-interval 5s --gossip-factor 2" Enter
sleep 2

# Pane 5: Change Introducer
echo "Starting change introducer on port 8089 (pane 5)..."
tmux select-pane -t "$SESSION_NAME:gossip.5"
tmux send-keys -t "$SESSION_NAME:gossip.5" "set -x OTEL_ENVIRONMENT \"dev\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.5" "set -x OTEL_EXPORTER_OTLP_ENDPOINT \"localhost:4317\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.5" "set -x OTEL_EXPORTER_OTLP_INSECURE \"true\"" Enter
tmux send-keys -t "$SESSION_NAME:gossip.5" "set -x OTEL_LOG_LEVEL \"debug\"" Enter
sleep 0.5
tmux send-keys -t "$SESSION_NAME:gossip.5" "cd '$PROJECT_ROOT'" Enter
sleep 0.5
tmux send-keys -t "$SESSION_NAME:gossip.5" "go run ./cmd/gossip change-introducer --listen-port 8089 --server-addr localhost:8080 --gossip-interval 5s --gossip-factor 2 --status-change-interval 15s" Enter
sleep 2

echo ""
echo "✓ All nodes started in tmux session '$SESSION_NAME'!"
echo ""
echo "Nodes:"
echo "  • Pane 1: Server (8080)"
echo "  • Pane 2: Client 1 (8081)"
echo "  • Pane 3: Client 2 (8082)"
echo "  • Pane 4: Client 3 (8083)"
echo "  • Pane 5: Change Introducer (8089)"
echo ""
echo "🎯 The change introducer will update its state every 30 seconds"
echo "👀 Watch how the state changes propagate through the cluster!"
echo "🔴 Red = Recent state change (last 3 seconds)"
echo "🟢 Green = Local node"
echo ""
echo "Controls:"
echo "  • To detach: Ctrl+B then D"
echo "  • To quit: Press 'q' in any pane"
echo "  • To kill session: tmux kill-session -t $SESSION_NAME"
echo ""

# Attach to the session
tmux attach-session -t "$SESSION_NAME"
