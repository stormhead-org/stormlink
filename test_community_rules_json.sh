#!/bin/bash

echo "🧪 Тест функционала правил сообществ с JSON файлами"
echo "===================================================="

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080/query"

# Функция для выполнения GraphQL запроса из JSON файла
execute_query_from_file() {
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

# Создаем JSON файлы для запросов
echo -e "\n${YELLOW}2. Создание JSON файлов для запросов...${NC}"

# Файл для проверки схемы
cat > schema_query.json << 'EOF'
{
  "query": "query { __schema { mutationType { fields { name } } } }"
}
EOF

# Файл для авторизации
cat > login_query.json << 'EOF'
{
  "query": "mutation { loginUser(input: { email: \"gamenimsi@gmail.com\", password: \"qqwdqqwd\" }) { accessToken user { id name email } } }"
}
EOF

# Файл для получения правил
cat > get_rules_query.json << 'EOF'
{
  "query": "query { communityRules(communityID: \"1\") { id title description createdAt } }"
}
EOF

# Файл для создания правила
cat > create_rule_query.json << 'EOF'
{
  "query": "mutation { createCommunityRule(input: { communityID: \"1\", title: \"Тест\", description: \"Описание\" }) { id title } }"
}
EOF

# Файл для обновления правила
cat > update_rule_query.json << 'EOF'
{
  "query": "mutation { updateCommunityRule(input: { id: \"1\", title: \"Обновлено\", description: \"Новое описание\" }) { id title } }"
}
EOF

# Файл для удаления правила
cat > delete_rule_query.json << 'EOF'
{
  "query": "mutation { deleteCommunityRule(id: \"1\") }"
}
EOF

echo -e "${GREEN}✅ JSON файлы созданы${NC}"

# Проверка GraphQL схемы
echo -e "\n${YELLOW}3. Проверка GraphQL схемы...${NC}"
execute_query_from_file "Проверка мутаций" "schema_query.json"

# Тест авторизации
echo -e "\n${YELLOW}4. Тест авторизации...${NC}"
execute_query_from_file "Авторизация пользователя" "login_query.json"

# Тест получения правил (ожидается ошибка авторизации)
echo -e "\n${YELLOW}5. Тест получения правил сообщества...${NC}"
execute_query_from_file "Получение правил сообщества" "get_rules_query.json"

# Тест создания правила (ожидается ошибка авторизации)
echo -e "\n${YELLOW}6. Тест создания правила...${NC}"
execute_query_from_file "Создание правила" "create_rule_query.json"

# Тест обновления правила (ожидается ошибка авторизации)
echo -e "\n${YELLOW}7. Тест обновления правила...${NC}"
execute_query_from_file "Обновление правила" "update_rule_query.json"

# Тест удаления правила (ожидается ошибка авторизации)
echo -e "\n${YELLOW}8. Тест удаления правила...${NC}"
execute_query_from_file "Удаление правила" "delete_rule_query.json"

# Очистка временных файлов
echo -e "\n${YELLOW}9. Очистка временных файлов...${NC}"
rm -f schema_query.json login_query.json get_rules_query.json create_rule_query.json update_rule_query.json delete_rule_query.json
echo -e "${GREEN}✅ Временные файлы удалены${NC}"

echo -e "\n${GREEN}✅ Тестирование с JSON файлами завершено${NC}"
echo -e "\n${YELLOW}Выводы:${NC}"
echo -e "${YELLOW}- Если получаете ошибки 'unauthorized', это нормально - нужна авторизация${NC}"
echo -e "${YELLOW}- Если получаете 'internal system error', проверьте:${NC}"
echo -e "${YELLOW}  1. Подключение к базе данных${NC}"
echo -e "${YELLOW}  2. Существование таблицы community_rules${NC}"
echo -e "${YELLOW}  3. Запуск всех необходимых сервисов${NC}"
