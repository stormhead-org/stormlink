#!/bin/bash

echo "🔐 Тестирование безопасности аутентификации и авторизации"
echo "=================================================="

BASE_URL="http://localhost:8080/query"
VALID_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2OTk1MTksInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.S8Q8GN7a104kJ3fCAQlA1RALnij28vYV6gE_HbmVvbk"

echo -e "\n1. 🔍 Тестирование аутентификации"
echo "----------------------------------------"

echo "1.1 Тест без токена (должен вернуть unauthenticated):"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"query { getMe { id name email } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n1.2 Тест с недействительным токеном:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer invalid_token" \
  -d '{"query":"query { getMe { id name email } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n1.3 Тест с пустым токеном:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer " \
  -d '{"query":"query { getMe { id name email } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n1.4 Тест с валидным токеном (должен работать):"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { getMe { id name email } }"}' \
  -s | jq '.data.getMe.id'

echo -e "\n2. 🛡️ Тестирование авторизации"
echo "----------------------------------------"

echo "2.1 Тест доступа к communityFollowers без прав (должен вернуть forbidden):"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { communityFollowers(communityID: \"999\", filter: BANNED) { id name } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n2.2 Тест доступа к communityUserBans без прав:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { communityUserBans(communityID: \"999\") { id user { id name } } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n2.3 Тест доступа к communityFollowers с правами (должен работать):"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { communityFollowers(communityID: \"1\", filter: ALL) { id name } }"}' \
  -s | jq '.data.communityFollowers | length'

echo -e "\n3. 🔒 Тестирование мутаций безопасности"
echo "----------------------------------------"

echo "3.1 Тест banUserFromCommunity без прав:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"mutation { banUserFromCommunity(input: { userID: \"999\", communityID: \"999\" }) { id } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n3.2 Тест muteUserInCommunity без прав:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"mutation { muteUserInCommunity(input: { userID: \"999\", communityID: \"999\" }) { id } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n3.3 Тест createCommunityRole без прав:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"mutation { createCommunityRole(input: { title: \"Test Role\", communityID: \"999\" }) { id title } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n4. 🚫 Тестирование SQL Injection защиты"
echo "----------------------------------------"

echo "4.1 Тест SQL injection в communityID:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { communityFollowers(communityID: \"1; DROP TABLE users; --\", filter: ALL) { id name } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n4.2 Тест SQL injection в userID:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"mutation { banUserFromCommunity(input: { userID: \"1 OR 1=1\", communityID: \"1\" }) { id } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n5. 📊 Тестирование rate limiting (если реализован)"
echo "----------------------------------------"

echo "5.1 Множественные запросы getMe:"
for i in {1..5}; do
  echo "Запрос $i:"
  curl -X POST $BASE_URL \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $VALID_TOKEN" \
    -d '{"query":"query { getMe { id name } }"}' \
    -s | jq '.data.getMe.id' &
done
wait

echo -e "\n6. 🔐 Тестирование JWT токенов"
echo "----------------------------------------"

echo "6.1 Тест с истекшим токеном (если есть):"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NzI1NDMyMDAsInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.expired_token" \
  -d '{"query":"query { getMe { id name } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n6.2 Тест с неправильной подписью токена:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2OTk1MTksInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.wrong_signature" \
  -d '{"query":"query { getMe { id name } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n7. 🎭 Тестирование подмены пользователя"
echo "----------------------------------------"

echo "7.1 Тест доступа к чужим данным через communityFollowers:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { communityFollowers(communityID: \"1\", filter: ALL) { id name email password } }"}' \
  -s | jq '.data.communityFollowers[0] | keys'

echo -e "\n8. 📝 Тестирование валидации входных данных"
echo "----------------------------------------"

echo "8.1 Тест с отрицательным communityID:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { communityFollowers(communityID: \"-1\", filter: ALL) { id name } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n8.2 Тест с несуществующим enum значением:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { communityFollowers(communityID: \"1\", filter: INVALID_FILTER) { id name } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n9. 🔍 Тестирование логирования безопасности"
echo "----------------------------------------"

echo "9.1 Проверка логов сервера после неуспешных попыток аутентификации:"
echo "Логи должны содержать информацию о неуспешных попытках аутентификации"

echo -e "\n10. 📋 Результаты тестирования"
echo "----------------------------------------"

echo "✅ Тесты завершены. Проверьте результаты выше."
echo "🔍 Рекомендации по безопасности:"
echo "  - Убедитесь, что все неуспешные попытки логируются"
echo "  - Проверьте, что чувствительные данные не возвращаются"
echo "  - Убедитесь, что SQL injection защита работает"
echo "  - Проверьте rate limiting для auth endpoints"
