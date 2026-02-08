#!/usr/bin/env bash
set -Eeuo pipefail

# --- Configuration ---
LOGFILE="${LOGFILE:-output.log}"
# No PIDFILE needed anymore
PORT=8080
SEARCH_PATTERN="make.*hack-me"

# --- Helper Functions ---

# Force matching the remote source of truth
update_and_rebuild() {
  echo ">>> Forcing local to match remote main..."
  git fetch origin main
  git reset --hard origin/main
  git clean -fd

  echo ">>> Cleaning dependencies..."
  go mod tidy
}

# Checks if the 'make' supervisor is currently active
is_running() {
  pgrep -f "$SEARCH_PATTERN" > /dev/null
}

# Kills the entire process group (make + doppler + go + main)
stop_process() {
  local target_pid
  target_pid=$(pgrep -f "$SEARCH_PATTERN") || true

  if [ -n "$target_pid" ]; then
    echo ">>> Stopping process group for PID $target_pid..."
    # Using the negative PID to kill the entire group
    sudo kill -9 -"$target_pid" 2>/dev/null || true
    echo ">>> Server stopped."
  else
    echo ">>> No running server found matching '$SEARCH_PATTERN'."
  fi
}

# The actual loop that keeps the server alive
run_supervisor() {
  echo ">>> Supervisor started (PID $$)"
  
  # Note: Since we are running 'make run' here, this script's PID 
  # will effectively be the 'Root' of the chain.
  while true; do
    echo "[$(date)] Starting server..." >> "$LOGFILE"
    
    # Executing make run. If it fails or is killed, we loop.
    make run >> "$LOGFILE" 2>&1 || true
    
    echo "[$(date)] Server exited. Restarting in 2s..." >> "$LOGFILE"
    sleep 2
  done
}

# --- Main Commands ---

case "${1:-start}" in
  start)
    if is_running; then
      echo "Already running. Use 'stop' or 'restart' first."
      exit 1
    fi
    # Run supervisor in background and disown so it stays alive
    run_supervisor &
    echo "Server started in background. Logs: $LOGFILE"
    ;;

  stop)
    stop_process
    ;;

  status)
    if is_running; then
      echo "Status: RUNNING"
      # Show the tree so you can see the active PIDs
      pgrep -f "$SEARCH_PATTERN" | xargs pstree -p -s
    else
      echo "Status: STOPPED"
      exit 1
    fi
    ;;

  restart|all)
    stop_process
    update_and_rebuild
    $0 start
    ;;

  *)
    echo "Usage: $0 {start|stop|status|restart}"
    exit 2
    ;;
esac
