#!/bin/bash

echo "🧪 Полный тест функционала правил сообществ"
echo "=============================================="

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Переменные
EMAIL="gamenimsi@gmail.com"
PASSWORD="qqwdqqwd"
COMMUNITY_ID="1"
BASE_URL="http://localhost:8080/query"

# Функция для выполнения GraphQL запроса
execute_query() {
    local name="$1"
    local query="$2"
    local token="$3"
    
    echo -e "\n${BLUE}🔍 $name${NC}"
    echo "Запрос: $query"
    
    local headers="Content-Type: application/json"
    if [ -n "$token" ]; then
        headers="$headers\nAuthorization: Bearer $token"
    fi
    
    response=$(curl -s -X POST "$BASE_URL" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "{\"query\": \"$query\"}")
    
    echo "Ответ: $response"
    echo "$response"
}

# Функция для извлечения значения из JSON ответа
extract_value() {
    local json="$1"
    local field="$2"
    echo "$json" | grep -o "\"$field\":\"[^\"]*\"" | cut -d'"' -f4
}

# Функция для извлечения ID из JSON ответа
extract_id() {
    local json="$1"
    echo "$json" | grep -o "\"id\":\"[^\"]*\"" | cut -d'"' -f4
}

# Проверяем, что сервер запущен
echo -e "${YELLOW}1. Проверка доступности сервера...${NC}"
if curl -s "$BASE_URL" > /dev/null; then
    echo -e "${GREEN}✅ Сервер доступен${NC}"
else
    echo -e "${RED}❌ Сервер недоступен${NC}"
    exit 1
fi

# Шаг 2: Авторизация
echo -e "\n${YELLOW}2. Авторизация пользователя...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { loginUser(input: { email: \"'$EMAIL'\", password: \"'$PASSWORD'\" }) { accessToken user { id name email } } }"
    }')

echo "Ответ авторизации: $LOGIN_RESPONSE"

# Извлекаем токен
TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"accessToken":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo -e "${RED}❌ Не удалось получить токен авторизации${NC}"
    echo "Возможные причины:"
    echo "1. Неверные учетные данные"
    echo "2. Пользователь не существует"
    echo "3. Проблемы с сервером"
    exit 1
fi

echo -e "${GREEN}✅ Токен получен: ${TOKEN:0:20}...${NC}"

# Шаг 3: Проверка существующих правил
echo -e "\n${YELLOW}3. Проверка существующих правил сообщества...${NC}"
execute_query "Получение правил сообщества" \
    "query { communityRules(communityID: \"$COMMUNITY_ID\") { id title description createdAt } }" \
    "$TOKEN"

# Шаг 4: Создание 5 правил
echo -e "\n${YELLOW}4. Создание 5 правил сообщества...${NC}"

RULE_IDS=()

# Правило 1
echo -e "\n${BLUE}Создание правила 1: Уважение к участникам${NC}"
RESPONSE=$(execute_query "Создание правила 1" \
    "mutation { createCommunityRule(input: { communityID: \"$COMMUNITY_ID\", title: \"Уважение к участникам\", description: \"Запрещены оскорбления, угрозы и дискриминация участников сообщества\" }) { id title description } }" \
    "$TOKEN")

RULE_ID=$(extract_id "$RESPONSE")
if [ -n "$RULE_ID" ]; then
    RULE_IDS+=("$RULE_ID")
    echo -e "${GREEN}✅ Правило 1 создано с ID: $RULE_ID${NC}"
else
    echo -e "${RED}❌ Ошибка создания правила 1${NC}"
fi

# Правило 2
echo -e "\n${BLUE}Создание правила 2: Качество контента${NC}"
RESPONSE=$(execute_query "Создание правила 2" \
    "mutation { createCommunityRule(input: { communityID: \"$COMMUNITY_ID\", title: \"Качество контента\", description: \"Посты должны содержать полезную информацию и соответствовать тематике сообщества\" }) { id title description } }" \
    "$TOKEN")

RULE_ID=$(extract_id "$RESPONSE")
if [ -n "$RULE_ID" ]; then
    RULE_IDS+=("$RULE_ID")
    echo -e "${GREEN}✅ Правило 2 создано с ID: $RULE_ID${NC}"
else
    echo -e "${RED}❌ Ошибка создания правила 2${NC}"
fi

# Правило 3
echo -e "\n${BLUE}Создание правила 3: Запрет спама${NC}"
RESPONSE=$(execute_query "Создание правила 3" \
    "mutation { createCommunityRule(input: { communityID: \"$COMMUNITY_ID\", title: \"Запрет спама\", description: \"Запрещена публикация рекламы, спама и коммерческих предложений без согласования\" }) { id title description } }" \
    "$TOKEN")

RULE_ID=$(extract_id "$RESPONSE")
if [ -n "$RULE_ID" ]; then
    RULE_IDS+=("$RULE_ID")
    echo -e "${GREEN}✅ Правило 3 создано с ID: $RULE_ID${NC}"
else
    echo -e "${RED}❌ Ошибка создания правила 3${NC}"
fi

