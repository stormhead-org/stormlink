package hostrule

import (
	"context"
	"fmt"
	"stormlink/server/ent"
	"stormlink/server/ent/hostrule"
	"stormlink/server/graphql/models"
	sharedauth "stormlink/shared/auth"
	"strconv"
)

type HostRuleUsecase interface {
	CreateHostRule(ctx context.Context, input *models.CreateHostRuleInput) (*ent.HostRule, error)
	UpdateHostRule(ctx context.Context, input *models.UpdateHostRuleInput) (*ent.HostRule, error)
	DeleteHostRule(ctx context.Context, id string) (bool, error)
	GetHostRule(ctx context.Context, id string) (*ent.HostRule, error)
	GetHostRules(ctx context.Context) ([]*ent.HostRule, error)
}

type hostRuleUsecase struct {
	client *ent.Client
}

func NewHostRuleUsecase(client *ent.Client) HostRuleUsecase {
	return &hostRuleUsecase{client: client}
}

func (uc *hostRuleUsecase) CreateHostRule(ctx context.Context, input *models.CreateHostRuleInput) (*ent.HostRule, error) {
	// Проверяем авторизацию
	userID, err := sharedauth.UserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Проверяем права на управление правилами платформы
	canManage, err := uc.canManageHostRules(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canManage {
		return nil, fmt.Errorf("insufficient permissions to manage platform rules")
	}

	// Создаем правило (всегда для хоста с ID = 1)
	rule, err := uc.client.HostRule.
		Create().
		SetTitle(input.Title).
		SetDescription(input.Description).
		SetHostID(1). // Фиксированный ID хоста
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create host rule: %w", err)
	}

	return rule, nil
}

func (uc *hostRuleUsecase) UpdateHostRule(ctx context.Context, input *models.UpdateHostRuleInput) (*ent.HostRule, error) {
	// Проверяем авторизацию
	userID, err := sharedauth.UserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Конвертируем ID в int
	ruleIDInt, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid rule ID: %w", err)
	}

	// Проверяем права на управление правилами платформы
	canManage, err := uc.canManageHostRules(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canManage {
		return nil, fmt.Errorf("insufficient permissions to manage platform rules")
	}

	// Обновляем правило
	update := uc.client.HostRule.UpdateOneID(ruleIDInt)
	
	if input.Title != nil {
		update = update.SetTitle(*input.Title)
	}
	if input.Description != nil {
		update = update.SetDescription(*input.Description)
	}

	updatedRule, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update host rule: %w", err)
	}

	return updatedRule, nil
}

func (uc *hostRuleUsecase) DeleteHostRule(ctx context.Context, id string) (bool, error) {
	// Проверяем авторизацию
	userID, err := sharedauth.UserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("unauthorized: %w", err)
	}

	// Конвертируем ID в int
	ruleIDInt, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("invalid rule ID: %w", err)
	}

	// Проверяем права на управление правилами платформы
	canManage, err := uc.canManageHostRules(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canManage {
		return false, fmt.Errorf("insufficient permissions to manage platform rules")
	}

	// Удаляем правило
	err = uc.client.HostRule.DeleteOneID(ruleIDInt).Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to delete host rule: %w", err)
	}

	return true, nil
}

func (uc *hostRuleUsecase) GetHostRule(ctx context.Context, id string) (*ent.HostRule, error) {
	ruleIDInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid rule ID: %w", err)
	}

	rule, err := uc.client.HostRule.
		Query().
		Where(hostrule.IDEQ(ruleIDInt)).
		WithHost().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	return rule, nil
}

func (uc *hostRuleUsecase) GetHostRules(ctx context.Context) ([]*ent.HostRule, error) {
	// Получаем все правила для хоста с ID = 1
	rules, err := uc.client.HostRule.
		Query().
		Where(hostrule.HostIDEQ(1)).
		Order(ent.Asc(hostrule.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host rules: %w", err)
	}

	return rules, nil
}

// canManageHostRules проверяет, может ли пользователь управлять правилами платформы
func (uc *hostRuleUsecase) canManageHostRules(ctx context.Context, userID int) (bool, error) {
	// 1. Проверяем, является ли пользователь владельцем платформы
	host, err := uc.client.Host.Get(ctx, 1)
	if err != nil {
		return false, fmt.Errorf("host not found: %w", err)
	}
	
	if host.OwnerID != nil && *host.OwnerID == userID {
		return true, nil
	}
	
	// 2. Проверяем роли пользователя на платформе
	// TODO: Добавить проверку ролей когда будет реализована система ролей платформы
	// Пока что только владелец может управлять правилами
	
	return false, nil
}
