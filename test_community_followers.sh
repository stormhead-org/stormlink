#!/bin/bash

echo "Testing communityFollowers query with filters..."

COMMUNITY_ID="1"

echo "1. Testing communityFollowers with ALL filter (default)..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"query { communityFollowers(communityID: \\\"$COMMUNITY_ID\\\") { id name slug email isVerified } }\"}" \
  -s | jq '.'

echo -e "\n2. Testing communityFollowers with ACTIVE filter..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"query { communityFollowers(communityID: \\\"$COMMUNITY_ID\\\", filter: ACTIVE) { id name slug email isVerified } }\"}" \
  -s | jq '.'

echo -e "\n3. Testing communityFollowers with BANNED filter..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"query { communityFollowers(communityID: \\\"$COMMUNITY_ID\\\", filter: BANNED) { id name slug email isVerified } }\"}" \
  -s | jq '.'

echo -e "\n4. Testing communityFollowers with MUTED filter..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"query { communityFollowers(communityID: \\\"$COMMUNITY_ID\\\", filter: MUTED) { id name slug email isVerified } }\"}" \
  -s | jq '.'

echo -e "\n5. Testing communityFollowers with pagination (limit: 5, offset: 0)..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"query { communityFollowers(communityID: \\\"$COMMUNITY_ID\\\", filter: ALL, limit: 5, offset: 0) { id name slug email isVerified } }\"}" \
  -s | jq '.'
