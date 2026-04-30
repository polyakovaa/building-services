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
    
    if (response.status === 401) {
        localStorage.clear();
        window.location.href = '/';
        throw new Error('Session expired');
    }
    return response;
}

function getStatusText(status) {
    const map = { 1: 'К выполнению', 2: 'В работе', 3: 'Завершена', 4: 'Заблокирована' };
    return map[status] || 'Неизвестно';
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

async function loadUserInfo() {
    try {
        const response = await apiRequest('/api/users/me');
        if (response.ok) {
            const user = await response.json();
            const userNameEl = document.getElementById('userName');
            const userRoleEl = document.getElementById('userRole');
            if (userNameEl) userNameEl.textContent = user.full_name || 'Пользователь';
            if (userRoleEl) userRoleEl.textContent = getUserRole();
        }
    } catch (err) {
        console.error('Failed to load user info:', err);
    }
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
