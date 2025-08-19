-- Проверка сообществ
SELECT id, title, owner_id FROM communities WHERE id = 1;

-- Проверка ролей для сообщества 1
SELECT id, title, community_id, community_roles_management, community_user_ban, community_user_mute, community_delete_post, community_delete_comments, community_remove_post_from_publication 
FROM roles 
WHERE community_id = 1;

-- Проверка пользователей
SELECT id, name, slug FROM users WHERE id = 1;

-- Проверка связи пользователей с ролями
SELECT u.id as user_id, u.name, r.id as role_id, r.title as role_title
FROM users u
JOIN role_users ru ON u.id = ru.user_id
JOIN roles r ON ru.role_id = r.id
WHERE r.community_id = 1;
