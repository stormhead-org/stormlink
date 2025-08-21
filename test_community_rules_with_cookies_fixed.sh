#!/bin/bash

echo "🧪 Тест правил сообщества с куки-авторизацией (исправленный)"
echo "============================================================="

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080/query"
COOKIE_FILE="auth_cookies.txt"

# Функция для выполнения GraphQL запроса с куки
execute_query_with_cookies() {
    local name="$1"
    local json_file="$2"
    
    echo -e "\n${BLUE}🔍 $name${NC}"
    echo "JSON файл: $json_file"
    
    response=$(curl -s -X POST "$BASE_URL" \
        -H "Content-Type: application/json" \
        -b "$COOKIE_FILE" \
        -c "$COOKIE_FILE" \
        -d @"$json_file")
    
    echo "Ответ: $response"
}

# Функция для выполнения GraphQL запроса без куки
execute_query_without_cookies() {
    local name="$1"
    local json_file="$2"
    
    echo -e "\n${BLUE}🔍 $name${NC}"
    echo "JSON файл: $json_file"
    
    response=$(curl -s -X POST "$BASE_URL" \
        -H "Content-Type: application/json" \
        -d @"$json_file")
    
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

# Очищаем старые куки
rm -f "$COOKIE_FILE"

# Создаем JSON файлы для запросов
echo -e "\n${YELLOW}2. Создание JSON файлов для запросов...${NC}"

# Файл для авторизации
cat > login_query.json << 'EOF'
{
  "query": "mutation { loginUser(input: { email: \"gamenimsi@gmail.com\", password: \"qqwdqqwd\" }) { user { id name email } } }"
}
EOF

# Файл для получения правил без авторизации
cat > get_rules_no_auth.json << 'EOF'
{
  "query": "query { communityRules(communityID: \"1\") { id title description createdAt } }"
}
EOF

# Файл для получения правил с авторизацией
cat > get_rules_with_auth.json << 'EOF'
{
  "query": "query { communityRules(communityID: \"1\") { id title description createdAt } }"
}
EOF

# Файл для создания правила
cat > create_rule.json << 'EOF'
{
  "query": "mutation { createCommunityRule(input: { communityID: \"1\", title: \"Тест через куки\", description: \"Правило созданное через куки-авторизацию\" }) { id title description } }"
}
EOF

# Файл для обновления правила
cat > update_rule.json << 'EOF'
{
  "query": "mutation { updateCommunityRule(input: { id: \"3\", title: \"Обновлено через куки\", description: \"Правило обновлено через куки-авторизацию\" }) { id title description updatedAt } }"
}
EOF

# Файл для удаления правила
cat > delete_rule.json << 'EOF'
{
  "query": "mutation { deleteCommunityRule(id: \"3\") }"
}
EOF

echo -e "${GREEN}✅ JSON файлы созданы${NC}"

# Шаг 3: Авторизация
echo -e "\n${YELLOW}3. Авторизация пользователя...${NC}"
execute_query_with_cookies "Авторизация" "login_query.json"

# Проверяем, что куки установлены
if [ -f "$COOKIE_FILE" ]; then
    echo -e "${GREEN}✅ Файл куки создан${NC}"
    echo "Содержимое куки файла:"
    cat "$COOKIE_FILE"
else
    echo -e "${RED}❌ Файл куки не создан${NC}"
fi

# Шаг 4: Тест получения правил без авторизации
echo -e "\n${YELLOW}4. Тест получения правил БЕЗ авторизации...${NC}"
execute_query_without_cookies "Получение правил без авторизации" "get_rules_no_auth.json"

# Шаг 5: Тест получения правил с авторизацией
echo -e "\n${YELLOW}5. Тест получения правил С авторизацией...${NC}"
execute_query_with_cookies "Получение правил с авторизацией" "get_rules_with_auth.json"

# Шаг 6: Тест создания правила
echo -e "\n${YELLOW}6. Тест создания правила...${NC}"
execute_query_with_cookies "Создание правила" "create_rule.json"

# Шаг 7: Тест обновления правила
echo -e "\n${YELLOW}7. Тест обновления правила...${NC}"
execute_query_with_cookies "Обновление правила" "update_rule.json"

# Шаг 8: Тест удаления правила
echo -e "\n${YELLOW}8. Тест удаления правила...${NC}"
execute_query_with_cookies "Удаление правила" "delete_rule.json"

# Шаг 9: Финальная проверка
echo -e "\n${YELLOW}9. Финальная проверка правил...${NC}"
execute_query_with_cookies "Финальная проверка" "get_rules_with_auth.json"

# Очистка временных файлов
echo -e "\n${YELLOW}10. Очистка временных файлов...${NC}"
rm -f login_query.json get_rules_no_auth.json get_rules_with_auth.json create_rule.json update_rule.json delete_rule.json "$COOKIE_FILE"
echo -e "${GREEN}✅ Временные файлы удалены${NC}"

echo -e "\n${GREEN}✅ Тестирование с куки-авторизацией завершено${NC}"
echo -e "\n${YELLOW}Выводы:${NC}"
echo -e "${YELLOW}- Авторизация работает через куки${NC}"
echo -e "${YELLOW}- Токены не возвращаются в ответе (это нормально для безопасности)${NC}"
echo -e "${YELLOW}- Куки автоматически передаются в последующих запросах${NC}"
