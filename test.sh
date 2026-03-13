#!/bin/bash

set -u

ENDPOINT=${1:-health}
ENDPOINT=${ENDPOINT#/}
COUNT=${2:-${COUNT:-100}}
BASE_URL=${BASE_URL:-http://localhost:8000}
URL="${BASE_URL}/${ENDPOINT}"

RESULTS_FILE=$(mktemp)
LATENCY_FILE=$(mktemp)

cleanup() {
  rm -f "$RESULTS_FILE" "$LATENCY_FILE"
}

trap cleanup EXIT

send_request() {
  local index=$1
  local metrics=""
  local payload=""
  local rc=0

  if [ "$ENDPOINT" = "createuser" ]; then
    payload=$(printf '{"name":"User%d","email":"user%d@test.com"}' "$index" "$index")
    metrics=$(curl -sS -o /dev/null \
      -w "%{http_code}\t%{time_total}\t%{time_connect}\t%{time_starttransfer}\t%{size_download}" \
      -X POST "$URL" \
      -H "Content-Type: application/json" \
      -d "$payload" 2>/dev/null)
    rc=$?
  else
    metrics=$(curl -sS -o /dev/null \
      -w "%{http_code}\t%{time_total}\t%{time_connect}\t%{time_starttransfer}\t%{size_download}" \
      "$URL" 2>/dev/null)
    rc=$?
  fi

  if [ -z "$metrics" ]; then
    metrics=$'000\t0\t0\t0\t0'
  fi

  printf "%s\t%s\t%s\n" "$index" "$rc" "$metrics" >> "$RESULTS_FILE"
}

format_ms() {
  awk -v seconds="$1" 'BEGIN { printf "%.2f ms", seconds * 1000 }'
}

format_avg_bytes() {
  awk -v bytes="$1" 'BEGIN { printf "%.2f bytes", bytes }'
}

start_ms=$(date +%s%3N)

echo "Sending $COUNT requests to $URL ..."

for i in $(seq 1 "$COUNT"); do
  send_request "$i"
done

end_ms=$(date +%s%3N)
wall_seconds=$(awk -v start="$start_ms" -v end="$end_ms" 'BEGIN { printf "%.6f", (end - start) / 1000 }')

summary=$(
  awk -F '\t' '
    BEGIN {
      min = -1
      max = 0
      slowest_request = 0
    }
    {
      total++
      request_id = $1 + 0
      curl_exit = $2 + 0
      http_code = $3
      total_time = $4 + 0
      connect_time = $5 + 0
      ttfb_time = $6 + 0
      body_bytes = $7 + 0

      status_codes[http_code]++
      total_latency += total_time
      total_connect += connect_time
      total_ttfb += ttfb_time
      total_bytes += body_bytes

      if (min < 0 || total_time < min) {
        min = total_time
      }
      if (total_time > max) {
        max = total_time
        slowest_request = request_id
      }

      if (curl_exit == 0 && http_code ~ /^2[0-9][0-9]$/) {
        success++
      } else if (curl_exit != 0) {
        curl_errors++
      } else {
        http_errors++
      }
    }
    END {
      avg_latency = total ? total_latency / total : 0
      avg_connect = total ? total_connect / total : 0
      avg_ttfb = total ? total_ttfb / total : 0
      avg_bytes = total ? total_bytes / total : 0
      printf "%d\t%d\t%d\t%d\t%.6f\t%.6f\t%.6f\t%.6f\t%.6f\t%d",
        total,
        success,
        http_errors,
        curl_errors,
        min,
        avg_latency,
        avg_connect,
        avg_ttfb,
        avg_bytes,
        slowest_request
    }
  ' "$RESULTS_FILE"
)

IFS=$'\t' read -r total_requests successful_requests http_errors curl_errors min_latency avg_latency avg_connect avg_ttfb avg_bytes slowest_request <<< "$summary"

awk -F '\t' '{ print $4 }' "$RESULTS_FILE" | sort -n > "$LATENCY_FILE"

percentiles=$(
  awk '
    function percentile_index(p, n) {
      idx = int((p * n + 99) / 100)
      if (idx < 1) {
        idx = 1
      }
      return idx
    }
    {
      values[NR] = $1
    }
    END {
      if (NR == 0) {
        printf "0\t0\t0\t0\t0"
        exit
      }
      printf "%s\t%s\t%s\t%s\t%s",
        values[percentile_index(50, NR)],
        values[percentile_index(90, NR)],
        values[percentile_index(95, NR)],
        values[percentile_index(99, NR)],
        values[NR]
    }
  ' "$LATENCY_FILE"
)

IFS=$'\t' read -r p50_latency p90_latency p95_latency p99_latency max_latency <<< "$percentiles"

success_rate=$(awk -v success="$successful_requests" -v total="$total_requests" 'BEGIN {
  if (total == 0) {
    printf "0.00%%"
  } else {
    printf "%.2f%%", (success / total) * 100
  }
}')

throughput=$(awk -v total="$total_requests" -v seconds="$wall_seconds" 'BEGIN {
  if (seconds == 0) {
    printf "0.00 req/s"
  } else {
    printf "%.2f req/s", total / seconds
  }
}')

echo
echo "Run Summary"
echo "-----------"
printf "%-20s %s\n" "Endpoint" "/$ENDPOINT"
printf "%-20s %s\n" "Target" "$BASE_URL"
printf "%-20s %s\n" "Requests sent" "$total_requests"
printf "%-20s %s\n" "Successful" "$successful_requests"
printf "%-20s %s\n" "HTTP errors" "$http_errors"
printf "%-20s %s\n" "cURL errors" "$curl_errors"
printf "%-20s %s\n" "Success rate" "$success_rate"
printf "%-20s %.2f s\n" "Wall time" "$wall_seconds"
printf "%-20s %s\n" "Throughput" "$throughput"

echo
echo "Status Codes"
echo "------------"
awk -F '\t' '{ counts[$3]++ } END { for (code in counts) printf "%s\t%d\n", code, counts[code] }' "$RESULTS_FILE" | sort -n | awk -F '\t' '{ printf "%-20s %s\n", $1, $2 }'

echo
echo "Latency"
echo "-------"
printf "%-20s %s\n" "Min" "$(format_ms "$min_latency")"
printf "%-20s %s\n" "P50" "$(format_ms "$p50_latency")"
printf "%-20s %s\n" "P90" "$(format_ms "$p90_latency")"
printf "%-20s %s\n" "P95" "$(format_ms "$p95_latency")"
printf "%-20s %s\n" "P99" "$(format_ms "$p99_latency")"
printf "%-20s %s\n" "Max" "$(format_ms "$max_latency")"
printf "%-20s %s\n" "Average" "$(format_ms "$avg_latency")"

echo
echo "Timing Breakdown"
echo "----------------"
printf "%-20s %s\n" "Avg connect" "$(format_ms "$avg_connect")"
printf "%-20s %s\n" "Avg TTFB" "$(format_ms "$avg_ttfb")"
printf "%-20s %s\n" "Avg body size" "$(format_avg_bytes "$avg_bytes")"
printf "%-20s %s\n" "Slowest request" "#$slowest_request"

echo
echo "Done."
