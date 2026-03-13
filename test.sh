#!/bin/bash

ENDPOINT=${1:-health}
COUNT=100
BASE_URL="http://localhost:8000"

echo "Sending $COUNT requests to /$ENDPOINT..."

for i in $(seq 1 $COUNT); do
  if [ "$ENDPOINT" = "createuser" ]; then
    curl -s -o /dev/null -w "%{http_code}\n" \
      -X POST "$BASE_URL/createuser" \
      -H "Content-Type: application/json" \
      -d "{\"name\":\"User$i\",\"email\":\"user$i@test.com\"}"
  else
    curl -s -o /dev/null -w "%{http_code}\n" "$BASE_URL/health"
  fi
done | sort | uniq -c | awk '{print $2 " -> " $1 " responses"}'

echo "Done."
