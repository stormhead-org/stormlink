-- Миграция для создания таблицы community_rules
-- Выполнить: psql -h localhost -U postgres -d stormlink -f migrate_community_rules.sql

-- Создание таблицы community_rules
CREATE TABLE IF NOT EXISTS community_rules (
    id SERIAL PRIMARY KEY,
    community_id INTEGER,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Создание индексов
CREATE INDEX IF NOT EXISTS idx_community_rules_community_id ON community_rules(community_id);
CREATE INDEX IF NOT EXISTS idx_community_rules_created_at ON community_rules(created_at);

-- Добавление внешнего ключа
ALTER TABLE community_rules 
ADD CONSTRAINT fk_community_rules_community_id 
FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE CASCADE;

-- Комментарии к таблице
COMMENT ON TABLE community_rules IS 'Правила сообществ';
COMMENT ON COLUMN community_rules.id IS 'Уникальный идентификатор правила';
COMMENT ON COLUMN community_rules.community_id IS 'ID сообщества, к которому относится правило';
COMMENT ON COLUMN community_rules.title IS 'Название правила';
COMMENT ON COLUMN community_rules.description IS 'Описание правила';
COMMENT ON COLUMN community_rules.created_at IS 'Дата создания';
COMMENT ON COLUMN community_rules.updated_at IS 'Дата последнего обновления';
