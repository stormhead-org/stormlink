package main

import (
	"context"
	"fmt"
	"log"

	"stormlink/server/ent"

	_ "github.com/lib/pq"
)

func main() {
	// Подключение к базе данных
	client, err := ent.Open("postgres", "host=147.45.244.86 port=5442 user=stormic password=ojXJm6yYBfD87yFWFQq40Dfc dbname=stormic sslmode=disable")
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer client.Close()

	// Проверяем подключение
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Printf("Ошибка создания схемы: %v", err)
	}

	// Пробуем создать тестовое правило
	fmt.Println("🔍 Тестирование создания правила...")
	
	rule, err := client.CommunityRule.
		Create().
		SetTitle("Тестовое правило").
		SetDescription("Описание тестового правила").
		SetCommunityID(1).
		Save(context.Background())
	
	if err != nil {
		fmt.Printf("❌ Ошибка создания правила: %v\n", err)
		
		// Проверяем, существует ли таблица
		fmt.Println("🔍 Проверка существования таблицы community_rules...")
		_, err := client.CommunityRule.Query().Limit(1).All(context.Background())
		if err != nil {
			fmt.Printf("❌ Таблица community_rules не существует или недоступна: %v\n", err)
			fmt.Println("💡 Необходимо выполнить миграцию: psql -h localhost -U postgres -d stormlink -f migrate_community_rules.sql")
		} else {
			fmt.Println("✅ Таблица community_rules существует")
		}
	} else {
		fmt.Printf("✅ Правило создано успешно: ID=%d, Title=%s\n", rule.ID, rule.Title)
		
		// Удаляем тестовое правило
		client.CommunityRule.DeleteOne(rule).Exec(context.Background())
		fmt.Println("🗑️ Тестовое правило удалено")
	}

	// Пробуем получить правила
	fmt.Println("🔍 Тестирование получения правил...")
	rules, err := client.CommunityRule.Query().All(context.Background())
	if err != nil {
		fmt.Printf("❌ Ошибка получения правил: %v\n", err)
	} else {
		fmt.Printf("✅ Получено правил: %d\n", len(rules))
		for _, r := range rules {
			fmt.Printf("  - ID: %d, Title: %s\n", r.ID, r.Title)
		}
	}
}
