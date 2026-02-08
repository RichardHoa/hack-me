#!/usr/bin/env bash
set -Eeuo pipefail

# --- Configuration ---
LOGFILE="${LOGFILE:-output.log}"
PIDFILE="${PIDFILE:-run.pid}"
PORT=8080

# --- Helper Functions ---

# Pulls latest code and ensures it actually compiles before we try to restart
update_and_rebuild() {
  echo ">>> Pulling latest changes..."
  git pull origin main

  echo ">>> Cleaning dependencies..."
  go mod tidy

  echo ">>> Verifying build..."
  # Better to check if it builds before killing the old process
  if ! go build ./...; then
    echo "ERROR: Build failed. Aborting update."
    exit 1
  fi
}

# Finds and kills the supervisor and any processes it started
stop_process() {
  if [[ -f "$PIDFILE" ]]; then
    local pid
    pid=$(cat "$PIDFILE")

    echo ">>> Stopping Supervisor (PID $pid) and its children..."
    
    # 1. Kill the parent supervisor process
    # 2. Kill all processes belonging to that parent's process group
    pkill -P "$pid" || true
    kill "$pid" 2>/dev/null || true
    
    # Optional: Cleanup specific port if still bound
    lsof -t -i:"$PORT" | xargs kill -9 2>/dev/null || true
    
    rm -f "$PIDFILE"
    echo ">>> Stopped."
  else
    echo ">>> Not running (no $PIDFILE found)."
  fi
}

# The actual loop that keeps the server alive
run_supervisor() {
  # Write the PID of THIS subshell to the pidfile
  echo "$$" > "$PIDFILE"

  # Ensure the logfile exists and is empty
  : > "$LOGFILE"

  echo ">>> Supervisor started (PID $$)"
  
  while true; do
    echo "[$(date)] Starting server..." >> "$LOGFILE"
    
    # Run the server. Using 'exec' here isn't ideal because we want to loop, 
    # so we run it in the foreground and wait.
    make run >> "$LOGFILE" 2>&1 || true
    
    echo "[$(date)] Server exited. Restarting in 2s..." >> "$LOGFILE"
    sleep 2
  done
}

# --- Main Commands ---

case "${1:-start}" in
  start)
    if [[ -f "$PIDFILE" ]] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
      echo "Already running (PID $(cat "$PIDFILE"))."
      exit 1
    fi
    # Launch the supervisor function in the background
    run_supervisor &
    echo "Server started in background. Logs: $LOGFILE"
    ;;

  stop)
    stop_process
    ;;

  status)
    if [[ -f "$PIDFILE" ]] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
      echo "Status: RUNNING (PID $(cat "$PIDFILE"))"
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
