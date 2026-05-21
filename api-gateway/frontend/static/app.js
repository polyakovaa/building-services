const API_URL = 'http://localhost:8080';

function getToken() {
    return localStorage.getItem('access_token');
}
async function apiRequest(endpoint, options = {}) {
    const token = getToken();
    const response = await fetch(`${API_URL}${endpoint}`, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
            ...options.headers
        },
        credentials: 'include'
    });
    
    if (!response.ok) {
        let errorMessage = `Ошибка ${response.status}`;
        try {
            const errorData = await response.json();
            errorMessage = errorData.error || errorData.message || errorMessage;
        } catch(e) {}
 
        const errorMap = {
            'invalid input': 'Неверные данные',
            'name required': 'Название обязательно',
            'customer required': 'Заказчик обязателен',
            'address required': 'Адрес обязателен',
            'start date required': 'Дата начала обязательна',
            'permission denied': 'Нет прав доступа',
            'project not found': 'Проект не найден'
        };
        
        const rusMessage = errorMap[errorMessage.toLowerCase()] || errorMessage;
        throw new Error(rusMessage);
    }
    
    return response;
}


function getStatusClass(status) {
    const map = { 1: 'status-todo', 2: 'status-progress', 3: 'status-completed', 4: 'status-blocked' };
    return map[status] || '';
}

function logout() {
    localStorage.clear();
    window.location.href = '/';
}

function getUserRole() {
    const roles = {
        'ROLE_DIRECTOR': 'Директор',
        'ROLE_GIP': 'ГИП',
        'ROLE_DEPARTMENT_MANAGER': 'Руководитель отдела',
        'ROLE_PROJECT_MANAGER': 'Проектный менеджер',
        'ROLE_WORKER': 'Инженер'
    };
    return roles[localStorage.getItem('user_role')] || '';
}
function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/[&<>]/g, function(m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        return m;
    });
}


if (window.location.pathname !== '/' && window.location.pathname !== '/login') {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = '/';
    }
}

function isUrgent(deadline) {
    if (!deadline) return false;
    const daysLeft = (deadline.seconds * 1000 - Date.now()) / (1000 * 60 * 60 * 24);
    return daysLeft < 3;
}
function getProjectStatusText(status) {
    const map = {
        1: 'Активен',
        2: 'Завершен',
        3: 'Приостановлен',
        4: 'Отменен'
    };
    return map[status] || 'Неизвестно';
}
function getProjectStatusClass(status) {
    const map = { 1: 'status-active', 2: 'status-completed', 3: 'status-onhold', 4: 'status-cancelled' };
    return map[status] || '';
}


function goToProject(projectId) {
    window.location.href = `/project/${projectId}`;
}


function getDeadlineStatus(deadline) {
    if (!deadline) return { text: 'Не указан', class: '' };
    const deadlineDate = new Date(deadline.seconds * 1000);
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const daysLeft = Math.ceil((deadlineDate - today) / (1000 * 60 * 60 * 24));
    
    if (daysLeft < 0) return { text: `Просрочена на ${Math.abs(daysLeft)} дн.`, class: 'deadline-overdue' };
    if (daysLeft === 0) return { text: 'Сегодня', class: 'deadline-urgent' };
    if (daysLeft <= 3) return { text: `${daysLeft} дн.`, class: 'deadline-urgent' };
    return { text: formatDate(deadline), class: '' };
}

function getPriorityText(priority) {
    const map = { 1: 'Низкий', 2: 'Средний', 3: 'Высокий', 4: 'Срочный' };
    return map[priority] || 'Не указан';
}

function getPriorityClass(priority) {
    const map = { 1: 'priority-low', 2: 'priority-medium', 3: 'priority-high', 4: 'priority-urgent' };
    return map[priority] || '';
}


function getStatusText(status) {
    const map = { 1: 'К выполнению', 2: 'В работе', 3: 'Завершена', 4: 'Заблокирована' };
    return map[status] || 'Неизвестно';
}


function getDeadlineStatus(deadline) {
    if (!deadline) return { text: 'Не указан', class: '' };
    const deadlineDate = new Date(deadline.seconds * 1000);
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    
    if (deadlineDate < today) {
        return { text: `Просрочен (${deadlineDate.toLocaleDateString('ru-RU')})`, class: 'deadline-overdue' };
    }
    return { text: deadlineDate.toLocaleDateString('ru-RU'), class: '' };
}

