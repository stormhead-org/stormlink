#!/bin/bash

echo "🧪 Тест работы с существующим правилом сообщества"
echo "=================================================="

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

# Файл для получения всех правил
cat > get_all_rules_query.json << 'EOF'
{
  "query": "query { communityRules(communityID: \"1\") { id title description createdAt } }"
}
EOF

# Файл для получения конкретного правила (ID 3)
cat > get_rule_3_query.json << 'EOF'
{
  "query": "query { communityRule(id: \"3\") { id title description createdAt updatedAt community { id title } } }"
}
EOF

# Файл для обновления правила 3
cat > update_rule_3_query.json << 'EOF'
{
  "query": "mutation { updateCommunityRule(input: { id: \"3\", title: \"Обновленное правило\", description: \"Это правило было обновлено в ходе тестирования\" }) { id title description updatedAt } }"
}
EOF

# Файл для удаления правила 3
cat > delete_rule_3_query.json << 'EOF'
{
  "query": "mutation { deleteCommunityRule(id: \"3\") }"
}
EOF

echo -e "${GREEN}✅ JSON файлы созданы${NC}"

# Получение всех правил
echo -e "\n${YELLOW}3. Получение всех правил сообщества...${NC}"
execute_query_from_file "Получение всех правил" "get_all_rules_query.json"

# Получение конкретного правила
echo -e "\n${YELLOW}4. Получение правила с ID 3...${NC}"
execute_query_from_file "Получение правила 3" "get_rule_3_query.json"

# Попытка обновления правила (ожидается ошибка прав)
echo -e "\n${YELLOW}5. Попытка обновления правила 3...${NC}"
execute_query_from_file "Обновление правила 3" "update_rule_3_query.json"

# Попытка удаления правила (ожидается ошибка прав)
echo -e "\n${YELLOW}6. Попытка удаления правила 3...${NC}"
execute_query_from_file "Удаление правила 3" "delete_rule_3_query.json"

# Очистка временных файлов
echo -e "\n${YELLOW}7. Очистка временных файлов...${NC}"
rm -f get_all_rules_query.json get_rule_3_query.json update_rule_3_query.json delete_rule_3_query.json
echo -e "${GREEN}✅ Временные файлы удалены${NC}"

echo -e "\n${GREEN}✅ Тестирование с существующим правилом завершено${NC}"
echo -e "\n${YELLOW}Выводы:${NC}"
echo -e "${YELLOW}- В базе данных есть правило с ID 3${NC}"
echo -e "${YELLOW}- Получение правил работает без авторизации${NC}"
echo -e "${YELLOW}- Для создания/обновления/удаления нужны права администратора${NC}"
echo -e "${YELLOW}- Проблема с токеном авторизации - нужно настроить JWT${NC}"
