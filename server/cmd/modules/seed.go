package modules

import (
	"context"
	"log"
	"stormlink/server/ent/hostrole"
	"stormlink/server/ent/hostsidebarnavigation"

	"stormlink/server/ent"
	"stormlink/server/ent/host"
)

func Seed(client *ent.Client) error {
	ctx := context.Background()

	// Проверка: сидились ли роли?
	roleOwnerExists, err := client.HostRole.Query().Where(hostrole.TitleEQ("owner")).Exist(ctx)
	if err != nil {
		log.Println("✅ Роль host owner уже существует...")
		return err
	}
	if !roleOwnerExists {
		log.Println("🌱 Сидим роль host owner...")
		if _, err := client.HostRole.Create().
			SetTitle("owner").
			SetColor("#99AAB5").
			SetCommunityRolesManagement(true).
			SetHostUserBan(true).
			SetHostUserMute(true).
			SetHostCommunityDeletePost(true).
			SetHostCommunityRemovePostFromPublication(true).
			SetHostCommunityDeleteComments(true).
			Save(ctx); err != nil {
			return err
		}
		log.Printf("✅ Роль host owner создана")
	}
	roleEveryoneExists, err := client.HostRole.Query().Where(hostrole.TitleEQ("@everyone")).Exist(ctx)
	if err != nil {
		log.Println("✅ Роль host @everyone уже существует...")
		return err
	}
	if !roleEveryoneExists {
		log.Println("🌱 Сидим роль host @everyone...")
		if _, err := client.HostRole.Create().
			SetTitle("@everyone").
			SetColor("#99AAB5").
			Save(ctx); err != nil {
			return err
		}
		log.Printf("✅ Роль host @everyone создана")
	}

	// Проверка: сидился ли дефолтный host
	hostExists, err := client.Host.Query().Where(host.IDEQ(1)).Exist(ctx)
	if err != nil {
		log.Println("✅ Таблица настроек host уже существует...")
		return err
	}
	if !hostExists {
		log.Println("🌱 Сидим настройки host...")
		if _, err := client.Host.Create().
			SetTitle("Stormic").
			SetSlogan("код, GitHub и ты").
			SetFirstSettings(true).
			SetDescription("Социальная платформа с открытым исходным кодом").
			Save(ctx); err != nil {
			return err
		}
		log.Println("✅ Таблица настроек host создана")
	}

	// Проверка: сидился ли дефолтный HostSidebarNavigation
	hostSidebarNavigationExists, err := client.HostSidebarNavigation.Query().Where(hostsidebarnavigation.IDEQ(1)).Exist(ctx)
	if err != nil {
		log.Println("✅ Таблица навигации HostSidebarNavigation уже существует...")
		return err
	}
	if !hostSidebarNavigationExists {
		log.Println("🌱 Сидим навигацию HostSidebarNavigation...")
		if _, err := client.HostSidebarNavigation.Create().
			Save(ctx); err != nil {
			return err
		}
		log.Println("✅ Таблица навигации HostSidebarNavigation создана")
	}

	return nil
}
