#!/bin/bash

echo "Testing GraphQL server..."

# Тестовый запрос community roles
echo "1. Testing community roles query..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d '{"query":"query { communityRoles(communityID: \"1\") { id title color communityRolesManagement communityUserBan communityUserMute communityDeletePost communityDeleteComments communityRemovePostFromPublication users { id name } } }"}' \
  -s | jq '.'

echo -e "\n2. Testing community query..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d '{"query":"query { community(id: \"1\") { id title ownerID } }"}' \
  -s | jq '.'
