-- Миграция: добавление роли "@everyone" для существующих сообществ (ИСПРАВЛЕННАЯ)
-- Этот скрипт нужно выполнить один раз для существующих сообществ

-- 1. Создаем роль "@everyone" для каждого существующего сообщества
INSERT INTO roles (title, color, community_roles_management, community_user_ban, community_user_mute, community_delete_post, community_delete_comments, community_remove_post_from_publication, community_id, created_at, updated_at)
SELECT 
    '@everyone' as title,
    '#99AAB5' as color,
    false as community_roles_management,
    false as community_user_ban,
    false as community_user_mute,
    false as community_delete_post,
    false as community_delete_comments,
    false as community_remove_post_from_publication,
    c.id as community_id,
    NOW() as created_at,
    NOW() as updated_at
FROM communities c
WHERE NOT EXISTS (
    SELECT 1 FROM roles r 
    WHERE r.community_id = c.id AND r.title = '@everyone'
);

-- 2. Назначаем роль "@everyone" владельцам сообществ
INSERT INTO user_communities_roles (user_id, role_id)
SELECT 
    c.owner_id as user_id,
    r.id as role_id
FROM communities c
JOIN roles r ON r.community_id = c.id AND r.title = '@everyone'
WHERE NOT EXISTS (
    SELECT 1 FROM user_communities_roles ucr 
    WHERE ucr.user_id = c.owner_id AND ucr.role_id = r.id
);

-- 3. Назначаем роль "@everyone" всем участникам сообществ (подписчикам)
INSERT INTO user_communities_roles (user_id, role_id)
SELECT 
    cf.user_id as user_id,
    r.id as role_id
FROM community_follows cf
JOIN roles r ON r.community_id = cf.community_id AND r.title = '@everyone'
WHERE NOT EXISTS (
    SELECT 1 FROM user_communities_roles ucr 
    WHERE ucr.user_id = cf.user_id AND ucr.role_id = r.id
);

-- Проверка результатов
SELECT 
    c.title as community_name,
    c.id as community_id,
    COUNT(r.id) as roles_count,
    COUNT(ucr.user_id) as users_with_everyone_role
FROM communities c
LEFT JOIN roles r ON r.community_id = c.id
LEFT JOIN user_communities_roles ucr ON ucr.role_id = r.id AND r.title = '@everyone'
GROUP BY c.id, c.title
ORDER BY c.id;
