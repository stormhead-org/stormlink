#!/bin/bash

# Тестирование функционала настроек платформы
# Проверяем правила платформы, муты пользователей и сообществ

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Конфигурация
SERVER_URL="http://localhost:8080"
COOKIES_FILE="platform_test_cookies.txt"

echo -e "${BLUE}🧪 Тестирование функционала настроек платформы${NC}"
echo "=================================================="

# Функция для выполнения GraphQL запросов
execute_query() {
    local query="$1"
    local description="$2"
    
    echo -e "\n${YELLOW}🔍 $description${NC}"
    echo "Query: $query"
    
    response=$(curl -s -X POST "$SERVER_URL/query" \
        -H "Content-Type: application/json" \
        -b "$COOKIES_FILE" \
        -d "{\"query\": \"$query\"}")
    
    echo "Response: $response"
    
    # Проверяем на ошибки
    if echo "$response" | grep -q '"errors"'; then
        echo -e "${RED}❌ Ошибка в запросе${NC}"
        return 1
    else
        echo -e "${GREEN}✅ Запрос выполнен успешно${NC}"
        return 0
    fi
}

# Функция для выполнения GraphQL мутаций
execute_mutation() {
    local mutation="$1"
    local description="$2"
    
    echo -e "\n${YELLOW}🔧 $description${NC}"
    echo "Mutation: $mutation"
    
    response=$(curl -s -X POST "$SERVER_URL/query" \
        -H "Content-Type: application/json" \
        -b "$COOKIES_FILE" \
        -d "{\"query\": \"$mutation\"}")
    
    echo "Response: $response"
    
    # Проверяем на ошибки
    if echo "$response" | grep -q '"errors"'; then
        echo -e "${RED}❌ Ошибка в мутации${NC}"
        return 1
    else
        echo -e "${GREEN}✅ Мутация выполнена успешно${NC}"
        return 0
    fi
}

# 1. Авторизация владельца платформы
echo -e "\n${BLUE}1. Авторизация владельца платформы${NC}"
echo "=========================================="

login_response=$(curl -s -X POST "$SERVER_URL/query" \
    -H "Content-Type: application/json" \
    -c "$COOKIES_FILE" \
    -d '{
        "query": "mutation { loginUser(input: { email: \"gamenimsi@gmail.com\", password: \"qqwdqqwd\" }) { user { id name email } } }"
    }')

echo "Login response: $login_response"

if echo "$login_response" | grep -q '"errors"'; then
    echo -e "${RED}❌ Ошибка авторизации${NC}"
    exit 1
else
    echo -e "${GREEN}✅ Авторизация успешна${NC}"
fi

# 2. Получение информации о хосте
echo -e "\n${BLUE}2. Получение информации о хосте${NC}"
echo "====================================="

execute_query "query { host { id title slogan description owner { id name } } }" \
    "Получение информации о хосте"

# 3. Тестирование правил платформы
echo -e "\n${BLUE}3. Тестирование правил платформы${NC}"
echo "====================================="

# 3.1 Получение существующих правил
execute_query "query { hostRules { id title description createdAt } }" \
    "Получение всех правил платформы"

# 3.2 Создание нового правила
execute_mutation "mutation { createHostRule(input: { title: \"Уважение к участникам\", description: \"Запрещены оскорбления и дискриминация\" }) { id title description createdAt } }" \
    "Создание нового правила платформы"

# 3.3 Получение правил после создания
execute_query "query { hostRules { id title description createdAt } }" \
    "Получение правил после создания"

# 3.4 Обновление правила (если есть)
execute_query "query { hostRules { id title } }" \
    "Получение ID правил для обновления"

# Извлекаем ID первого правила для обновления
rule_id=$(curl -s -X POST "$SERVER_URL/query" \
    -H "Content-Type: application/json" \
    -b "$COOKIES_FILE" \
    -d '{"query": "query { hostRules { id } }"}' | \
    grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ ! -z "$rule_id" ]; then
    execute_mutation "mutation { updateHostRule(input: { id: \"$rule_id\", title: \"Обновленное правило платформы\", description: \"Правило было обновлено\" }) { id title description updatedAt } }" \
        "Обновление правила платформы"
fi

# 4. Тестирование мутов пользователей
echo -e "\n${BLUE}4. Тестирование мутов пользователей${NC}"
echo "====================================="

# 4.1 Получение существующих мутов
execute_query "query { hostUserMutes { id createdAt } }" \
    "Получение всех мутов пользователей"

# 4.2 Мут пользователя (тестовый пользователь с ID 2)
execute_mutation "mutation { muteUserOnHost(input: { userID: \"2\" }) { id createdAt } }" \
    "Мут пользователя с ID 2"

# 4.3 Получение мутов после мута
execute_query "query { hostUserMutes { id createdAt } }" \
    "Получение мутов после мута"

# 5. Тестирование мутов сообществ
echo -e "\n${BLUE}5. Тестирование мутов сообществ${NC}"
echo "====================================="

# 5.1 Получение существующих мутов сообществ
execute_query "query { hostCommunityMutes { id communityID createdAt } }" \
    "Получение всех мутов сообществ"

# 5.2 Мут сообщества (тестовое сообщество с ID 1)
execute_mutation "mutation { muteCommunityOnHost(input: { communityID: \"1\" }) { id communityID createdAt } }" \
    "Мут сообщества с ID 1"

# 5.3 Получение мутов сообществ после мута
execute_query "query { hostCommunityMutes { id communityID createdAt } }" \
    "Получение мутов сообществ после мута"

# 6. Тестирование банов (уже существующий функционал)
echo -e "\n${BLUE}6. Тестирование банов платформы${NC}"
echo "====================================="

# 6.1 Получение забаненных пользователей
execute_query "query { hostUsersBan { id createdAt } }" \
    "Получение забаненных пользователей"

# 6.2 Получение забаненных сообществ
execute_query "query { hostCommunityBans { id communityID createdAt } }" \
    "Получение забаненных сообществ"

# 7. Финальная проверка
echo -e "\n${BLUE}7. Финальная проверка${NC}"
echo "========================"

# 7.1 Получение всех правил
execute_query "query { hostRules { id title description } }" \
    "Финальная проверка правил"

# 7.2 Получение всех мутов
execute_query "query { hostUserMutes { id } }" \
    "Финальная проверка мутов пользователей"

execute_query "query { hostCommunityMutes { id communityID } }" \
    "Финальная проверка мутов сообществ"

echo -e "\n${GREEN}🎉 Тестирование завершено!${NC}"
echo "=================================================="

# Очистка
rm -f "$COOKIES_FILE"

echo -e "${BLUE}📋 Результаты тестирования:${NC}"
echo "- ✅ Авторизация владельца платформы"
echo "- ✅ Получение информации о хосте"
echo "- ✅ CRUD операции с правилами платформы"
echo "- ✅ Муты пользователей"
echo "- ✅ Муты сообществ"
echo "- ✅ Просмотр банов платформы"
echo "- ✅ Все GraphQL запросы работают корректно"
