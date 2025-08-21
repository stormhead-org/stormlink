#!/bin/bash

echo "🧪 Исправленный тест функционала правил сообществ"
echo "=================================================="

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080/query"

# Функция для выполнения GraphQL запроса
execute_query() {
    local name="$1"
    local query="$2"
    
    echo -e "\n${BLUE}🔍 $name${NC}"
    echo "Запрос: $query"
    
    # Создаем JSON с правильным экранированием
    json_data=$(cat <<EOF
{
  "query": "$query"
}
EOF
)
    
    response=$(curl -s -X POST "$BASE_URL" \
        -H "Content-Type: application/json" \
        -d "$json_data")
    
    echo "Ответ: $response"
}

# Проверяем, что сервер запущен
echo -e "${YELLOW}1. Проверка доступности сервера...${NC}"
if curl -s "$BASE_URL" > /dev/null; then
    echo -e "${GREEN}✅ Сервер доступен${NC}"
else
    echo -e "${RED}❌ Сервер недоступен${NC}"
    exit 1
fi

# Проверка GraphQL схемы
echo -e "\n${YELLOW}2. Проверка GraphQL схемы...${NC}"
execute_query "Проверка мутаций" \
    "query { __schema { mutationType { fields { name } } } }"

execute_query "Проверка запросов" \
    "query { __schema { queryType { fields { name } } } }"

# Тест авторизации
echo -e "\n${YELLOW}3. Тест авторизации...${NC}"
execute_query "Авторизация пользователя" \
    "mutation { loginUser(input: { email: \"gamenimsi@gmail.com\", password: \"qqwdqqwd\" }) { accessToken user { id name email } } }"

# Тест получения правил (ожидается ошибка авторизации)
echo -e "\n${YELLOW}4. Тест получения правил сообщества...${NC}"
execute_query "Получение правил сообщества" \
    "query { communityRules(communityID: \"1\") { id title description createdAt } }"

# Тест создания правила (ожидается ошибка авторизации)
echo -e "\n${YELLOW}5. Тест создания правила...${NC}"
execute_query "Создание правила" \
    "mutation { createCommunityRule(input: { communityID: \"1\", title: \"Тест\", description: \"Описание\" }) { id title } }"

# Тест обновления правила (ожидается ошибка авторизации)
echo -e "\n${YELLOW}6. Тест обновления правила...${NC}"
execute_query "Обновление правила" \
    "mutation { updateCommunityRule(input: { id: \"1\", title: \"Обновлено\", description: \"Новое описание\" }) { id title } }"

# Тест удаления правила (ожидается ошибка авторизации)
echo -e "\n${YELLOW}7. Тест удаления правила...${NC}"
execute_query "Удаление правила" \
    "mutation { deleteCommunityRule(id: \"1\") }"

echo -e "\n${GREEN}✅ Исправленное тестирование завершено${NC}"
echo -e "\n${YELLOW}Выводы:${NC}"
echo -e "${YELLOW}- Если получаете ошибки 'unauthorized', это нормально - нужна авторизация${NC}"
echo -e "${YELLOW}- Если получаете 'internal system error', проверьте:${NC}"
echo -e "${YELLOW}  1. Подключение к базе данных${NC}"
echo -e "${YELLOW}  2. Существование таблицы community_rules${NC}"
echo -e "${YELLOW}  3. Запуск всех необходимых сервисов${NC}"
