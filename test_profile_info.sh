#!/bin/bash

echo "Testing ProfileTableInfoItem mutations..."

# Тестовый запрос создания элемента профиля для сообщества
echo "1. Testing createProfileTableInfoItem for community..."
CREATE_RESPONSE=$(curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d '{"query":"mutation { createProfileTableInfoItem(input: { key: \"website\", value: \"https://example.com\", type: COMMUNITY, communityID: \"1\" }) { id key value type communityID } }"}' \
  -s)

echo "$CREATE_RESPONSE" | jq '.'

# Извлекаем ID созданного элемента для удаления
ITEM_ID=$(echo "$CREATE_RESPONSE" | jq -r '.data.createProfileTableInfoItem.id')

if [ "$ITEM_ID" != "null" ] && [ "$ITEM_ID" != "" ]; then
    echo -e "\n2. Testing updateProfileTableInfoItem..."
    curl -X POST http://localhost:8080/query \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
      -d "{\"query\":\"mutation { updateProfileTableInfoItem(input: { id: \\\"$ITEM_ID\\\", value: \\\"https://updated-example.com\\\" }) { id key value type } }\"}" \
      -s | jq '.'

    echo -e "\n3. Testing deleteProfileTableInfoItem..."
    curl -X POST http://localhost:8080/query \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
      -d "{\"query\":\"mutation { deleteProfileTableInfoItem(id: \\\"$ITEM_ID\\\") }\"}" \
      -s | jq '.'
else
    echo -e "\n2. Skipping update/delete tests - no item created"
fi

echo -e "\n4. Testing profileTableInfoItems query for community..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d '{"query":"query { profileTableInfoItems(id: \"1\", type: COMMUNITY) { id key value type communityID } }"}' \
  -s | jq '.'

echo -e "\n5. Testing createProfileTableInfoItem for user..."
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2MTcyNDEsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.V0tI90NPtSZ9pL6yEJZDXxe2CEj3DWUW1DAd_Acq-TA" \
  -d '{"query":"mutation { createProfileTableInfoItem(input: { key: \"location\", value: \"Moscow\", type: USER, userID: \"1\" }) { id key value type userID } }"}' \
  -s | jq '.'
