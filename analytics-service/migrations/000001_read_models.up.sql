CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS events_raw (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    project_id UUID,
    task_id UUID,
    user_id UUID,
    department_id UUID,
    actor_user_id UUID,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS department_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    department_id UUID NOT NULL,
    department_name VARCHAR(255),
    date DATE NOT NULL,
    wip_count INTEGER DEFAULT 0,
    completed_tasks INTEGER DEFAULT 0,
    overdue_tasks INTEGER DEFAULT 0,
    total_tasks INTEGER DEFAULT 0,
    avg_cycle_time DECIMAL(10,2) DEFAULT 0,
    productivity_rate DECIMAL(10,2) DEFAULT 0,  
    on_time_rate DECIMAL(10,2) DEFAULT 0,      
    throughput INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(department_id, date)
);

CREATE TABLE IF NOT EXISTS task_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL,
    project_id UUID NOT NULL,
    department_id UUID,
    assigned_user_id UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    assigned_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    due_date TIMESTAMP WITH TIME ZONE,
    status INTEGER,
    is_overdue BOOLEAN DEFAULT FALSE,
    cycle_time_days INTEGER,
    delayed_days INTEGER,
    created_by UUID,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(task_id)
);

CREATE TABLE IF NOT EXISTS project_timeline_control (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL,
    project_name VARCHAR(255),
    department_id UUID,
    total_tasks INTEGER DEFAULT 0,
    completed_on_time INTEGER DEFAULT 0,
    overdue_tasks INTEGER DEFAULT 0,
    on_time_rate DECIMAL(10,2) DEFAULT 0,
    avg_delay_days DECIMAL(10,2) DEFAULT 0,
    start_date TIMESTAMP WITH TIME ZONE,
    end_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id)
);

CREATE TABLE IF NOT EXISTS department_delays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL,
    department_id UUID NOT NULL,
    department_name VARCHAR(255),
    overdue_tasks INTEGER DEFAULT 0,
    avg_delay_days DECIMAL(10,2) DEFAULT 0,
    date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, department_id, date)
);

CREATE TABLE IF NOT EXISTS employee_productivity (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    full_name VARCHAR(255),
    email VARCHAR(255),
    department_id UUID,
    date DATE NOT NULL,
    tasks_completed INTEGER DEFAULT 0,
    tasks_overdue INTEGER DEFAULT 0,
    avg_cycle_time DECIMAL(10,2) DEFAULT 0,
    completion_rate DECIMAL(10,2) DEFAULT 0,
    on_time_rate DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, date)
);

CREATE TABLE IF NOT EXISTS weekly_trends (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    week DATE NOT NULL,
    department_id UUID,
    tasks_created INTEGER DEFAULT 0,
    tasks_completed INTEGER DEFAULT 0,
    tasks_overdue INTEGER DEFAULT 0,
    completion_rate DECIMAL(10,2) DEFAULT 0,
    on_time_rate DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(week, department_id)
);