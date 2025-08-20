#!/bin/bash

echo "🧪 Простой тест Rate Limiting"
echo "============================="

BASE_URL="http://localhost:8080"

echo "Отправляем 10 быстрых запросов..."

for i in {1..10}; do
    echo -n "Запрос #$i: "
    
    # Используем curl с правильным извлечением HTTP кода
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE_URL/query \
      -H "Content-Type: application/json" \
      -d '{"query":"query { __schema { types { name } } }"}' 2>/dev/null)
    
    echo "HTTP $HTTP_CODE"
    
    if [ "$HTTP_CODE" = "429" ]; then
        echo "🚫 RATE LIMITED! Rate limiting работает!"
        break
    fi
    
    # Очень маленькая задержка
    sleep 0.05
done

echo ""
echo "Тест завершен."
