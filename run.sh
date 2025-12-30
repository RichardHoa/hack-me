#!/usr/bin/env bash
set -Eeuo pipefail

LOGFILE="${LOGFILE:-output.log}"
PIDFILE="${PIDFILE:-run.pid}"

update_and_rebuild() {
  echo ">>> Pulling latest changes from git..."
  git pull origin main

  echo ">>> Updating Go modules..."
  go mod download
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

  # Clear previous log and ensure first line is PID of the supervisor (written by the supervisor itself)
  : > "$LOGFILE"

  # Launch the supervisor in the background, fully detached
  nohup bash -c '
    set -Eeuo pipefail

    child_pid=""

    cleanup() {
      if [[ -n "${child_pid:-}" ]] && kill -0 "$child_pid" 2>/dev/null; then
        kill "$child_pid" 2>/dev/null || true
        # Give it a moment to exit gracefully
        for _ in {1..20}; do
          kill -0 "$child_pid" 2>/dev/null || break
          sleep 0.1
        done
        kill -9 "$child_pid" 2>/dev/null || true
      fi
    }

    trap "cleanup; exit 0" INT TERM EXIT

    # First line of the log: the SUPERVISOR PID (just the number)
    echo "$$" > "'"$LOGFILE"'"

    # Main loop: run your server, restart on exit
    while true; do
      echo "Starting Go server at $(date)" >> "'"$LOGFILE"'"
      # Run your command in background so we can capture PID and wait on it
      make run >> "'"$LOGFILE"'" 2>&1 &
      child_pid=$!
      echo "CHILD_PID=$child_pid" >> "'"$LOGFILE"'"
      # Wait for server to exit (crash or normal stop)
      wait "$child_pid" || true
      echo "Server exited at $(date). Restarting in 2s..." >> "'"$LOGFILE"'"
      sleep 2
    done
  ' >/dev/null 2>&1 &

  # Save the SUPERVISOR PID so we can stop later
  echo $! > "$PIDFILE"
  echo "Started. Supervisor PID $(cat "$PIDFILE"). Logs: $LOGFILE"
}

stop() {
  if [[ ! -f "$PIDFILE" ]]; then
    echo "Not running (no $PIDFILE)."
    exit 0
  fi

  local pid
  pid="$(cat "$PIDFILE" 2>/dev/null || true)"
  if [[ -z "${pid:-}" ]]; then
    echo "PID file empty. Removing."
    rm -f "$PIDFILE"
    exit 0
  fi

  if ! kill -0 "$pid" 2>/dev/null; then
    echo "Process $pid not alive. Cleaning up."
    rm -f "$PIDFILE"
    exit 0
  fi

  sudo lsof -t -i:8080 2>/dev/null | sort -u | xargs -r -n1 sudo kill || true
  # Ask supervisor to exit (it will kill the child and stop the loop)
  kill "$pid" 2>/dev/null || true

  # Wait up to ~5s for clean shutdown, then force
  for _ in {1..50}; do
    kill -0 "$pid" 2>/dev/null || { rm -f "$PIDFILE"; echo "Stopped."; exit 0; }
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
    stop
    update_and_rebuild
    start
    ;;
  *)
    echo "Usage: $0 [start|stop|status|all]"
    exit 2
    ;;
esac
