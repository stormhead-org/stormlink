package communityrule

import (
	"context"
	"fmt"
	"stormlink/server/ent"
	"stormlink/server/ent/community"
	"stormlink/server/ent/communityrule"
	"stormlink/server/graphql/models"
	sharedauth "stormlink/shared/auth"
	"strconv"
)

type CommunityRuleUsecase interface {
	CreateCommunityRule(ctx context.Context, input *models.CreateCommunityRuleInput) (*ent.CommunityRule, error)
	UpdateCommunityRule(ctx context.Context, input *models.UpdateCommunityRuleInput) (*ent.CommunityRule, error)
	DeleteCommunityRule(ctx context.Context, id string) (bool, error)
	GetCommunityRule(ctx context.Context, id string) (*ent.CommunityRule, error)
	GetCommunityRules(ctx context.Context, communityID string) ([]*ent.CommunityRule, error)
}

type communityRuleUsecase struct {
	client *ent.Client
}

func NewCommunityRuleUsecase(client *ent.Client) CommunityRuleUsecase {
	return &communityRuleUsecase{client: client}
}

func (uc *communityRuleUsecase) CreateCommunityRule(ctx context.Context, input *models.CreateCommunityRuleInput) (*ent.CommunityRule, error) {
	// Проверяем авторизацию
	userID, err := sharedauth.UserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Конвертируем communityID в int
	communityIDInt, err := strconv.Atoi(input.CommunityID)
	if err != nil {
		return nil, fmt.Errorf("invalid community ID: %w", err)
	}

	// Проверяем права на управление сообществом
	canManage, err := uc.canManageCommunity(ctx, userID, communityIDInt)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canManage {
		return nil, fmt.Errorf("insufficient permissions to manage community rules")
	}

	// Создаем правило
	rule, err := uc.client.CommunityRule.
		Create().
		SetTitle(input.Title).
		SetDescription(input.Description).
		SetCommunityID(communityIDInt).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create community rule: %w", err)
	}

	return rule, nil
}

func (uc *communityRuleUsecase) UpdateCommunityRule(ctx context.Context, input *models.UpdateCommunityRuleInput) (*ent.CommunityRule, error) {
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

	// Получаем существующее правило
	rule, err := uc.client.CommunityRule.
		Query().
		Where(communityrule.IDEQ(ruleIDInt)).
		WithCommunity().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	// Проверяем права на управление сообществом
	canManage, err := uc.canManageCommunity(ctx, userID, *rule.CommunityID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canManage {
		return nil, fmt.Errorf("insufficient permissions to manage community rules")
	}

	// Обновляем правило
	update := uc.client.CommunityRule.UpdateOneID(ruleIDInt)
	
	if input.Title != nil {
		update = update.SetTitle(*input.Title)
	}
	if input.Description != nil {
		update = update.SetDescription(*input.Description)
	}

	updatedRule, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update community rule: %w", err)
	}

	return updatedRule, nil
}

func (uc *communityRuleUsecase) DeleteCommunityRule(ctx context.Context, id string) (bool, error) {
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

	// Получаем правило для проверки прав
	rule, err := uc.client.CommunityRule.
		Query().
		Where(communityrule.IDEQ(ruleIDInt)).
		WithCommunity().
		Only(ctx)
	if err != nil {
		return false, fmt.Errorf("rule not found: %w", err)
	}

	// Проверяем права на управление сообществом
	canManage, err := uc.canManageCommunity(ctx, userID, *rule.CommunityID)
	if err != nil {
		return false, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canManage {
		return false, fmt.Errorf("insufficient permissions to manage community rules")
	}

	// Удаляем правило
	err = uc.client.CommunityRule.DeleteOneID(ruleIDInt).Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to delete community rule: %w", err)
	}

	return true, nil
}

func (uc *communityRuleUsecase) GetCommunityRule(ctx context.Context, id string) (*ent.CommunityRule, error) {
	ruleIDInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid rule ID: %w", err)
	}

	rule, err := uc.client.CommunityRule.
		Query().
		Where(communityrule.IDEQ(ruleIDInt)).
		WithCommunity().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	return rule, nil
}

func (uc *communityRuleUsecase) GetCommunityRules(ctx context.Context, communityID string) ([]*ent.CommunityRule, error) {
	communityIDInt, err := strconv.Atoi(communityID)
	if err != nil {
		return nil, fmt.Errorf("invalid community ID: %w", err)
	}

	rules, err := uc.client.CommunityRule.
		Query().
		Where(communityrule.CommunityIDEQ(communityIDInt)).
		Order(ent.Asc(communityrule.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get community rules: %w", err)
	}

	return rules, nil
}

// canManageCommunity проверяет, может ли пользователь управлять сообществом
func (uc *communityRuleUsecase) canManageCommunity(ctx context.Context, userID, communityID int) (bool, error) {
	// Проверяем, является ли пользователь владельцем сообщества
	comm, err := uc.client.Community.
		Query().
		Where(community.IDEQ(communityID)).
		Only(ctx)
	if err != nil {
		return false, fmt.Errorf("community not found: %w", err)
	}

	if comm.OwnerID == userID {
		return true, nil
	}

	// TODO: Здесь можно добавить проверку на роли модераторов
	// Пока что только владелец может управлять правилами

	return false, nil
}
