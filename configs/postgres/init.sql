-- Bluesky Automation Platform Database Schema
-- Created: 2025-01-02

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum types
CREATE TYPE account_status AS ENUM ('active', 'inactive', 'suspended', 'error');
CREATE TYPE proxy_type AS ENUM ('http', 'socks5');
CREATE TYPE proxy_status AS ENUM ('active', 'inactive', 'error');
CREATE TYPE strategy_type AS ENUM ('post', 'follow', 'like', 'repost', 'monitor', 'growth');
CREATE TYPE strategy_status AS ENUM ('active', 'inactive', 'paused');
CREATE TYPE task_status AS ENUM ('pending', 'running', 'completed', 'failed', 'cancelled');

-- Proxies table
CREATE TABLE proxies (
    id SERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    type proxy_type NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL CHECK (port > 0 AND port <= 65535),
    username VARCHAR(255),
    password VARCHAR(255),
    status proxy_status DEFAULT 'active',
    health_check_url VARCHAR(500),
    last_health_check TIMESTAMP,
    health_check_success BOOLEAN DEFAULT true,
    response_time_ms INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Accounts table
CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE NOT NULL,
    handle VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    host VARCHAR(255) DEFAULT 'https://bsky.social',
    bgs VARCHAR(255) DEFAULT 'https://bsky.network',
    status account_status DEFAULT 'active',
    proxy_id INTEGER REFERENCES proxies(id) ON DELETE SET NULL,
    did VARCHAR(255),
    access_jwt TEXT,
    refresh_jwt TEXT,
    last_login TIMESTAMP,
    last_activity TIMESTAMP,
    error_count INTEGER DEFAULT 0,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Strategies table
CREATE TABLE strategies (
    id SERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type strategy_type NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    schedule VARCHAR(255), -- cron expression
    status strategy_status DEFAULT 'active',
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    max_concurrent_tasks INTEGER DEFAULT 1,
    retry_count INTEGER DEFAULT 3,
    timeout_seconds INTEGER DEFAULT 300,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Account strategies association table
CREATE TABLE account_strategies (
    id SERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE NOT NULL,
    account_id INTEGER REFERENCES accounts(id) ON DELETE CASCADE,
    strategy_id INTEGER REFERENCES strategies(id) ON DELETE CASCADE,
    config JSONB DEFAULT '{}', -- account-specific configuration overrides
    status strategy_status DEFAULT 'active',
    last_executed TIMESTAMP,
    next_execution TIMESTAMP,
    execution_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(account_id, strategy_id)
);

-- Tasks table
CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE NOT NULL,
    account_id INTEGER REFERENCES accounts(id) ON DELETE CASCADE,
    strategy_id INTEGER REFERENCES strategies(id) ON DELETE CASCADE,
    account_strategy_id INTEGER REFERENCES account_strategies(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    status task_status DEFAULT 'pending',
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    timeout_seconds INTEGER DEFAULT 300,
    scheduled_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    worker_id VARCHAR(255),
    error_message TEXT,
    result JSONB DEFAULT '{}',
    execution_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Task dependencies table (for complex workflows)
CREATE TABLE task_dependencies (
    id SERIAL PRIMARY KEY,
    task_id INTEGER REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id INTEGER REFERENCES tasks(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(task_id, depends_on_task_id)
);

-- Metrics table for monitoring and analytics
CREATE TABLE metrics (
    id SERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE NOT NULL,
    account_id INTEGER REFERENCES accounts(id) ON DELETE CASCADE,
    strategy_id INTEGER REFERENCES strategies(id) ON DELETE CASCADE,
    metric_type VARCHAR(100) NOT NULL,
    metric_name VARCHAR(255) NOT NULL,
    metric_value NUMERIC,
    metric_data JSONB DEFAULT '{}',
    timestamp TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Audit logs table
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE NOT NULL,
    entity_type VARCHAR(100) NOT NULL, -- accounts, strategies, tasks, etc.
    entity_id INTEGER NOT NULL,
    action VARCHAR(100) NOT NULL, -- create, update, delete, execute
    old_values JSONB,
    new_values JSONB,
    user_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- System settings table
CREATE TABLE system_settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX idx_accounts_handle ON accounts(handle);
CREATE INDEX idx_accounts_status ON accounts(status);
CREATE INDEX idx_accounts_proxy_id ON accounts(proxy_id);
CREATE INDEX idx_accounts_last_activity ON accounts(last_activity);

CREATE INDEX idx_proxies_status ON proxies(status);
CREATE INDEX idx_proxies_type ON proxies(type);
CREATE INDEX idx_proxies_health ON proxies(health_check_success);

CREATE INDEX idx_strategies_type ON strategies(type);
CREATE INDEX idx_strategies_status ON strategies(status);
CREATE INDEX idx_strategies_priority ON strategies(priority);

CREATE INDEX idx_account_strategies_account_id ON account_strategies(account_id);
CREATE INDEX idx_account_strategies_strategy_id ON account_strategies(strategy_id);
CREATE INDEX idx_account_strategies_status ON account_strategies(status);
CREATE INDEX idx_account_strategies_next_execution ON account_strategies(next_execution);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_account_id ON tasks(account_id);
CREATE INDEX idx_tasks_strategy_id ON tasks(strategy_id);
CREATE INDEX idx_tasks_scheduled_at ON tasks(scheduled_at);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_worker_id ON tasks(worker_id);

CREATE INDEX idx_metrics_account_id ON metrics(account_id);
CREATE INDEX idx_metrics_strategy_id ON metrics(strategy_id);
CREATE INDEX idx_metrics_type ON metrics(metric_type);
CREATE INDEX idx_metrics_timestamp ON metrics(timestamp);

CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers to relevant tables
CREATE TRIGGER update_accounts_updated_at BEFORE UPDATE ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_proxies_updated_at BEFORE UPDATE ON proxies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_strategies_updated_at BEFORE UPDATE ON strategies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_account_strategies_updated_at BEFORE UPDATE ON account_strategies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tasks_updated_at BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_settings_updated_at BEFORE UPDATE ON system_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default system settings
INSERT INTO system_settings (key, value, description) VALUES
('max_concurrent_tasks_per_account', '5', 'Maximum number of concurrent tasks per account'),
('default_task_timeout', '300', 'Default task timeout in seconds'),
('proxy_health_check_interval', '300', 'Proxy health check interval in seconds'),
('rate_limit_requests_per_minute', '60', 'Default rate limit per account per minute'),
('strategy_execution_interval', '60', 'Strategy execution check interval in seconds'),
('max_retry_attempts', '3', 'Maximum retry attempts for failed tasks'),
('cleanup_completed_tasks_days', '7', 'Days to keep completed tasks before cleanup'),
('cleanup_metrics_days', '30', 'Days to keep metrics data before cleanup');

-- Create views for common queries
CREATE VIEW active_accounts AS
SELECT a.*, p.name as proxy_name, p.host as proxy_host, p.port as proxy_port
FROM accounts a
LEFT JOIN proxies p ON a.proxy_id = p.id
WHERE a.status = 'active';

CREATE VIEW pending_tasks AS
SELECT t.*, a.handle as account_handle, s.name as strategy_name
FROM tasks t
JOIN accounts a ON t.account_id = a.id
JOIN strategies s ON t.strategy_id = s.id
WHERE t.status = 'pending'
ORDER BY t.priority DESC, t.scheduled_at ASC;

CREATE VIEW strategy_performance AS
SELECT 
    s.id,
    s.name,
    s.type,
    COUNT(t.id) as total_tasks,
    COUNT(CASE WHEN t.status = 'completed' THEN 1 END) as completed_tasks,
    COUNT(CASE WHEN t.status = 'failed' THEN 1 END) as failed_tasks,
    AVG(t.execution_time_ms) as avg_execution_time_ms,
    MAX(t.completed_at) as last_execution
FROM strategies s
LEFT JOIN tasks t ON s.id = t.strategy_id
GROUP BY s.id, s.name, s.type;

-- Grant permissions (adjust as needed for your setup)
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO bsky_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO bsky_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO bsky_user;
