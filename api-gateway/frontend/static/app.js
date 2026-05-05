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
async function loadUserInfo() {
    try {
        const response = await apiRequest('/api/users/me');
        if (response.ok) {
            const user = await response.json();
            document.getElementById('userName').textContent = user.full_name || 'Пользователь';
            const roles = { 
                'ROLE_DIRECTOR':'Директор',
                'ROLE_GIP':'ГИП',
                'ROLE_DEPARTMENT_MANAGER':'Руководитель отдела',
                'ROLE_PROJECT_MANAGER':'ПМ',
                'ROLE_WORKER':'Инженер'
            };
            document.getElementById('userRole').textContent = roles[user.role] || '';
        }
    } catch(err) {
        console.error(err);
    }
}