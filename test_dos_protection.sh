#!/bin/bash

echo "💥 Тестирование DoS защиты"
echo "=========================="

BASE_URL="http://localhost:8080"

echo "Отправляем 50 очень быстрых запросов (имитация DoS атаки)..."

# Счетчики
success_count=0
rate_limited_count=0
error_count=0

for i in {1..50}; do
    RESPONSE=$(curl -s -w "%{http_code}" -X POST $BASE_URL/query \
      -H "Content-Type: application/json" \
      -d '{"query":"query { __schema { types { name } } }"}' 2>/dev/null)
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    
    case $HTTP_CODE in
        200)
            ((success_count++))
            echo -n "."
            ;;
        429)
            ((rate_limited_count++))
            echo -n "🚫"
            ;;
        *)
            ((error_count++))
            echo -n "❌"
            ;;
    esac
    
    # Очень маленькая задержка для имитации DoS
    sleep 0.01
done

echo ""
echo ""
echo "📊 Результаты DoS теста:"
echo "========================"
echo "✅ Успешных запросов: $success_count"
echo "🚫 Rate limited: $rate_limited_count"
echo "❌ Ошибок: $error_count"
echo ""

if [ $rate_limited_count -gt 0 ]; then
    echo "🎉 DoS защита работает! Rate limiting сработал $rate_limited_count раз."
else
    echo "⚠️  DoS защита может быть недостаточной. Rate limiting не сработал."
fi

echo ""
echo "Ожидаемое поведение:"
echo "- Первые 5-10 запросов должны пройти (200)"
echo "- Остальные должны быть заблокированы (429)"
echo "- Сервер должен остаться стабильным"
