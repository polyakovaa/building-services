let currentPeriod = 'week';
let currentUser = null;
let userDepartment = null;
let trendsChartInstance = null;
let workloadChartInstance = null;

async function initializeAnalytics() {
    try {
        currentUser = await getCurrentUser();
        if (currentUser.role === 'ROLE_WORKER') {
            showAccessDenied();
            return;
        }
        await loadUserData();
        setupFilters();
        await loadAnalyticsData();
        
    } catch (error) {
        console.error('Failed to initialize analytics:', error);
        showError('Не удалось загрузить аналитику');
    }
}

async function getCurrentUser() {
    const response = await apiRequest('/api/users/me');
    if (response.ok) {
        const userInfo = await response.json();
        const fullUserResponse = await apiRequest(`/api/users/${userInfo.id}`);
        if (fullUserResponse.ok) {
            return await fullUserResponse.json();
        }
        return userInfo;
    }
    throw new Error('Failed to get user info');
}

async function loadUserData() {
    try {
        if (currentUser.role === 'ROLE_DEPARTMENT_MANAGER') {
            userDepartment = currentUser.department_id;
        }
    } catch (error) {
        console.error('Failed to load user data:', error);
    }
}

function setupFilters() {
    document.querySelectorAll('.filter-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('.filter-tab').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            currentPeriod = tab.dataset.period;
            loadAnalyticsData();
        });
    });
}

async function loadAnalyticsData() {
    showLoading();
    
    try {
        const filters = getFilters();
        const [dashboardData, workloadData, trendsData, productivityData, timelineData] = await Promise.all([
            loadDashboard(filters),
            loadWorkload(filters),
            loadTrends(filters),
            loadProductivity(filters),
            loadTimeline(filters)
        ]);
        updateDashboard(dashboardData);
        updateWorkloadChart(workloadData);
        updateTrendsChart(trendsData);
        updateProductivityTable(productivityData);
        updateTimelineTable(timelineData);
        updateDepartmentCards(workloadData);

    } catch (error) {
        console.error('Failed to load analytics data:', error);
        showError('Не удалось загрузить данные аналитики');
    } finally {
        hideLoading();
    }
}

function getFilters() {
    const filters = {
        period: currentPeriod
    };

    if (currentUser.role === 'ROLE_DEPARTMENT_MANAGER' && userDepartment) {
        filters.department_id = userDepartment;
    }
    const { fromDate, toDate } = getPeriodDates(currentPeriod);
    filters.from_date = fromDate;
    filters.to_date = toDate;

    return filters;
}

function getPeriodDates(period) {
    const now = new Date();
    let fromDate, toDate;

    switch (period) {
        case 'week':
            fromDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
            break;
        case 'month':
            fromDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
            break;
        case 'quarter':
            fromDate = new Date(now.getTime() - 90 * 24 * 60 * 60 * 1000);
            break;
        default:
            fromDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
    }

    toDate = now;
    return {
        fromDate: fromDate.toISOString().split('T')[0],
        toDate: toDate.toISOString().split('T')[0]
    };
}

async function loadDashboard(filters) {
    const params = new URLSearchParams();
    if (filters.department_id) params.append('department_id', filters.department_id);
    if (filters.project_id) params.append('project_id', filters.project_id);
    if (filters.from_date) params.append('from_date', filters.from_date);
    if (filters.to_date) params.append('to_date', filters.to_date);

    const response = await apiRequest(`/api/analytics/dashboard?${params}`);
    if (!response.ok) throw new Error('Failed to load dashboard');
    return await response.json();
}

async function loadWorkload(filters) {
    const params = new URLSearchParams();
    if (filters.department_id) params.append('department_id', filters.department_id);
    if (filters.project_id) params.append('project_id', filters.project_id);
    if (filters.from_date) params.append('from_date', filters.from_date);
    if (filters.to_date) params.append('to_date', filters.to_date);
    params.append('days', getDaysForPeriod(currentPeriod));

    const response = await apiRequest(`/api/analytics/workload?${params}`);
    if (!response.ok) throw new Error('Failed to load workload');
    return await response.json();
}

async function loadTrends(filters) {
    const params = new URLSearchParams();
    if (filters.department_id) params.append('department_id', filters.department_id);
    if (filters.project_id) params.append('project_id', filters.project_id);
    if (filters.from_date) params.append('from_date', filters.from_date);
    if (filters.to_date) params.append('to_date', filters.to_date);
    params.append('group_by', currentPeriod === 'week' ? 'day' : 'week');
    params.append('weeks', getWeeksForPeriod(currentPeriod));

    const response = await apiRequest(`/api/analytics/trends?${params}`);
    if (!response.ok) throw new Error('Failed to load trends');
    return await response.json();
}

async function loadProductivity(filters) {
    const params = new URLSearchParams();
    if (filters.department_id) params.append('department_id', filters.department_id);
    if (filters.from_date) params.append('from_date', filters.from_date);
    if (filters.to_date) params.append('to_date', filters.to_date);

    const response = await apiRequest(`/api/analytics/productivity?${params}`);
    if (!response.ok) throw new Error('Failed to load productivity');
    return await response.json();
}

