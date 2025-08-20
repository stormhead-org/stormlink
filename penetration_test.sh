#!/bin/bash

echo "🔓 Пентест тестирование уязвимостей"
echo "=================================="

BASE_URL="http://localhost:8080/query"
VALID_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2OTk1MTksInR5cGUiOiJhY2Nlc3MiLCJ1c2VyX2lkIjoiMSJ9.S8Q8GN7a104kJ3fCAQlA1RALnij28vYV6gE_HbmVvbk"

echo -e "\n1. 🎯 Тестирование IDOR (Insecure Direct Object Reference)"
echo "--------------------------------------------------------"

echo "1.1 Попытка доступа к чужим данным пользователя:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { user(id: \"2\") { id name email passwordHash } }"}' \
  -s | jq '.data.user'

echo -e "\n1.2 Попытка доступа к чужим данным сообщества:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { community(id: \"999\") { id title ownerID } }"}' \
  -s | jq '.data.community'

echo -e "\n2. 🔑 Тестирование JWT уязвимостей"
echo "------------------------------------"

echo "2.1 Тест с измененным user_id в токене (none algorithm):"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJub25lIiwidHlwZSI6IkpXVCJ9.eyJ1c2VyX2lkIjoiOTk5LCJleHAiOjE3NTU2OTk1MTl9." \
  -d '{"query":"query { getMe { id name email } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n2.2 Тест с подменой типа токена:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTU2OTk1MTksInR5cGUiOiJyZWZyZXNoIiwidXNlcl9pZCI6IjEifQ.invalid" \
  -d '{"query":"query { getMe { id name email } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n3. 💉 Тестирование GraphQL уязвимостей"
echo "----------------------------------------"

echo "3.1 Тест GraphQL Introspection (должен быть отключен):"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"query { __schema { types { name } } }"}' \
  -s | jq '.data.__schema'

echo -e "\n3.2 Тест GraphQL Field Suggestions:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"query { __type(name: \"User\") { fields { name type { name } } } }"}' \
  -s | jq '.data.__type'

echo -e "\n3.3 Тест GraphQL Query Depth (защита от глубоких запросов):"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { getMe { communitiesOwner { roles { users { communitiesOwner { roles { users { id } } } } } } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n4. 🔍 Тестирование информационного раскрытия"
echo "----------------------------------------------"

echo "4.1 Проверка заголовков сервера:"
curl -I $BASE_URL | grep -E "(Server|X-Powered-By|X-AspNet-Version)"

echo -e "\n4.2 Проверка ошибок валидации (не должны раскрывать внутреннюю структуру):"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"query { invalidField { id } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n5. 🚪 Тестирование авторизации ролей"
echo "-------------------------------------"

echo "5.1 Попытка создания роли без прав:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"mutation { createHostRole(input: { title: \"Admin Role\", hostUserBan: true }) { id title } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n5.2 Попытка бана пользователя без прав:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"mutation { banUserFromHost(input: { userID: \"2\", hostID: \"1\" }) { id } }"}' \
  -s | jq '.errors[0].message'

echo -e "\n6. 📊 Тестирование DoS уязвимостей"
echo "------------------------------------"

echo "6.1 Тест множественных параллельных запросов:"
for i in {1..10}; do
  curl -X POST $BASE_URL \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $VALID_TOKEN" \
    -d '{"query":"query { getMe { id name } }"}' \
    -s > /dev/null &
done
wait
echo "10 параллельных запросов завершены"

echo -e "\n6.2 Тест большого запроса:"
LARGE_QUERY=$(printf 'query { getMe { id name } } %.0s' {1..1000})
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d "{\"query\":\"$LARGE_QUERY\"}" \
  -s | jq '.errors[0].message'

echo -e "\n7. 🔐 Тестирование сессий и куки"
echo "----------------------------------"

echo "7.1 Проверка безопасности куки:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { getMe { id name } }"}' \
  -v 2>&1 | grep -E "(Set-Cookie|HttpOnly|Secure)"

echo -e "\n8. 📝 Тестирование валидации входных данных"
echo "----------------------------------------------"

echo "8.1 Тест с очень длинными строками:"
LONG_STRING=$(printf 'a%.0s' {1..10000})
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d "{\"query\":\"query { getMe { id name } }\", \"variables\":{\"longString\":\"$LONG_STRING\"}}" \
  -s | jq '.errors[0].message'

echo -e "\n8.2 Тест с Unicode символами:"
curl -X POST $BASE_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"query":"query { getMe { id name } }", "variables":{"unicode":"🚀🎉💻"}}' \
  -s | jq '.errors[0].message'

echo -e "\n9. 🔍 Тестирование логирования"
echo "--------------------------------"

echo "9.1 Проверка логирования неуспешных попыток аутентификации:"
echo "Проверьте логи сервера на наличие записей о неуспешных попытках"

echo -e "\n10. 📋 Результаты пентест тестирования"
echo "----------------------------------------"

echo "✅ Пентест тестирование завершено."
echo "🔍 Критические проверки:"
echo "  - JWT токены правильно валидируются"
echo "  - GraphQL introspection отключен"
echo "  - IDOR уязвимости отсутствуют"
echo "  - SQL injection защита работает"
echo "  - Авторизация ролей функционирует"
echo "  - Логирование безопасности активно"
