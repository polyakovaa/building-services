let toolsCurrentUser = null;
let toolsDepartmentsCache = [];

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
    const staffCard = document.getElementById('departmentStaffCard');
    if (!section) return;

    section.style.display = 'block';
    if (isDirector && deptCard) deptCard.style.display = 'block';
    if ((isDirector || isGip) && activityCard) activityCard.style.display = 'block';
    if (isDirector && staffCard) {
        staffCard.style.display = 'block';
        initDepartmentStaffCard();
    }

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

async function fetchToolsDepartments() {
    const response = await apiRequest('/api/departments');
    if (!response.ok) throw new Error('departments');
    const data = await response.json();
    toolsDepartmentsCache = data.departments || data.Departments || [];
    return toolsDepartmentsCache;
}

async function loadDepartmentsList() {
    const list = document.getElementById('departmentsList');
    if (!list) return;
    try {
        const departments = await fetchToolsDepartments();
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

function fillStaffDepartmentSelect(selectEl, departments, selectedId) {
    if (!selectEl) return;
    selectEl.innerHTML = '';
    if (!departments.length) {
        const opt = document.createElement('option');
        opt.value = '';
        opt.textContent = 'Сначала создайте отдел';
        selectEl.appendChild(opt);
        selectEl.disabled = true;
        return;
    }
    selectEl.disabled = false;
    departments.forEach((d) => {
        const opt = document.createElement('option');
        opt.value = d.id;
        opt.textContent = d.name || d.id;
        if (selectedId && d.id === selectedId) opt.selected = true;
        selectEl.appendChild(opt);
    });
}

async function initDepartmentStaffCard() {
    const deptSelect = document.getElementById('staffDepartmentId');
    const userSelect = document.getElementById('staffUserId');
    const userFilter = document.getElementById('staffUserFilter');
    const staffList = document.getElementById('departmentStaffList');
    if (!deptSelect || !userSelect || !staffList) return;

    try {
        const departments = await fetchToolsDepartments();
        const prevDeptId = deptSelect.value;
        fillStaffDepartmentSelect(deptSelect, departments, prevDeptId);
    } catch (err) {
        staffList.innerHTML = '<li class="admin-catalog-empty">Не удалось загрузить отделы</li>';
        return;
    }

    const users = await findUsers('');
    fillUserSelect(userSelect, users, '', 'Выберите пользователя');

    if (userFilter && userFilter.dataset.bound !== '1') {
        userFilter.addEventListener('input', () => filterUserSelect(userSelect, userFilter.value));
        userFilter.dataset.bound = '1';
    }
    if (deptSelect.dataset.bound !== '1') {
        deptSelect.addEventListener('change', () => loadDepartmentStaff());
        deptSelect.dataset.bound = '1';
    }

    if (staffList.dataset.bound !== '1') {
        staffList.addEventListener('click', (e) => {
            const btn = e.target.closest('[data-remove-user-id]');
            if (!btn) return;
            const userId = btn.getAttribute('data-remove-user-id');
            const deptId = deptSelect.value;
            if (userId && deptId) removeUserFromDepartment(deptId, userId);
        });
        staffList.dataset.bound = '1';
    }

    if (deptSelect.value) {
        await loadDepartmentStaff();
    } else if (deptSelect.options.length && deptSelect.options[0].value) {
        await loadDepartmentStaff();
    } else {
        staffList.innerHTML = '<li class="admin-catalog-empty">Нет отделов для назначения</li>';
    }
}

function formatStaffUserLabel(user) {
    const name = user.full_name || user.fullName || '';
    const email = user.email || '';
    if (name && email) return `${name} (${email})`;
    return name || email || user.id || '';
}

async function loadDepartmentStaff() {
    const deptSelect = document.getElementById('staffDepartmentId');
    const list = document.getElementById('departmentStaffList');
    if (!deptSelect || !list) return;

    const deptId = deptSelect.value;
    if (!deptId) {
        list.innerHTML = '<li class="admin-catalog-empty">Выберите отдел</li>';
        return;
    }

    list.innerHTML = '<li class="admin-catalog-empty">Загрузка…</li>';
    try {
        const response = await apiRequest(`/api/departments/${deptId}/users`);
        if (!response.ok) throw new Error('staff');
        const data = await response.json();
        const users = data.users || [];
        if (!users.length) {
            list.innerHTML = '<li class="admin-catalog-empty">В отделе пока нет сотрудников</li>';
            return;
        }
        list.innerHTML = users
            .map((u) => {
                const label = escapeToolsHtml(formatStaffUserLabel(u));
                const userId = escapeToolsHtml(u.id || '');
                return `<li class="admin-catalog-staff-row">
                    <span>${label}</span>
                    <button type="button" class="cancel-btn admin-catalog-staff-remove" data-remove-user-id="${userId}">Убрать</button>
                </li>`;
            })
            .join('');
    } catch (err) {
        list.innerHTML = '<li class="admin-catalog-empty">Не удалось загрузить сотрудников</li>';
    }
}

async function assignUserToDepartment() {
    const deptSelect = document.getElementById('staffDepartmentId');
    const userSelect = document.getElementById('staffUserId');
    if (!deptSelect || !userSelect) return;

    const deptId = deptSelect.value;
    const userId = userSelect.value;
    if (!deptId) {
        showAdminCatalogMessage('Сначала создайте и выберите отдел', true);
        return;
    }
    if (!userId) {
        showAdminCatalogMessage('Выберите сотрудника', true);
        return;
    }

    try {
        const response = await apiRequest(`/api/departments/${deptId}/users/${userId}`, {
            method: 'POST',
        });
        if (!response.ok) {
            const err = await response.json().catch(() => ({}));
            throw new Error(err.error || err.message || 'Не удалось назначить сотрудника');
        }
        userSelect.value = '';
        const filter = document.getElementById('staffUserFilter');
        if (filter) filter.value = '';
        showAdminCatalogMessage('Сотрудник назначен в отдел');
        await loadDepartmentStaff();
    } catch (err) {
        showAdminCatalogMessage(err.message || 'Ошибка при назначении', true);
    }
}

async function removeUserFromDepartment(deptId, userId) {
    if (!deptId || !userId) return;
    try {
        const response = await apiRequest(`/api/departments/${deptId}/users/${userId}`, {
            method: 'DELETE',
        });
        if (!response.ok) {
            const err = await response.json().catch(() => ({}));
            throw new Error(err.error || err.message || 'Не удалось убрать сотрудника');
        }
        showAdminCatalogMessage('Сотрудник убран из отдела');
        await loadDepartmentStaff();
    } catch (err) {
        showAdminCatalogMessage(err.message || 'Ошибка при удалении из отдела', true);
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
        if (document.getElementById('departmentStaffCard')?.style.display !== 'none') {
            await initDepartmentStaffCard();
        }
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
