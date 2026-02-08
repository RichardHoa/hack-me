#!/usr/bin/env bash
set -Eeuo pipefail

# --- Configuration ---
LOGFILE="${LOGFILE:-output.log}"
# We identify the root by the exact command you use to start it
SEARCH_PATTERN="^make run"
# We'll also keep a secondary check for the main binary path just in case
BINARY_PATH="/home/ubuntu/codes/hack-me/main"

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

# Checks if the 'make run' command is active
is_running() {
  pgrep -f "$SEARCH_PATTERN" > /dev/null
}

stop_process() {
  sudo lsof -t -i:8080 | xargs pstree -p -s
}

# The loop that restarts the server
run_supervisor() {
  echo ">>> Supervisor started (PID $$)"
  
  while true; do
    echo "[$(date)] Starting server via make run..." >> "$LOGFILE"
    
    # We use 'exec' logic or just direct call. 
    # Since it's in a loop, we just call it.
    make run >> "$LOGFILE" 2>&1 || true
    
    echo "[$(date)] Server exited. Restarting in 2s..." >> "$LOGFILE"
    sleep 2
  done
}

# --- Main Commands ---
case "${1:-status}" in
  start)
    if is_running; then
      echo "Already running. Use 'stop' or 'restart'."
      exit 1
    fi
    # Run in background and detach from the current terminal
    nohup "$0" run-internal >> "$LOGFILE" 2>&1 &
    echo "Server started in background. Logs: $LOGFILE"
    ;;

  run-internal)
    # Hidden command used by 'start' to run the loop
    run_supervisor
    ;;

  stop)
    stop_process
    ;;

  status)
    if is_running; then
      local pid
      pid=$(pgrep -f "$SEARCH_PATTERN")
      echo "Status: RUNNING (Root PID: $pid)"
      pstree -p -s "$pid"
    else
      echo "Status: STOPPED"
    fi
    ;;

  restart|all)
    $0 stop
    update_and_rebuild
    $0 start
    ;;

  *)
    echo "Usage: $0 {start|stop|status|restart}"
    exit 2
    ;;
esac
