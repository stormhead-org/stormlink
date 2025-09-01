package testhelper

import (
	"context"
	"testing"
	"time"

	"stormlink/server/ent"
	"stormlink/server/ent/enttest"
	"stormlink/tests/testcontainers"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// PostgresTestHelper упрощает настройку PostgreSQL тестов
type PostgresTestHelper struct {
	containers *testcontainers.TestContainers
	client     *ent.Client
	ctx        context.Context
}

// NewPostgresTestHelper создает новый helper для PostgreSQL тестов
func NewPostgresTestHelper(t *testing.T) *PostgresTestHelper {
	ctx := context.Background()

	// Настраиваем TestContainers
	containers, err := testcontainers.Setup(ctx)
	require.NoError(t, err)

	// Создаем Ent клиента с PostgreSQL
	client := enttest.Open(t, "postgres", containers.GetPostgresDSN())

	return &PostgresTestHelper{
		containers: containers,
		client:     client,
		ctx:        ctx,
	}
}

// GetClient возвращает Ent клиента
func (h *PostgresTestHelper) GetClient() *ent.Client {
	return h.client
}

// GetContext возвращает контекст
func (h *PostgresTestHelper) GetContext() context.Context {
	return h.ctx
}

// GetContainers возвращает TestContainers для доступа к DSN и URL
func (h *PostgresTestHelper) GetContainers() *testcontainers.TestContainers {
	return h.containers
}

// CleanDatabase очищает всю базу данных (для SetupTest/TearDownTest)
func (h *PostgresTestHelper) CleanDatabase(t *testing.T) {
	// Удаляем данные в правильном порядке (от дочерних к родительским таблицам)
	_, err := h.client.CommentLike.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.PostLike.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.Comment.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.Post.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.Bookmark.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.UserFollow.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.CommunityFollow.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.CommunityModerator.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.CommunityUserBan.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.CommunityUserMute.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.HostUserBan.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.HostUserMute.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.HostCommunityBan.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.HostCommunityMute.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.Media.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.CommunityRule.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.HostRule.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.ProfileTableInfoItem.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.HostSidebarNavigationItem.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.HostSidebarNavigation.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.HostSocialNavigation.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.EmailVerification.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.Community.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.User.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.Role.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.HostRole.Delete().Exec(h.ctx)
	require.NoError(t, err)

	_, err = h.client.Host.Delete().Exec(h.ctx)
	require.NoError(t, err)
}

// Cleanup завершает все контейнеры и закрывает соединения
func (h *PostgresTestHelper) Cleanup() {
	if h.client != nil {
		h.client.Close()
	}
	if h.containers != nil {
		h.containers.Cleanup()
	}
}

// SetupUniqueTest создает уникальное тестовое окружение для простых тестов
// Используется когда нужна быстрая настройка без полного TestContainers
func SetupUniqueTest(t *testing.T, testName string) *PostgresTestHelper {
	helper := NewPostgresTestHelper(t)
	helper.CleanDatabase(t)
	return helper
}

// WithTransaction выполняет функцию в транзакции и откатывает её
func (h *PostgresTestHelper) WithTransaction(t *testing.T, fn func(*ent.Tx)) {
	tx, err := h.client.Tx(h.ctx)
	require.NoError(t, err)

	defer func() {
		// Всегда откатываем транзакцию для изоляции тестов
		err := tx.Rollback()
		require.NoError(t, err)
	}()

	fn(tx)
}

// WaitForDatabase ждет готовности базы данных
func (h *PostgresTestHelper) WaitForDatabase(t *testing.T) {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		err := h.client.Schema.Create(h.ctx)
		if err == nil {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatal("База данных не готова после 30 секунд ожидания")
}
