-- Seed data for Bluesky Automation Platform
-- This file contains sample data for development and testing

-- Insert sample proxies
INSERT INTO proxies (name, type, host, port, username, password, status) VALUES
('Proxy Server 1', 'http', 'proxy1.example.com', 8080, 'user1', 'pass1', 'active'),
('Proxy Server 2', 'http', 'proxy2.example.com', 8080, 'user2', 'pass2', 'active'),
('Proxy Server 3', 'socks5', 'proxy3.example.com', 1080, 'user3', 'pass3', 'active'),
('Proxy Server 4', 'http', 'proxy4.example.com', 3128, 'user4', 'pass4', 'inactive'),
('Proxy Server 5', 'socks5', 'proxy5.example.com', 1080, 'user5', 'pass5', 'active');

-- Insert sample accounts (Note: These are dummy accounts for testing)
INSERT INTO accounts (handle, password, host, status, proxy_id) VALUES
('testuser1.bsky.social', 'dummy_password_1', 'https://bsky.social', 'active', 1),
('testuser2.bsky.social', 'dummy_password_2', 'https://bsky.social', 'active', 2),
('testuser3.bsky.social', 'dummy_password_3', 'https://bsky.social', 'active', 3),
('testuser4.bsky.social', 'dummy_password_4', 'https://bsky.social', 'inactive', 4),
('testuser5.bsky.social', 'dummy_password_5', 'https://bsky.social', 'active', 5);

-- Insert sample strategies
INSERT INTO strategies (name, description, type, config, schedule, status) VALUES
('自動關注科技用戶', '自動關注發布科技相關內容的用戶', 'follow', 
'{
  "keywords": ["AI", "機器學習", "科技", "程式設計", "開發"],
  "daily_limit": 50,
  "delay_range": [300, 600],
  "min_followers": 100,
  "max_followers": 10000,
  "exclude_keywords": ["廣告", "推銷"]
}', '0 */2 * * *', 'active'),

('定時發布內容', '定時發布預設的內容', 'post', 
'{
  "content_templates": [
    "今天是美好的一天！ #正能量",
    "分享一些有趣的科技新聞 #科技",
    "學習新技術的心得分享 #學習"
  ],
  "daily_limit": 5,
  "delay_range": [1800, 3600],
  "include_hashtags": true
}', '0 9,12,15,18 * * *', 'active'),

('智能點讚策略', '對相關內容進行智能點讚', 'like', 
'{
  "keywords": ["科技", "AI", "程式設計"],
  "daily_limit": 100,
  "delay_range": [60, 180],
  "like_probability": 0.7,
  "avoid_spam": true
}', '*/15 * * * *', 'active'),

('內容監控策略', '監控特定關鍵詞的內容', 'monitor', 
'{
  "keywords": ["競爭對手", "行業動態", "新產品"],
  "notification_webhook": "https://hooks.slack.com/services/...",
  "save_to_database": true,
  "analysis_enabled": true
}', '*/5 * * * *', 'active'),

('增長策略', '綜合增長策略', 'growth', 
'{
  "follow_back_probability": 0.8,
  "engagement_rate_target": 0.05,
  "content_quality_threshold": 0.7,
  "daily_activity_target": 20
}', '0 8 * * *', 'paused');

-- Insert account-strategy associations
INSERT INTO account_strategies (account_id, strategy_id, config, status) VALUES
(1, 1, '{"daily_limit": 30}', 'active'),
(1, 2, '{"daily_limit": 3}', 'active'),
(1, 3, '{"daily_limit": 80}', 'active'),
(2, 1, '{"daily_limit": 40}', 'active'),
(2, 4, '{}', 'active'),
(3, 2, '{"daily_limit": 5}', 'active'),
(3, 3, '{"daily_limit": 120}', 'active'),
(3, 5, '{}', 'active'),
(4, 4, '{}', 'inactive'),
(5, 1, '{"daily_limit": 25}', 'active'),
(5, 2, '{"daily_limit": 4}', 'active'),
(5, 3, '{"daily_limit": 90}', 'active');

