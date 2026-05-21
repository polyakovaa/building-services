let toolsCurrentUser = null;

async function initializeTools() {
    try {
        toolsCurrentUser = await getToolsCurrentUser();
        const role = toolsCurrentUser?.role;
        const isDirector = role === 'ROLE_DIRECTOR' || role === 'ROLE_ADMIN';
        const isGip = role === 'ROLE_GIP';
        if (!isDirector && !isGip) {
            showToolsAccessDenied();
            return;
        }
        setupToolsAdminCatalog(isDirector, isGip);
    } catch (error) {
        console.error('Failed to initialize tools:', error);
        showToolsError('Не удалось загрузить страницу инструментов');
    }
}

async function getToolsCurrentUser() {
    const response = await apiRequest('/api/users/me');
    if (!response.ok) throw new Error('Failed to get user info');
    const userInfo = await response.json();
    const fullUserResponse = await apiRequest(`/api/users/${userInfo.id}`);
    if (fullUserResponse.ok) {
        return await fullUserResponse.json();
    }
    return userInfo;
}

function setupToolsAdminCatalog(isDirector, isGip) {
    const section = document.getElementById('adminCatalogSection');
    const deptCard = document.getElementById('departmentAdminCard');
    const activityCard = document.getElementById('activityTypeAdminCard');
    if (!section) return;

    section.style.display = 'block';
    if (isDirector && deptCard) deptCard.style.display = 'block';
    if ((isDirector || isGip) && activityCard) activityCard.style.display = 'block';

    loadToolsCatalogLists(isDirector, isGip);

    const deptInput = document.getElementById('newDepartmentName');
    if (deptInput) {
        deptInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') createDepartment();
        });
    }
    const activityInput = document.getElementById('newActivityTypeName');
    if (activityInput) {
        activityInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') createActivityType();
        });
    }
}

async function loadToolsCatalogLists(isDirector, isGip) {
    if (isDirector) await loadDepartmentsList();
    if (isDirector || isGip) await loadActivityTypesList();
}

async function loadDepartmentsList() {
    const list = document.getElementById('departmentsList');
    if (!list) return;
    try {
        const response = await apiRequest('/api/departments');
        const data = await response.json();
        const departments = data.departments || data.Departments || [];
        if (!departments.length) {
            list.innerHTML = '<li class="admin-catalog-empty">Пока нет отделов</li>';
            return;
        }
        list.innerHTML = departments
            .map((d) => `<li>${escapeToolsHtml(d.name || d.id)}</li>`)
            .join('');
    } catch (err) {
        list.innerHTML = '<li class="admin-catalog-empty">Не удалось загрузить отделы</li>';
    }
}

async function loadActivityTypesList() {
    const list = document.getElementById('activityTypesList');
    if (!list) return;
    try {
        const response = await apiRequest('/api/activity-types');
        if (!response.ok) throw new Error('activity types');
        const data = await response.json();
        const types = data.activity_types || [];
        if (!types.length) {
            list.innerHTML = '<li class="admin-catalog-empty">Пока нет видов работ</li>';
            return;
        }
        list.innerHTML = types
            .map((t) => `<li>${escapeToolsHtml(t.name || t.id)}</li>`)
            .join('');
    } catch (err) {
        list.innerHTML = '<li class="admin-catalog-empty">Не удалось загрузить виды работ</li>';
    }
}

function showAdminCatalogMessage(text, isError) {
    const section = document.getElementById('adminCatalogSection');
    if (!section) return;
    const msg = document.createElement('div');
    msg.className = isError ? 'message error' : 'message success';
    msg.textContent = text;
    section.insertBefore(msg, section.querySelector('.admin-catalog-grid'));
    setTimeout(() => msg.remove(), 4000);
}

async function createDepartment() {
    const input = document.getElementById('newDepartmentName');
    if (!input) return;
    const name = input.value.trim();
    if (!name) {
        showAdminCatalogMessage('Введите название отдела', true);
        return;
    }
    try {
        const response = await apiRequest('/api/departments', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name })
        });
        if (!response.ok) {
            const err = await response.json().catch(() => ({}));
            throw new Error(err.error || err.message || 'Не удалось создать отдел');
        }
        input.value = '';
        showAdminCatalogMessage('Отдел добавлен');
        await loadDepartmentsList();
    } catch (err) {
        showAdminCatalogMessage(err.message || 'Ошибка при создании отдела', true);
    }
}

async function createActivityType() {
    const input = document.getElementById('newActivityTypeName');
    if (!input) return;
    const name = input.value.trim();
    if (!name) {
        showAdminCatalogMessage('Введите название вида работ', true);
        return;
    }
    try {
        const response = await apiRequest('/api/activity-types', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name })
        });
        if (!response.ok) {
            const err = await response.json().catch(() => ({}));
            throw new Error(err.error || err.message || 'Не удалось создать вид работ');
        }
        input.value = '';
        if (typeof clearActivityTypesCache === 'function') clearActivityTypesCache();
        showAdminCatalogMessage('Вид работ добавлен');
        await loadActivityTypesList();
    } catch (err) {
        showAdminCatalogMessage(err.message || 'Ошибка при создании вида работ', true);
    }
}

function escapeToolsHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showToolsAccessDenied() {
    document.getElementById('mainContent').innerHTML = `
        <div class="empty-state">
            <h3>Доступ запрещен</h3>
            <p>Страница доступна директору и ГИПу</p>
            <button class="save-btn" onclick="window.location.href='/dashboard'">На главную</button>
        </div>
    `;
}

function showToolsError(message) {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'message error';
    errorDiv.textContent = message;
    document.getElementById('mainContent').prepend(errorDiv);
}
