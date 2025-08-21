#!/bin/bash

echo "🧪 Тестирование функционала правил сообществ"
echo "=============================================="

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Функция для тестирования
test_query() {
    local name="$1"
    local query="$2"
    local expected_error="$3"
    
    echo -e "\n${YELLOW}Тест: $name${NC}"
    echo "Запрос: $query"
    
    response=$(curl -s -X POST http://localhost:8080/query \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"$query\"}" | sed 's/"/\\"/g')
    
    echo "Ответ: $response"
    
    if [ -n "$expected_error" ]; then
        if echo "$response" | grep -q "$expected_error"; then
            echo -e "${GREEN}✅ Ожидаемая ошибка получена: $expected_error${NC}"
        else
            echo -e "${RED}❌ Ожидаемая ошибка не получена${NC}"
        fi
    else
        if echo "$response" | grep -q "errors"; then
            echo -e "${RED}❌ Получена ошибка${NC}"
        else
            echo -e "${GREEN}✅ Успешно${NC}"
        fi
    fi
}

# Проверяем, что сервер запущен
echo "🔍 Проверка доступности сервера..."
if ! curl -s http://localhost:8080/query > /dev/null; then
    echo -e "${RED}❌ Сервер недоступен на localhost:8080${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Сервер доступен${NC}"

# Тест 1: Получение правил сообщества (должен вернуть ошибку авторизации)
test_query \
    "Получение правил сообщества" \
    "query { communityRules(communityID: \"1\") { id title description createdAt } }" \
    "unauthorized"

# Тест 2: Создание правила без авторизации (должен вернуть ошибку авторизации)
test_query \
    "Создание правила без авторизации" \
    "mutation { createCommunityRule(input: { communityID: \"1\", title: \"Тест\", description: \"Описание\" }) { id title } }" \
    "unauthorized"

# Тест 3: Получение одного правила (должен вернуть ошибку авторизации)
test_query \
    "Получение одного правила" \
    "query { communityRule(id: \"1\") { id title description } }" \
    "unauthorized"

echo -e "\n${GREEN}✅ Тестирование завершено${NC}"
echo -e "\n${YELLOW}Примечание: Все тесты должны возвращать ошибки авторизации, так как запросы выполняются без токена.${NC}"
echo -e "${YELLOW}Для полного тестирования необходимо:${NC}"
echo -e "${YELLOW}1. Выполнить миграцию: psql -h localhost -U postgres -d stormlink -f migrate_community_rules.sql${NC}"
echo -e "${YELLOW}2. Получить токен авторизации${NC}"
echo -e "${YELLOW}3. Выполнить запросы с токеном${NC}"