function formatDate(timestamp) {
    if (!timestamp) return 'Не указана';
    return new Date(timestamp.seconds * 1000).toLocaleDateString('ru-RU', {
        year: 'numeric', month: 'long', day: 'numeric'
    });
}

function formatFileSize(bytes) {
    if (!bytes) return '';
    if (bytes < 1024) return bytes + ' Б';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' КБ';
    return (bytes / (1024 * 1024)).toFixed(1) + ' МБ';
}

async function getUserName(userId) {
    if (!userId) return '—';
    try {
        const response = await apiRequest(`/api/users/${userId}`);
        if (response.ok) {
            const user = await response.json();
            return user.full_name || user.email || 'Пользователь';
        }
    } catch (err) {}
    return 'Пользователь';
}

function formatUserOptionLabel(user) {
    if (!user) return '';
    const name = user.user_full_name || user.full_name || '';
    const email = user.user_email || user.email || '';
    if (name && email) return `${name} (${email})`;
    return name || email || user.user_id || user.id || '';
}

async function fetchProjectMembers(projectId) {
    if (!projectId) return [];
    try {
        const response = await apiRequest(`/api/projects/${projectId}/members`);
        if (!response.ok) return [];
        const data = await response.json();
        return data.members || [];
    } catch (err) {
        return [];
    }
}

function sortUsersByLabel(users) {
    return [...(users || [])].sort((a, b) =>
        formatUserOptionLabel(a).localeCompare(formatUserOptionLabel(b), 'ru', { sensitivity: 'base' })
    );
}

function fillUserSelect(selectEl, users, selectedUserId, placeholder) {
    if (!selectEl) return;
    const emptyLabel = placeholder || 'Не назначен';
    selectEl.innerHTML = `<option value="">${emptyLabel}</option>`;
    sortUsersByLabel(users).forEach((u) => {
        const id = u.user_id || u.id;
        if (!id) return;
        const opt = document.createElement('option');
        opt.value = id;
        opt.textContent = formatUserOptionLabel(u);
        if (selectedUserId && id === selectedUserId) opt.selected = true;
        selectEl.appendChild(opt);
    });
}

function filterUserSelect(selectEl, filterText) {
    if (!selectEl) return;
    const q = (filterText || '').trim().toLowerCase();
    Array.from(selectEl.options).forEach((opt) => {
        if (opt.value === '') {
            opt.hidden = false;
            return;
        }
        opt.hidden = q !== '' && !opt.textContent.toLowerCase().includes(q);
    });
}

async function findUsers(query) {
    const q = (query || '').trim();
    if (q.length === 1) return [];
    try {
        const response = await apiRequest(`/api/users/find?q=${encodeURIComponent(q)}`);
        if (!response.ok) return [];
        const data = await response.json();
        return data.users || [];
    } catch (err) {
        return [];
    }
}

let activityTypesCache = null;

function clearActivityTypesCache() {
    activityTypesCache = null;
}

async function loadActivityTypes() {
    if (activityTypesCache) return activityTypesCache;
    try {
        const response = await apiRequest('/api/activity-types');
        if (!response.ok) return [];
        const data = await response.json();
        activityTypesCache = data.activity_types || [];
        return activityTypesCache;
    } catch (err) {
        return [];
    }
}

function formatPlanFact(task) {
    if (!task) return '—';
    const planned = task.planned_hours ?? task.plannedHours ?? 0;
    const actual = task.actual_hours ?? task.actualHours ?? 0;
    if (planned <= 0 && actual <= 0) return '—';
    return `${planned > 0 ? planned : '—'} / ${actual > 0 ? actual : '—'}`;
}

function getActivityTypeName(activityTypeId) {
    if (!activityTypeId || !activityTypesCache) return '—';
    const found = activityTypesCache.find((a) => a.id === activityTypeId);
    return found ? found.name : '—';
}

