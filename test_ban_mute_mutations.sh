#!/bin/bash

echo "Testing ban/mute mutations for community users..."

COMMUNITY_ID="1"
USER_ID="2"  # ID пользователя для тестирования

echo "1. Testing banUserFromCommunity..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"mutation { banUserFromCommunity(input: { userID: \\\"$USER_ID\\\", communityID: \\\"$COMMUNITY_ID\\\" }) { id user { id name } community { id title } createdAt } }\"}" \
  -s | jq '.'

echo -e "\n2. Testing muteUserInCommunity..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"mutation { muteUserInCommunity(input: { userID: \\\"$USER_ID\\\", communityID: \\\"$COMMUNITY_ID\\\" }) { id user { id name } community { id title } createdAt expiresAt } }\"}" \
  -s | jq '.'

echo -e "\n3. Testing communityFollowers with BANNED filter (should show banned user)..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"query { communityFollowers(communityID: \\\"$COMMUNITY_ID\\\", filter: BANNED) { id name slug email } }\"}" \
  -s | jq '.'

echo -e "\n4. Testing communityFollowers with MUTED filter (should show muted user)..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"query { communityFollowers(communityID: \\\"$COMMUNITY_ID\\\", filter: MUTED) { id name slug email } }\"}" \
  -s | jq '.'

echo -e "\n5. Testing communityUserBans query..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"query { communityUserBans(communityID: \\\"$COMMUNITY_ID\\\") { id user { id name } createdAt } }\"}" \
  -s | jq '.'

echo -e "\n6. Testing communityUserMutes query..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d "{\"query\":\"query { communityUserMutes(communityID: \\\"$COMMUNITY_ID\\\") { id user { id name } createdAt expiresAt } }\"}" \
  -s | jq '.'

echo -e "\n7. Testing unbanUserFromCommunity (will need banID from previous response)..."
echo "Note: You'll need to manually extract banID from the communityUserBans response above"
echo "Example: unbanUserFromCommunity(banID: \"1\")"

echo -e "\n8. Testing unmuteUserInCommunity (will need muteID from previous response)..."
echo "Note: You'll need to manually extract muteID from the communityUserMutes response above"
echo "Example: unmuteUserInCommunity(muteID: \"1\")"
