#!/usr/bin/env bash
# log_viewer.sh – tiny, repo‑relative universal-log tailer
# Usage: log_viewer.sh [-n LINES] [-d YYYY-MM-DD] [-f] [-h]
#   -n  lines   (default 20)
#   -d  date    (override auto‑latest)
#   -f  follow  (tail -f)
#   -h  help
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOGDIR="$(realpath "$DIR/../../logs")"
LINES=20; FOLLOW=""; FILE=""
while getopts "n:d:fh" o; do case $o in
  n) LINES=$OPTARG;;
  d) FILE="$LOGDIR/universal_logs-$OPTARG.log";;
  f) FOLLOW=-f;;
  h) echo "usage: $(basename $0) [-n lines] [-d YYYY-MM-DD] [-f]"; exit 0;;
  *) exit 1;;
esac; done
shift $((OPTIND-1))
# auto‑pick newest log if no -d
[[ -z $FILE ]] && FILE=$(ls "$LOGDIR"/universal_logs-*.log | sort | tail -n1)
exec tail $FOLLOW -n "$LINES" "$FILE"