#!/usr/bin/env bash
set -Eeuo pipefail

LOGFILE="${LOGFILE:-output.log}"
PIDFILE="${PIDFILE:-run.pid}"

# --- Added Update Logic ---
update_and_rebuild() {
  echo ">>> Pulling latest changes from git..."
  git pull origin main

  echo ">>> Updating Go modules..."
  go mod tidy

  echo ">>> Update complete."
}

ensure_not_running() {
  if [[ -f "$PIDFILE" ]]; then
    local pid
    pid="$(cat "$PIDFILE" 2>/dev/null || true)"
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      echo "Already running (PID $pid). Use --stop first." >&2
      exit 1
    else
      rm -f "$PIDFILE"
    fi
  fi
}

start() {
  ensure_not_running

  : > "$LOGFILE"

  nohup bash -c '
    set -Eeuo pipefail
    child_pid=""

    cleanup() {
      if [[ -n "${child_pid:-}" ]] && kill -0 "$child_pid" 2>/dev/null; then
        kill "$child_pid" 2>/dev/null || true
        for _ in {1..20}; do
          kill -0 "$child_pid" 2>/dev/null || break
          sleep 0.1
        done
        kill -9 "$child_pid" 2>/dev/null || true
      fi
    }

    trap "cleanup; exit 0" INT TERM EXIT
    echo "$$" > "'"$LOGFILE"'"

    while true; do
      echo "Starting Go server at $(date)" >> "'"$LOGFILE"'"
      make run >> "'"$LOGFILE"'" 2>&1 &
      child_pid=$!
      echo "CHILD_PID=$child_pid" >> "'"$LOGFILE"'"
      wait "$child_pid" || true
      echo "Server exited at $(date). Restarting in 2s..." >> "'"$LOGFILE"'"
      sleep 2
    done
  ' >/dev/null 2>&1 &

  echo $! > "$PIDFILE"
  echo "Started. Supervisor PID $(cat "$PIDFILE"). Logs: $LOGFILE"
}

stop() {
  if [[ ! -f "$PIDFILE" ]]; then
    echo "Not running (no $PIDFILE)."
    return 0
  fi

  local pid
  pid="$(cat "$PIDFILE" 2>/dev/null || true)"
  if [[ -z "${pid:-}" ]]; then
    echo "PID file empty. Removing."
    rm -f "$PIDFILE"
    return 0
  fi

  if ! kill -0 "$pid" 2>/dev/null; then
    echo "Process $pid not alive. Cleaning up."
    rm -f "$PIDFILE"
    return 0
  fi

  # Attempt to kill whatever is on port 8080 (optional but helpful)
  sudo lsof -t -i:8080 2>/dev/null | sort -u | xargs -r -n1 sudo kill || true
  kill "$pid" 2>/dev/null || true

  for _ in {1..50}; do
    kill -0 "$pid" 2>/dev/null || { rm -f "$PIDFILE"; echo "Stopped."; return 0; }
    sleep 0.1
  done

  echo "Supervisor did not exit in time; forcing."
  kill -9 "$pid" 2>/dev/null || true
  rm -f "$PIDFILE"
  echo "Stopped (forced)."
}

status() {
  if [[ -f "$PIDFILE" ]]; then
    local pid
    pid="$(cat "$PIDFILE" 2>/dev/null || true)"
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      echo "Running (PID $pid). Log: $LOGFILE"
      exit 0
    fi
  fi
  echo "Not running."
  exit 1
}

# --- Main Logic ---
case "${1:-start}" in
  start|--start) 
    start 
    ;;
  stop|--stop)   
    stop 
    ;;
  status|--status) 
    status 
    ;;
  all|--all)
    echo ">>> Performing full update and restart..."
    # 1. Stop current process
    stop
    # 2. Update code and dependencies
    update_and_rebuild
    # 3. Start new process
    start
    ;;
  *)
    echo "Usage: $0 [start|stop|status|all]"
    exit 2
    ;;
esac
