#!/bin/bash

echo "🧪 Тестирование Rate Limiting"
echo "============================="

BASE_URL="http://localhost:8080"

echo "Отправляем 20 быстрых запросов подряд..."

for i in {1..20}; do
    echo -n "Запрос #$i: "
    RESPONSE=$(curl -s -w "%{http_code}" -X POST $BASE_URL/query \
      -H "Content-Type: application/json" \
      -d '{"query":"query { __schema { types { name } } }"}' 2>/dev/null)
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    
    if [ "$HTTP_CODE" = "429" ]; then
        echo "🚫 RATE LIMITED (429) - Rate limiting работает!"
        echo "Сработал на запросе #$i"
        break
    elif [ "$HTTP_CODE" = "200" ]; then
        echo "✅ OK (200)"
    else
        echo "❓ Неожиданный код: $HTTP_CODE"
    fi
    
    # Небольшая задержка между запросами
    sleep 0.1
done

echo ""
echo "Тест завершен. Если вы не увидели 429 ошибку, rate limiting может не работать."