-- Insert sample tasks
INSERT INTO tasks (account_id, strategy_id, type, payload, status, priority, scheduled_at) VALUES
(1, 1, 'follow_user', '{"target_handle": "techuser1.bsky.social", "reason": "keyword_match"}', 'completed', 5, NOW() - INTERVAL '1 hour'),
(1, 2, 'create_post', '{"content": "今天是美好的一天！ #正能量", "template_id": 1}', 'completed', 3, NOW() - INTERVAL '2 hours'),
(2, 1, 'follow_user', '{"target_handle": "aiexpert.bsky.social", "reason": "keyword_match"}', 'pending', 5, NOW() + INTERVAL '30 minutes'),
(2, 4, 'monitor_keyword', '{"keyword": "競爭對手", "search_limit": 50}', 'running', 7, NOW()),
(3, 3, 'like_post', '{"post_uri": "at://did:plc:example/app.bsky.feed.post/123", "reason": "keyword_match"}', 'completed', 4, NOW() - INTERVAL '15 minutes'),
(3, 5, 'analyze_growth', '{"metrics": ["followers", "engagement", "reach"]}', 'pending', 6, NOW() + INTERVAL '1 hour');

-- Insert sample metrics
INSERT INTO metrics (account_id, strategy_id, metric_type, metric_name, metric_value, metric_data, timestamp) VALUES
(1, 1, 'performance', 'follows_completed', 25, '{"success_rate": 0.95}', NOW() - INTERVAL '1 day'),
(1, 2, 'performance', 'posts_created', 3, '{"engagement_rate": 0.08}', NOW() - INTERVAL '1 day'),
(1, 3, 'performance', 'likes_given', 75, '{"success_rate": 0.98}', NOW() - INTERVAL '1 day'),
(2, 1, 'performance', 'follows_completed', 30, '{"success_rate": 0.92}', NOW() - INTERVAL '1 day'),
(2, 4, 'performance', 'keywords_monitored', 150, '{"matches_found": 12}', NOW() - INTERVAL '1 day'),
(3, 2, 'performance', 'posts_created', 4, '{"engagement_rate": 0.12}', NOW() - INTERVAL '1 day'),
(3, 3, 'performance', 'likes_given', 95, '{"success_rate": 0.97}', NOW() - INTERVAL '1 day'),
(3, 5, 'performance', 'growth_analysis', 1, '{"follower_growth": 15, "engagement_growth": 0.02}', NOW() - INTERVAL '1 day');

-- Insert sample audit logs
INSERT INTO audit_logs (entity_type, entity_id, action, new_values, user_id, ip_address) VALUES
('accounts', 1, 'create', '{"handle": "testuser1.bsky.social", "status": "active"}', 'admin', '127.0.0.1'),
('strategies', 1, 'create', '{"name": "自動關注科技用戶", "type": "follow"}', 'admin', '127.0.0.1'),
('tasks', 1, 'execute', '{"status": "completed", "result": "success"}', 'system', '127.0.0.1'),
('accounts', 2, 'update', '{"status": "active"}', 'admin', '127.0.0.1'),
('strategies', 2, 'update', '{"status": "active"}', 'admin', '127.0.0.1');

-- Update system settings with more realistic values
UPDATE system_settings SET value = '10' WHERE key = 'max_concurrent_tasks_per_account';
UPDATE system_settings SET value = '600' WHERE key = 'default_task_timeout';
UPDATE system_settings SET value = '180' WHERE key = 'proxy_health_check_interval';
UPDATE system_settings SET value = '30' WHERE key = 'rate_limit_requests_per_minute';
UPDATE system_settings SET value = '30' WHERE key = 'strategy_execution_interval';

-- Insert additional system settings for development
INSERT INTO system_settings (key, value, description) VALUES
('debug_mode', 'true', 'Enable debug mode for development'),
('log_level', 'debug', 'Logging level for the application'),
('enable_webhooks', 'false', 'Enable webhook notifications'),
('default_proxy_timeout', '30', 'Default proxy timeout in seconds'),
('max_workers', '20', 'Maximum number of worker processes'),
('task_queue_size', '1000', 'Maximum size of task queue'),
('enable_metrics_collection', 'true', 'Enable metrics collection'),
('metrics_retention_days', '90', 'Days to retain metrics data');

-- Create some sample task dependencies
INSERT INTO task_dependencies (task_id, depends_on_task_id) VALUES
(6, 5), -- Growth analysis depends on like task completion
(3, 1); -- Follow task depends on previous follow completion

COMMIT;
