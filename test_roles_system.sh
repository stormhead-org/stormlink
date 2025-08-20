#!/bin/bash

echo "Testing @everyone role system..."

# Тестовый запрос создания сообщества
echo "1. Testing createCommunity (should create @everyone role)..."
CREATE_COMMUNITY_RESPONSE=$(curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d '{"query":"mutation { createCommunity(input: { title: \"Test Community\", slug: \"test-community\", description: \"Test community for roles\", ownerID: \"1\" }) { id title slug ownerID } }"}' \
  -s)

echo "$CREATE_COMMUNITY_RESPONSE" | jq '.'

# Извлекаем ID созданного сообщества
COMMUNITY_ID=$(echo "$CREATE_COMMUNITY_RESPONSE" | jq -r '.data.createCommunity.id')

if [ "$COMMUNITY_ID" != "null" ] && [ "$COMMUNITY_ID" != "" ]; then
    echo -e "\n2. Testing communityRoles query (should show @everyone role)..."
    curl -X POST http://localhost:8080/query \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
      -d "{\"query\":\"query { communityRoles(communityID: \\\"$COMMUNITY_ID\\\") { id title color communityRolesManagement communityUserBan communityUserMute communityDeletePost communityDeleteComments communityRemovePostFromPublication users { id name } } }\"}" \
      -s | jq '.'

    echo -e "\n3. Testing followCommunity (should assign @everyone role)..."
    curl -X POST http://localhost:8080/query \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
      -d "{\"query\":\"mutation { followCommunity(input: { communityID: \\\"$COMMUNITY_ID\\\" }) { followersCount postsCount isBanned isMuted isFollowing } }\"}" \
      -s | jq '.'

    echo -e "\n4. Testing communityRoles query again (should show user in @everyone role)..."
    curl -X POST http://localhost:8080/query \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
      -d "{\"query\":\"query { communityRoles(communityID: \\\"$COMMUNITY_ID\\\") { id title color communityRolesManagement communityUserBan communityUserMute communityDeletePost communityDeleteComments communityRemovePostFromPublication users { id name } } }\"}" \
      -s | jq '.'
else
    echo -e "\n2. Skipping role tests - no community created"
fi