async function loadTimeline(filters) {
    const params = new URLSearchParams();
    if (filters.project_id) params.append('project_id', filters.project_id);
    if (filters.department_id) params.append('department_id', filters.department_id);
    if (filters.from_date) params.append('from_date', filters.from_date);
    if (filters.to_date) params.append('to_date', filters.to_date);

    const response = await apiRequest(`/api/analytics/projects/timeline?${params}`);
    if (!response.ok) throw new Error('Failed to load timeline');
    return await response.json();
}

function updateDashboard(data) {
    const dashboard = data.dashboard || data;
    
    updateStat('activeProjects', dashboard.active_projects || 0);
    updateStat('totalTasks', dashboard.total_tasks || 0);
    updateStat('overdueTasks', dashboard.overdue_tasks || 0);
    updateStat('completionRate', formatPercentage(dashboard.completion_rate || 0));
    updateStat('onTimeRate', formatPercentage(dashboard.on_time_rate || 0));
}
function updateStat(id, value) {
    const element = document.getElementById(id);
    if (element) {
        element.textContent = value;
    }
}
function updateTrendsChart(data) {
    const canvas = document.getElementById('trendsChart');
    if (!canvas) return;

    const trends = data.weekly_trend || data.trends || [];

    if (typeof Chart === 'undefined') {
        const ctx = canvas.getContext('2d');
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        if (trends.length === 0) showEmptyChart(canvas, 'Нет данных');
        return;
    }

    if (trendsChartInstance) {
        trendsChartInstance.destroy();
        trendsChartInstance = null;
    }

    if (trends.length === 0) {
        const ctx = canvas.getContext('2d');
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        showEmptyChart(canvas, 'Нет данных для отображения');
        return;
    }

    const labels = trends.map(t => formatTrendLabel(t.week || ''));
    const created = trends.map(t => Number(t.created) || 0);
    const completed = trends.map(t => Number(t.completed) || 0);

    trendsChartInstance = new Chart(canvas.getContext('2d'), {
        type: 'line',
        data: {
            labels,
            datasets: [
                {
                    label: 'Создано',
                    data: created,
                    borderColor: '#4a6cf7',
                    backgroundColor: 'rgba(74,108,247,0.12)',
                    tension: 0.25,
                    fill: false
                },
                {
                    label: 'Завершено',
                    data: completed,
                    borderColor: '#2e7d32',
                    backgroundColor: 'rgba(46,125,50,0.12)',
                    tension: 0.25,
                    fill: false
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                title: { display: true, text: currentPeriod === 'week' ? 'Тренды задач по дням' : 'Тренды задач по неделям' },
                legend: { display: true, position: 'bottom' }
            },
            scales: {
                x: { title: { display: true, text: currentPeriod === 'week' ? 'День' : 'Неделя' } },
                y: { beginAtZero: true, title: { display: true, text: 'Количество задач' } }
            }
        }
    });
}

function updateWorkloadChart(data) {
    const canvas = document.getElementById('workloadChart');
    if (!canvas) return;

    const workloads = data.department_workload || data.workloads || [];

    if (typeof Chart === 'undefined') {
        const ctx = canvas.getContext('2d');
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        if (workloads.length === 0) showEmptyChart(canvas, 'Нет данных');
        return;
    }

    if (workloadChartInstance) {
        workloadChartInstance.destroy();
        workloadChartInstance = null;
    }

    if (workloads.length === 0) {
        const ctx = canvas.getContext('2d');
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        showEmptyChart(canvas, 'Нет данных для отображения');
        return;
    }

    const labels = workloads.map(w => w.department_name || w.department_id || '');
    const wip = workloads.map(w => Number(w.wip) || 0);
    const completed = workloads.map(w => Number(w.completed) || 0);

    workloadChartInstance = new Chart(canvas.getContext('2d'), {
        type: 'bar',
        data: {
            labels,
            datasets: [
                { label: 'В работе (WIP)', data: wip, backgroundColor: 'rgba(74,108,247,0.75)' },
                { label: 'Завершено (за период)', data: completed, backgroundColor: 'rgba(46,125,50,0.75)' }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                title: { display: true, text: 'Нагрузка по отделам' },
                legend: { display: true, position: 'bottom' }
            },
            scales: {
                x: { title: { display: true, text: 'Отдел' } },
                y: { beginAtZero: true, title: { display: true, text: 'Задачи' } }
            }
        }
    });
}
function updateProductivityTable(data) {
    const tbody = document.querySelector('#productivityTable tbody');
    if (!tbody) return;
    const employees = data.employees || [];
    if (employees.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="empty-state">Нет данных</td></tr>';
        return;
    }
    tbody.innerHTML = employees.map(emp => `
        <tr>
            <td>${escapeHtml(employeeDisplayName(emp))}</td>
            <td>${emp.tasks_completed || 0}</td>
            <td class="danger">${emp.tasks_overdue || 0}</td>
            <td>${formatDuration(emp.avg_cycle_time)}</td>
            <td>
                <div class="progress-bar">
                    <div class="progress-fill ${getProgressClass(efficiencyPct(emp))}" 
                         style="width: ${efficiencyPct(emp)}%"></div>
                </div>
                <span class="status-badge ${getBadgeClass(efficiencyPct(emp))}">
                    ${formatPercentage(efficiencyPct(emp))}
                </span>
            </td>
        </tr>
    `).join('');
}
function updateTimelineTable(data) {
    const tbody = document.querySelector('#timelineTable tbody');
    if (!tbody) return;
    const projects = data.projects || data.project_timeline_control || [];
    
    if (projects.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="empty-state">Нет данных</td></tr>';
        return;
    }

    tbody.innerHTML = projects.map(project => `
        <tr>
            <td>${escapeHtml(project.project_name || 'N/A')}</td>
            <td>${project.total_tasks || 0}</td>
            <td class="good">${project.completed_on_time || 0}</td>
            <td class="danger">${project.overdue_tasks || 0}</td>
            <td>
                <div class="progress-bar">
                    <div class="progress-fill ${getProgressClass(project.on_time_rate)}" 
                         style="width: ${project.on_time_rate || 0}%"></div>
                </div>
                <span class="status-badge ${getBadgeClass(project.on_time_rate)}">
                    ${formatPercentage(project.on_time_rate || 0)}
                </span>
            </td>
            <td>${formatDuration(project.avg_delay_days)}</td>
        </tr>
    `).join('');
}
function updateDepartmentCards(data) {
    const container = document.getElementById('departmentGrid');
    if (!container) return;

    const workloads = data.workloads || data.department_workload || [];
 
    if (workloads.length === 0) {
        container.innerHTML = '<div class="empty-state">Нет данных по отделам</div>';
        return;
    }
    container.innerHTML = workloads.map(dept => `
        <div class="department-card">
            <h4>${escapeHtml(dept.department_name || 'N/A')}</h4>
            <div class="department-metrics">
                <div class="department-metric">
                    <span class="department-metric-label">В работе</span>
                    <span class="department-metric-value">${dept.wip || 0}</span>
                </div>
                <div class="department-metric">
                    <span class="department-metric-label">Завершено</span>
                    <span class="department-metric-value good">${dept.completed || 0}</span>
                   </div>
                <div class="department-metric">
                    <span class="department-metric-label">Просрочено</span>
                    <span class="department-metric-value danger">${dept.overdue || 0}</span>
                </div>
                <div class="department-metric">
                    <span class="department-metric-label">Продуктивность</span>
                    <span class="department-metric-value">${formatPercentage(dept.productivity || 0)}</span>
                </div>
            </div>
        </div>
    `).join('');
}

function efficiencyPct(emp) {
    const v = emp.on_time_rate;
    return v != null && !Number.isNaN(Number(v)) ? Number(v) : 0;
}

function formatPercentage(value) {
    return `${Math.round(value)}%`;
}

function formatDuration(days) {
    if (!days) return '0 дней';
    return `${Math.round(days)} дней`;
}

function getDaysForPeriod(period) {
    switch (period) {
        case 'week': return 7;
        case 'month': return 30;
        case 'quarter': return 90;
        default: return 30;
    }
}

function getWeeksForPeriod(period) {
    switch (period) {
        case 'week': return 1;
        case 'month': return 4;
        case 'quarter': return 12;
        default: return 4;
    }
}

function formatTrendLabel(value) {
    if (!value) return '';
    const date = new Date(`${value}T00:00:00`);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' });
}

function employeeDisplayName(emp) {
    if (emp.full_name && emp.full_name !== emp.user_id) return emp.full_name;
    if (emp.email) return emp.email;
    return emp.user_id || 'Неизвестный сотрудник';
}

function getProgressClass(value) {
    if (value >= 80) return 'good';
    if (value >= 60) return 'warning';
    return 'danger';
}

function getBadgeClass(value) {
    if (value >= 80) return 'good';
    if (value >= 60) return 'warning';
    return 'danger';
}

function showEmptyChart(canvas, message) {
    const ctx = canvas.getContext('2d');
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.fillStyle = '#64748b';
    ctx.font = '14px Arial';
    ctx.textAlign = 'center';
    ctx.fillText(message, canvas.width / 2, canvas.height / 2);
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showLoading() {
    document.querySelectorAll('.loading').forEach(el => {
        el.style.display = 'flex';
    });
}

function hideLoading() {
    document.querySelectorAll('.loading').forEach(el => {
        el.style.display = 'none';
    });
}

function showAccessDenied() {
    document.getElementById('mainContent').innerHTML = `
        <div class="empty-state">
            <h3>Доступ запрещен</h3>
            <p>У вас нет прав для просмотра аналитики</p>
            <button class="save-btn" onclick="window.location.href='/dashboard'">Вернуться на главную</button>
        </div>
    `;
}

function showError(message) {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'message error';
    errorDiv.textContent = message;
    document.getElementById('mainContent').prepend(errorDiv);
    
    setTimeout(() => {
        errorDiv.remove();
    }, 5000);
}
