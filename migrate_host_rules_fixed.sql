-- Миграция для исправления схемы HostRule
-- Переименовываем поля и добавляем правильные связи

-- 1. Создаем временную таблицу с новой структурой
CREATE TABLE IF NOT EXISTS host_rules_new (
    id SERIAL PRIMARY KEY,
    host_id INTEGER,
    title VARCHAR(255),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Копируем данные из старой таблицы в новую
INSERT INTO host_rules_new (id, host_id, title, description, created_at, updated_at)
SELECT 
    id,
    rule_id as host_id,
    name_rule as title,
    description_rule as description,
    created_at,
    updated_at
FROM host_rules;

-- 3. Удаляем старую таблицу
DROP TABLE IF EXISTS host_rules;

-- 4. Переименовываем новую таблицу
ALTER TABLE host_rules_new RENAME TO host_rules;

-- 5. Добавляем индексы
CREATE INDEX IF NOT EXISTS idx_host_rules_host_id ON host_rules(host_id);
CREATE INDEX IF NOT EXISTS idx_host_rules_created_at ON host_rules(created_at);

-- 6. Добавляем внешний ключ на хост (всегда ID = 1)
ALTER TABLE host_rules 
ADD CONSTRAINT fk_host_rules_host 
FOREIGN KEY (host_id) REFERENCES hosts(id) ON DELETE CASCADE;

-- 7. Устанавливаем host_id = 1 для всех существующих правил
UPDATE host_rules SET host_id = 1 WHERE host_id IS NULL;

-- 8. Делаем host_id обязательным
ALTER TABLE host_rules ALTER COLUMN host_id SET NOT NULL;

-- Комментарии к таблице
COMMENT ON TABLE host_rules IS 'Правила платформы';
COMMENT ON COLUMN host_rules.id IS 'Уникальный идентификатор правила';
COMMENT ON COLUMN host_rules.host_id IS 'ID хоста (всегда 1)';
COMMENT ON COLUMN host_rules.title IS 'Название правила';
COMMENT ON COLUMN host_rules.description IS 'Описание правила';
COMMENT ON COLUMN host_rules.created_at IS 'Дата создания';
COMMENT ON COLUMN host_rules.updated_at IS 'Дата обновления';