# Правило 4
echo -e "\n${BLUE}Создание правила 4: Конфиденциальность${NC}"
RESPONSE=$(execute_query "Создание правила 4" \
    "mutation { createCommunityRule(input: { communityID: \"$COMMUNITY_ID\", title: \"Конфиденциальность\", description: \"Запрещено разглашение личной информации других участников без их согласия\" }) { id title description } }" \
    "$TOKEN")

RULE_ID=$(extract_id "$RESPONSE")
if [ -n "$RULE_ID" ]; then
    RULE_IDS+=("$RULE_ID")
    echo -e "${GREEN}✅ Правило 4 создано с ID: $RULE_ID${NC}"
else
    echo -e "${RED}❌ Ошибка создания правила 4${NC}"
fi

# Правило 5
echo -e "\n${BLUE}Создание правила 5: Язык общения${NC}"
RESPONSE=$(execute_query "Создание правила 5" \
    "mutation { createCommunityRule(input: { communityID: \"$COMMUNITY_ID\", title: \"Язык общения\", description: \"Основной язык общения в сообществе - русский. Допускается использование английского языка\" }) { id title description } }" \
    "$TOKEN")

RULE_ID=$(extract_id "$RESPONSE")
if [ -n "$RULE_ID" ]; then
    RULE_IDS+=("$RULE_ID")
    echo -e "${GREEN}✅ Правило 5 создано с ID: $RULE_ID${NC}"
else
    echo -e "${RED}❌ Ошибка создания правила 5${NC}"
fi

# Шаг 5: Проверка созданных правил
echo -e "\n${YELLOW}5. Проверка всех созданных правил...${NC}"
execute_query "Получение всех правил" \
    "query { communityRules(communityID: \"$COMMUNITY_ID\") { id title description createdAt } }" \
    "$TOKEN"

# Шаг 6: Обновление правил
echo -e "\n${YELLOW}6. Обновление правил...${NC}"

if [ ${#RULE_IDS[@]} -gt 0 ]; then
    # Обновляем первое правило
    echo -e "\n${BLUE}Обновление правила 1...${NC}"
    execute_query "Обновление правила 1" \
        "mutation { updateCommunityRule(input: { id: \"${RULE_IDS[0]}\", title: \"Уважение к участникам (обновлено)\", description: \"Запрещены оскорбления, угрозы, дискриминация и токсичное поведение\" }) { id title description updatedAt } }" \
        "$TOKEN"
    
    # Обновляем третье правило
    if [ ${#RULE_IDS[@]} -gt 2 ]; then
        echo -e "\n${BLUE}Обновление правила 3...${NC}"
        execute_query "Обновление правила 3" \
            "mutation { updateCommunityRule(input: { id: \"${RULE_IDS[2]}\", description: \"Запрещена публикация рекламы, спама, коммерческих предложений и нежелательного контента без предварительного согласования с модераторами\" }) { id title description updatedAt } }" \
            "$TOKEN"
    fi
fi

# Шаг 7: Удаление правил
echo -e "\n${YELLOW}7. Удаление правил...${NC}"

if [ ${#RULE_IDS[@]} -gt 1 ]; then
    # Удаляем второе правило
    echo -e "\n${BLUE}Удаление правила 2...${NC}"
    execute_query "Удаление правила 2" \
        "mutation { deleteCommunityRule(id: \"${RULE_IDS[1]}\") }" \
        "$TOKEN"
    
    # Удаляем четвертое правило
    if [ ${#RULE_IDS[@]} -gt 3 ]; then
        echo -e "\n${BLUE}Удаление правила 4...${NC}"
        execute_query "Удаление правила 4" \
            "mutation { deleteCommunityRule(id: \"${RULE_IDS[3]}\") }" \
            "$TOKEN"
    fi
fi

# Шаг 8: Финальная проверка
echo -e "\n${YELLOW}8. Финальная проверка оставшихся правил...${NC}"
execute_query "Финальная проверка правил" \
    "query { communityRules(communityID: \"$COMMUNITY_ID\") { id title description createdAt updatedAt } }" \
    "$TOKEN"

# Шаг 9: Проверка отдельных правил
echo -e "\n${YELLOW}9. Проверка отдельных правил...${NC}"

if [ ${#RULE_IDS[@]} -gt 0 ]; then
    # Проверяем первое правило
    echo -e "\n${BLUE}Проверка правила 1...${NC}"
    execute_query "Получение правила 1" \
        "query { communityRule(id: \"${RULE_IDS[0]}\") { id title description createdAt updatedAt community { id title } } }" \
        "$TOKEN"
    
    # Проверяем третье правило
    if [ ${#RULE_IDS[@]} -gt 2 ]; then
        echo -e "\n${BLUE}Проверка правила 3...${NC}"
        execute_query "Получение правила 3" \
            "query { communityRule(id: \"${RULE_IDS[2]}\") { id title description createdAt updatedAt } }" \
            "$TOKEN"
    fi
fi

echo -e "\n${GREEN}✅ Тестирование завершено${NC}"
echo -e "\n${YELLOW}Результаты:${NC}"
echo -e "${YELLOW}- Создано правил: ${#RULE_IDS[@]}${NC}"
echo -e "${YELLOW}- Обновлено правил: 2${NC}"
echo -e "${YELLOW}- Удалено правил: 2${NC}"
echo -e "${YELLOW}- Осталось правил: $(( ${#RULE_IDS[@]} - 2 ))${NC}"
