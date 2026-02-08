#!/usr/bin/env bash
set -Eeuo pipefail

# --- Configuration ---
LOGFILE="${LOGFILE:-output.log}"
PORT=8080

# --- Helper Functions ---

# 1. STOP: Kills supervisor first, then port
stop_process() {
  echo ">>> [1/3] Killing Supervisor loop (run.sh)..."
  # Finds the background 'run-internal' loop
  local supervisor_pid
  supervisor_pid=$(pgrep -f "run.sh run-internal") || true
  
  if [ -n "$supervisor_pid" ]; then
    sudo kill -9 "$supervisor_pid" 2>/dev/null || true
    echo "    Supervisor (PID $supervisor_pid) terminated."
  else
    echo "    No supervisor found."
  fi

  echo ">>> [2/3] Wiping port $PORT..."
  # Clears the workers (make, doppler, go, main)
  sudo fuser -k -9 "$PORT/tcp" 2>/dev/null || true

  echo ">>> [3/3] Final verification..."
  if sudo lsof -i :$PORT > /dev/null; then
    echo "    ERROR: Port $PORT still occupied. Forcing one last wipe..."
    sudo fuser -k -9 "$PORT/tcp"
  else
    echo "    SUCCESS: Port $PORT is clear."
  fi
}

# 2. UPDATE: Force match remote
update_source() {
  echo ">>> Fetching and Overriding with remote origin/main..."
  git fetch origin main
  git reset --hard origin/main
  git clean -fd
  echo ">>> Updating Go dependencies..."
  go mod tidy
}

# 3. SUPERVISOR: The loop
run_supervisor() {
  while true; do
    echo "[$(date)] Starting server..." >> "$LOGFILE"
    make run >> "$LOGFILE" 2>&1 || true
    echo "[$(date)] Server exited. Respawning in 2s..." >> "$LOGFILE"
    sleep 2
  done
}

# --- Main Commands ---

case "${1:-status}" in
  start)
    if sudo lsof -i :$PORT > /dev/null; then
      echo "Port $PORT is already in use. Run 'stop' first."
      exit 1
    fi
    # Start loop in background
    nohup "$0" run-internal >> "$LOGFILE" 2>&1 &
    echo "Server started. Logs: $LOGFILE"
    ;;

  run-internal)
    run_supervisor
    ;;

  stop)
    stop_process
    ;;

  status)
    if sudo lsof -i :$PORT > /dev/null; then
      echo "Status: RUNNING"
      sudo lsof -t -i:$PORT | xargs pstree -p -s
    else
      echo "Status: STOPPED"
    fi
    ;;

  update)
    stop_process
    update_source
    echo "Update complete. Run './run.sh start' to resume."
    ;;

  *)
    echo "Usage: $0 {start|stop|status|update}"
    exit 2
    ;;
esac