function fillActivityTypeSelect(selectEl, selectedId, includeEmpty) {
    if (!selectEl) return;
    const emptyLabel = includeEmpty === false ? '' : 'Не выбран';
    selectEl.innerHTML = emptyLabel ? `<option value="">${emptyLabel}</option>` : '';
    (activityTypesCache || []).forEach((a) => {
        const opt = document.createElement('option');
        opt.value = a.id;
        opt.textContent = a.name;
        if (selectedId && a.id === selectedId) opt.selected = true;
        selectEl.appendChild(opt);
    });
}

async function loadUsersForMemberSelect(selectEl, projectId) {
    const [allUsers, members] = await Promise.all([
        findUsers(''),
        fetchProjectMembers(projectId),
    ]);
    const memberIds = new Set(members.map((m) => m.user_id).filter(Boolean));
    const available = allUsers.filter((u) => u && u.id && !memberIds.has(u.id));
    fillUserSelect(selectEl, available, '', 'Выберите пользователя');
}
async function loadUserInfo() {
    const cachedName = localStorage.getItem('user_name') || '';
    const cachedRole = localStorage.getItem('user_role') || '';
    const userNameEl = document.getElementById('userName');
    const userRoleEl = document.getElementById('userRole');
    const roles = {
        'ROLE_DIRECTOR':'Директор',
        'ROLE_GIP':'ГИП',
        'ROLE_DEPARTMENT_MANAGER':'Руководитель отдела',
        'ROLE_PROJECT_MANAGER':'ПМ',
        'ROLE_WORKER':'Инженер'
    };

    if (userNameEl && cachedName) userNameEl.textContent = cachedName;
    if (userRoleEl && cachedRole) userRoleEl.textContent = roles[cachedRole] || '';
    updateAnalyticsMenu(cachedRole);
    updateToolsMenu(cachedRole);

    try {
        const response = await apiRequest('/api/users/me');
        if (response.ok) {
            const user = await response.json();
            localStorage.setItem('user_name', user.full_name || user.email || 'Пользователь');
            localStorage.setItem('user_role', user.role || cachedRole);
            if (userNameEl) userNameEl.textContent = user.full_name || user.email || 'Пользователь';
            if (userRoleEl) userRoleEl.textContent = roles[user.role] || '';
            updateAnalyticsMenu(user.role);
            updateToolsMenu(user.role);
        }
    } catch(err) {
        console.error(err);
    }
}

function updateAnalyticsMenu(role) {
    const analyticsMenuItem = document.getElementById('analyticsMenuItem');
    if (!analyticsMenuItem) return;
    if (role && role !== 'ROLE_WORKER') {
        analyticsMenuItem.style.display = 'flex';
        analyticsMenuItem.onclick = () => window.location.href = '/analytics';
    } else {
        analyticsMenuItem.style.display = 'none';
    }
}

function updateToolsMenu(role) {
    const toolsMenuItem = document.getElementById('toolsMenuItem');
    if (!toolsMenuItem) return;
    const canUse = role === 'ROLE_DIRECTOR' || role === 'ROLE_ADMIN' || role === 'ROLE_GIP';
    if (canUse) {
        toolsMenuItem.style.display = 'flex';
        toolsMenuItem.onclick = () => window.location.href = '/tools';
    } else {
        toolsMenuItem.style.display = 'none';
    }
}

async function updateNotificationBadge() {
    const badge = document.getElementById('notifCount');
    if (!badge) return;
    try {
        const response = await apiRequest('/api/notifications/unread-count');
        const data = await response.json();
        const count = Number(data.count || 0);
        if (count > 0) {
            badge.textContent = count > 99 ? '99+' : String(count);
            badge.style.display = 'inline-flex';
        } else {
            badge.textContent = '';
            badge.style.display = 'none';
        }
    } catch (err) {
        console.error('Failed to load notification count', err);
        badge.style.display = 'none';
    }
}

function initNotificationsNavigation() {
    document.querySelectorAll('[data-page="notifications"]').forEach(item => {
        item.addEventListener('click', () => {
            window.location.href = '/notifications';
        });
    });
}

document.addEventListener('DOMContentLoaded', () => {
    initNotificationsNavigation();
    updateNotificationBadge();
    if (document.getElementById('notifCount')) {
        setInterval(updateNotificationBadge, 60000);
    }
});