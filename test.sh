#!/bin/sh

echo "Populating score events for 6 different talents..."

# Talent 1: Alice - dribble and shoot
curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_001",
    "talent_id": "alice_001",
    "raw_metric": 90,
    "skill": "dribble",
    "ts": "2025-01-27T10:30:00Z"
  }'

curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_002",
    "talent_id": "alice_001",
    "raw_metric": 85,
    "skill": "shoot",
    "ts": "2025-01-27T10:31:00Z"
  }'

# Talent 2: Bob - only pass
curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_003",
    "talent_id": "bob_002",
    "raw_metric": 95,
    "skill": "pass",
    "ts": "2025-01-27T10:32:00Z"
  }'

# Talent 3: Charlie - dribble and pass
curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_004",
    "talent_id": "charlie_003",
    "raw_metric": 88,
    "skill": "dribble",
    "ts": "2025-01-27T10:33:00Z"
  }'

curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_005",
    "talent_id": "charlie_003",
    "raw_metric": 82,
    "skill": "pass",
    "ts": "2025-01-27T10:34:00Z"
  }'

# Talent 4: Diana - shoot and pass
curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_006",
    "talent_id": "diana_004",
    "raw_metric": 92,
    "skill": "shoot",
    "ts": "2025-01-27T10:35:00Z"
  }'

curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_007",
    "talent_id": "diana_004",
    "raw_metric": 87,
    "skill": "pass",
    "ts": "2025-01-27T10:36:00Z"
  }'

# Talent 5: Eve - only dribble
curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_008",
    "talent_id": "eve_005",
    "raw_metric": 96,
    "skill": "dribble",
    "ts": "2025-01-27T10:37:00Z"
  }'

# Talent 6: Frank - all three skills
curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_009",
    "talent_id": "frank_006",
    "raw_metric": 80,
    "skill": "dribble",
    "ts": "2025-01-27T10:38:00Z"
  }'

curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_010",
    "talent_id": "frank_006",
    "raw_metric": 75,
    "skill": "shoot",
    "ts": "2025-01-27T10:39:00Z"
  }'

curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "event_011",
    "talent_id": "frank_006",
    "raw_metric": 70,
    "skill": "pass",
    "ts": "2025-01-27T10:40:00Z"
  }'

echo ""
echo "All events sent! Now let's check the leaderboard..."

# Wait a moment for processing
sleep 2


printf "\n\nGET /leaderboard\n"
# Get the leaderboard
if command -v jq &> /dev/null; then
  curl -s -X GET http://localhost:8080/leaderboard?limit=10 | jq .
else
  curl -s -X GET http://localhost:8080/leaderboard?limit=10
fi

sleep 1

printf "\n\nGET /rank/diana_004\n"

# get someone's rank
if command -v jq &> /dev/null; then
  curl -s -X GET http://localhost:8080/rank/diana_004 | jq .
else
  curl -s -X GET http://localhost:8080/rank/diana_004
fi


# get metrics
printf "\n\nGET /metrics\n"
curl -s -X GET http://localhost:9090/metrics