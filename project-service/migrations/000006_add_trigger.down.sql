DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_tasks_updated_at ON tasks;
DROP TRIGGER IF EXISTS update_project_timeline_updated_at ON project_timeline;
DROP FUNCTION IF EXISTS update_updated_at_column;