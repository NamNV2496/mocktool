const API_BASE_URL = 'http://localhost:8081/api/v1/mocktool';

let features = [];
let scenarios = [];
let mockAPIs = [];
let loadTestScenarios = [];
let activeScenarioId = null; // Track the active scenario ID for the current accountId
let globalActiveScenarioId = null; // Track the global active scenario ID

let parsedEnums = {};
let parsedMessages = {};
let protoTemplates = [];
let selectedProtoInput = null;
let selectedProtoOutput = null;

const pagination = {
    features: { page: 1, totalPages: 1 },
    scenarios: { page: 1, totalPages: 1 },
    mockapis: { page: 1, totalPages: 1 },
    loadtest: { page: 1, totalPages: 1 }
};

document.addEventListener('DOMContentLoaded', function() {
    initializeTabs();
    loadFeatures();
    initProtoFileInput();
    
    // Add event listener for accounts input to update count
    const accountsInput = document.getElementById('loadtest-accounts-input');
    if (accountsInput) {
        accountsInput.addEventListener('input', updateAccountCount);
    }
});

function initializeTabs() {
    const tabButtons = document.querySelectorAll('.tab-btn');
    tabButtons.forEach(button => {
        button.addEventListener('click', function() {
            const targetTab = this.getAttribute('data-tab');
            switchTab(targetTab);
        });
    });
}

function navigateToMockAPIs(featureName, scenarioName) {
    // Switch to Mock APIs tab
    switchTab('mockapis');

    // Wait for filters to be populated, then set the values and load data
    setTimeout(() => {
        // Set feature filter
        const featureSelect = document.getElementById('mockapi-feature-filter');
        if (featureSelect) {
            featureSelect.value = featureName;
            featureSelect.dataset.selectedValue = featureName;
            featureSelect.dataset.selectedText = featureName;
        }

        // Set scenario filter
        const scenarioSelect = document.getElementById('mockapi-scenario-filter');
        if (scenarioSelect) {
            scenarioSelect.value = scenarioName;
            scenarioSelect.dataset.selectedValue = scenarioName;
            scenarioSelect.dataset.selectedText = scenarioName;
        }

        // Load Mock APIs with the selected scenario
        loadMockAPIs(scenarioName);
    }, 100);
}

function switchTab(tabName) {
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));

    document.querySelector(`[data-tab="${tabName}"]`).classList.add('active');
    document.getElementById(tabName).classList.add('active');

    if (tabName === 'features') {
        loadFeatures();

        // Add Enter key support for search
        const searchInput = document.getElementById('feature-search-query');
        if (searchInput && !searchInput.dataset.listenerAdded) {
            searchInput.dataset.listenerAdded = 'true';
            searchInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    searchFeatures();
                }
            });
        }
    } else if (tabName === 'scenarios') {
        populateFeatureFilters();
    } else if (tabName === 'mockapis') {
        populateMockAPIFilters();
    } else if (tabName === 'loadtest') {
        loadLoadTestScenarios();

        // Add Enter key support for search
        const searchInput = document.getElementById('loadtest-search-query');
        if (searchInput && !searchInput.dataset.listenerAdded) {
            searchInput.dataset.listenerAdded = 'true';
            searchInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    searchLoadTestScenarios();
                }
            });
        }
    }
}

async function loadFeatures(page = 1, searchQuery = '') {
    try {
        let url;
        if (searchQuery && searchQuery.trim() !== '') {
            // Use search endpoint
            url = `${API_BASE_URL}/features/search?q=${encodeURIComponent(searchQuery)}&page=${page}&page_size=10`;
        } else {
            // Use regular list endpoint
            url = `${API_BASE_URL}/features?page=${page}&page_size=10`;
        }

        const response = await fetch(url);
        if (!response.ok) throw new Error('Failed to load features');

        const result = await response.json();
        features = result.data;
        pagination.features = { page: result.page, totalPages: result.total_pages };
        renderFeaturesTable();
        renderPagination('features', result.page, result.total_pages);
    } catch (error) {
        console.error('Error loading features:', error);
        showError(t('feature.error.load'));
    }
}

function renderFeaturesTable() {
    const tbody = document.getElementById('features-table-body');

    if (!features || features.length === 0) {
        tbody.innerHTML = `<tr><td colspan="5" class="loading">${t('feature.noResults')}</td></tr>`;
        return;
    }

    tbody.innerHTML = features.map(feature => `
        <tr>
            <td class="text-truncate" title="${feature.name || 'N/A'}"><strong>${feature.name || 'N/A'}</strong></td>
            <td class="text-truncate" title="${feature.description || '-'}">${feature.description || '-'}</td>
            <td><span class="status-badge ${feature.is_active ? 'status-active' : 'status-inactive'}">
                ${feature.is_active ? t('common.active') : t('common.inactive')}
            </span></td>
            <td>${formatDate(feature.created_at)}</td>
            <td class="actions">
                <button class="btn btn-edit" onclick='editFeature(${JSON.stringify(feature).replace(/'/g, "&#39;")})'>${t('common.edit')}</button>
                <button class="btn btn-delete" onclick="deleteFeature('${feature.id}')">${t('common.delete')}</button>
            </td>
        </tr>
    `).join('');
}

async function loadScenarios(featureName, page = 1) {
    if (!featureName) {
        document.getElementById('scenarios-table-body').innerHTML =
            `<tr><td colspan="6" class="loading">${t('scenario.loading')}</td></tr>`;
        document.getElementById('scenarios-pagination').innerHTML = '';
        activeScenarioId = null;
        globalActiveScenarioId = null;
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/scenarios?feature_name=${featureName}&page=${page}&page_size=10`);
        if (!response.ok) throw new Error('Failed to load scenarios');

        const result = await response.json();
        scenarios = result.data;
        pagination.scenarios = { page: result.page, totalPages: result.total_pages };

        // Fetch active scenario if accountId is provided
        const accountId = document.getElementById('scenario-account-id-filter')?.value.trim();
        if (accountId) {
            await fetchActiveScenario(featureName, accountId);
        } else {
            activeScenarioId = null;
        }

        // Always fetch global active scenario
        await fetchGlobalActiveScenario(featureName);

        renderScenariosTable();
        renderPagination('scenarios', result.page, result.total_pages);
    } catch (error) {
        console.error('Error loading scenarios:', error);
        showError(t('scenario.error.load'));
    }
}

async function fetchActiveScenario(featureName, accountId) {
    try {
        const response = await fetch(`${API_BASE_URL}/scenarios/active?feature_name=${featureName}`, {
            headers: {
                'X-Account-Id': accountId
            }
        });

        if (response.ok) {
            const activeScenario = await response.json();
            activeScenarioId = activeScenario.id;
        } else {
            // No active scenario for this account
            activeScenarioId = null;
        }
    } catch (error) {
        console.error('Error fetching active scenario:', error);
        activeScenarioId = null;
    }
}

async function fetchGlobalActiveScenario(featureName) {
    try {
        // Fetch global active scenario (without X-Account-Id header)
        const response = await fetch(`${API_BASE_URL}/scenarios/active?feature_name=${featureName}`);

        if (response.ok) {
            const activeScenario = await response.json();
            globalActiveScenarioId = activeScenario.id;
        } else {
            // No global active scenario
            globalActiveScenarioId = null;
        }
    } catch (error) {
        console.error('Error fetching global active scenario:', error);
        globalActiveScenarioId = null;
    }
}

function renderScenariosTable() {
    const tbody = document.getElementById('scenarios-table-body');

    if (!scenarios || scenarios.length === 0) {
        tbody.innerHTML = `<tr><td colspan="6" class="loading">${t('scenario.noResults')}</td></tr>`;
        return;
    }

    // Get the accountId filter value
    const filterAccountId = document.getElementById('scenario-account-id-filter')?.value.trim();

    tbody.innerHTML = scenarios.map(scenario => {
        // Check if this scenario is the active one for the current account
        const isActive = activeScenarioId && scenario.id === activeScenarioId;
        // Check if this scenario is the global active one
        const isGlobalActive = globalActiveScenarioId && scenario.id === globalActiveScenarioId;

        const rowStyle = (isActive || isGlobalActive) ? 'background-color: #e6ffed;' : '';

        return `
            <tr style="${rowStyle}">
                <td class="text-truncate" title="${scenario.feature_name || 'N/A'}">${scenario.feature_name || 'N/A'}</td>
                <td class="text-truncate" title="${scenario.name || 'N/A'}">
                    <strong style="cursor: pointer; color: #3b82f6; text-decoration: underline;" onclick="navigateToMockAPIs('${scenario.feature_name}', '${scenario.name}')">${scenario.name || 'N/A'}</strong>
                    ${isActive ? `<span class="status-badge status-active" style="margin-left: 8px;">${t('scenario.badge.active')}</span>` : ''}
                    ${isGlobalActive ? `<span class="status-badge" style="margin-left: 8px; background-color: #9f7aea; color: white;">${t('scenario.badge.activeGlobal')}</span>` : ''}
                </td>
                <td class="text-truncate" title="${scenario.description || '-'}">${scenario.description || '-'}</td>
                <td>${formatDate(scenario.created_at)}</td>
                <td class="actions">
                    <button class="btn btn-edit" onclick='editScenario(${JSON.stringify(scenario).replace(/'/g, "&#39;")})'>${t('common.edit')}</button>
                    <button class="btn btn-delete" onclick="deleteScenario('${scenario.id}')">${t('common.delete')}</button>
                    ${!isActive && !isGlobalActive ? (filterAccountId ? `<button class="btn btn-primary" onclick="activateScenario('${scenario.id}', '${filterAccountId}', '${activeScenarioId}')">${t('scenario.btn.activateForAccount')} ${filterAccountId}</button>` : `<button class="btn btn-primary" onclick="activateScenarioGlobal('${scenario.id}', '${globalActiveScenarioId}')">${t('scenario.btn.activateGlobally')}</button>`) : ''}
                </td>
            </tr>
        `;
    }).join('');
}

async function loadMockAPIs(scenarioName, page = 1, searchQuery = '') {
    if (!scenarioName) {
        document.getElementById('mockapis-table-body').innerHTML =
            `<tr><td colspan="8" class="loading">${t('mockapi.loading')}</td></tr>`;
        document.getElementById('mockapis-pagination').innerHTML = '';
        return;
    }

    try {
        let url;
        if (searchQuery && searchQuery.trim() !== '') {
            // Use search endpoint
            url = `${API_BASE_URL}/mockapis/search?scenario_name=${encodeURIComponent(scenarioName)}&q=${encodeURIComponent(searchQuery)}&page=${page}&page_size=10`;
        } else {
            // Use regular list endpoint
            url = `${API_BASE_URL}/mockapis?scenario_name=${encodeURIComponent(scenarioName)}&page=${page}&page_size=10`;
        }

        const response = await fetch(url);
        if (!response.ok) throw new Error('Failed to load mock APIs');

        const result = await response.json();
        mockAPIs = result.data;
        pagination.mockapis = { page: result.page, totalPages: result.total_pages };
        renderMockAPIsTable();
        renderPagination('mockapis', result.page, result.total_pages);
    } catch (error) {
        console.error('Error loading mock APIs:', error);
        showError(t('mockapi.error.load'));
    }
}

function renderMockAPIsTable() {
    const tbody = document.getElementById('mockapis-table-body');

    if (!mockAPIs || mockAPIs.length === 0) {
        tbody.innerHTML = `<tr><td colspan="9" class="loading">${t('mockapi.noResults')}</td></tr>`;
        return;
    }

    tbody.innerHTML = mockAPIs.map(api => `
        <tr>
            <td class="text-truncate" title="${api.feature_name || 'N/A'}">${api.feature_name || 'N/A'}</td>
            <td class="text-truncate" title="${api.scenario_name || 'N/A'}">${api.scenario_name || 'N/A'}</td>
            <td class="text-truncate" title="${api.name || 'N/A'}"><strong>${api.name || 'N/A'}</strong></td>
            <td class="text-truncate" title="${api.description || '-'}">${api.description || '-'}</td>
            <td><span class="status-badge" style="background-color: #4299e1; color: white;">${api.method || 'GET'}</span></td>
            <td class="text-truncate" title="${api.path || 'N/A'}"><code>${api.path || 'N/A'}</code></td>
            <td><span class="status-badge ${api.is_active ? 'status-active' : 'status-inactive'}">
                ${api.is_active ? t('common.active') : t('common.inactive')}
            </span></td>
            <td>${formatDate(api.created_at)}</td>
            <td class="actions">
                <button class="btn btn-edit" onclick='editMockAPI(${JSON.stringify(api)})'>${t('common.edit')}</button>
                <button class="btn btn-duplicate" onclick='duplicateMockAPI(${JSON.stringify(api)})'>${t('common.duplicate')}</button>
                <button class="btn btn-delete" onclick="deleteMockAPI('${api.id}')">${t('common.delete')}</button>
            </td>
        </tr>
    `).join('');
}

function renderScenariosTable() {
    const tbody = document.getElementById('scenarios-table-body');

    if (!scenarios || scenarios.length === 0) {
        tbody.innerHTML = `<tr><td colspan="6" class="loading">${t('scenario.noResults')}</td></tr>`;
        return;
    }

    // Get the accountId filter value
    const filterAccountId = document.getElementById('scenario-account-id-filter')?.value.trim();

    tbody.innerHTML = scenarios.map(scenario => {

        // Check if this scenario is the active one for the current account
        const isActive = activeScenarioId && scenario.id === activeScenarioId;

        // Check if this scenario is the global active one
        const isGlobalActive = globalActiveScenarioId && scenario.id === globalActiveScenarioId;

        // Highlight row if active or global active
        const rowStyle = (isActive || isGlobalActive)
            ? 'background-color: #e6ffed;'
            : '';

        /**
         * CORE FIX LOGIC
         *
         * Case 1:
         * globalActiveScenarioId != activeScenarioId
         * → show button on ACTIVE GLOBAL row
         *
         * Case 2:
         * globalActiveScenarioId == activeScenarioId
         * → show button on other rows
         */
        const shouldShowActiveForAccount =
            filterAccountId &&
            (
                (globalActiveScenarioId !== activeScenarioId && scenario.id === globalActiveScenarioId)
                ||
                (globalActiveScenarioId === activeScenarioId && scenario.id !== globalActiveScenarioId)
            );

        return `
            <tr style="${rowStyle}">
                <td class="text-truncate" title="${scenario.feature_name || 'N/A'}">
                    ${scenario.feature_name || 'N/A'}
                </td>

                <td class="text-truncate" title="${scenario.name || 'N/A'}">

                    <strong
                        style="cursor:pointer;color:#3b82f6;text-decoration:underline;"
                        onclick="navigateToMockAPIs('${scenario.feature_name}', '${scenario.name}')"
                    >
                        ${scenario.name || 'N/A'}
                    </strong>

                    ${isActive
                        ? `<span class="status-badge status-active" style="margin-left:8px;">${t('scenario.badge.active')}</span>`
                        : ''
                    }

                    ${isGlobalActive
                        ? `<span class="status-badge" style="margin-left:8px;background-color:#9f7aea;color:white;">${t('scenario.badge.activeGlobal')}</span>`
                        : ''
                    }

                </td>

                <td class="text-truncate" title="${scenario.description || '-'}">
                    ${scenario.description || '-'}
                </td>

                <td>
                    ${formatDate(scenario.created_at)}
                </td>

                <td class="actions">

                    <button
                        class="btn btn-edit"
                        onclick='editScenario(${JSON.stringify(scenario).replace(/'/g, "&#39;")})'
                    >
                        ${t('common.edit')}
                    </button>

                    <button
                        class="btn btn-delete"
                        onclick="deleteScenario('${scenario.id}')"
                    >
                        ${t('common.delete')}
                    </button>

                    ${
                        shouldShowActiveForAccount
                        ? `<button
                                class="btn btn-primary"
                                onclick="activateScenario('${scenario.id}', '${filterAccountId}', '${activeScenarioId}')"
                           >
                                ${t('scenario.btn.activateForAccount')} ${filterAccountId}
                           </button>`
                        : (!filterAccountId && !isGlobalActive
                            ? `<button
                                    class="btn btn-primary"
                                    onclick="activateScenarioGlobal('${scenario.id}', '${globalActiveScenarioId}')"
                               >
                                    ${t('scenario.btn.activateGlobally')}
                               </button>`
                            : ''
                        )
                    }

                </td>
            </tr>
        `;

    }).join('');
}


async function fetchAllFeatures() {
    try {
        const response = await fetch(`${API_BASE_URL}/features?page_size=100`);
        if (!response.ok) throw new Error('Failed to load features');
        const result = await response.json();
        return result.data;
    } catch (error) {
        console.error('Error fetching all features:', error);
        return [];
    }
}

async function fetchAllScenarios(featureName) {
    if (!featureName) return [];
    try {
        const response = await fetch(`${API_BASE_URL}/scenarios?feature_name=${featureName}&page_size=100`);
        if (!response.ok) throw new Error('Failed to load scenarios');
        const result = await response.json();
        return result.data;
    } catch (error) {
        console.error('Error fetching all scenarios:', error);
        return [];
    }
}

function renderPagination(entity, currentPage, totalPages) {
    const container = document.getElementById(`${entity}-pagination`);
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }

    let html = '<div class="pagination">';

    // Previous button
    html += `<button class="page-btn" ${currentPage === 1 ? 'disabled' : ''} onclick="loadPage('${entity}', ${currentPage - 1})">${t('pagination.prev')}</button>`;

    // Page number range
    const range = 2;
    const start = Math.max(1, currentPage - range);
    const end = Math.min(totalPages, currentPage + range);

    if (start > 1) {
        html += `<button class="page-btn" onclick="loadPage('${entity}', 1)">1</button>`;
        if (start > 2) html += '<span class="page-ellipsis">...</span>';
    }

    for (let i = start; i <= end; i++) {
        html += `<button class="page-btn ${i === currentPage ? 'active' : ''}" onclick="loadPage('${entity}', ${i})">${i}</button>`;
    }

    if (end < totalPages) {
        if (end < totalPages - 1) html += '<span class="page-ellipsis">...</span>';
        html += `<button class="page-btn" onclick="loadPage('${entity}', ${totalPages})">${totalPages}</button>`;
    }

    // Next button
    html += `<button class="page-btn" ${currentPage === totalPages ? 'disabled' : ''} onclick="loadPage('${entity}', ${currentPage + 1})">${t('pagination.next')}</button>`;

    html += '</div>';
    container.innerHTML = html;
}

function loadPage(entity, page) {
    if (entity === 'features') {
        const searchInput = document.getElementById('feature-search-query');
        const searchQuery = searchInput ? searchInput.value.trim() : '';
        loadFeatures(page, searchQuery);
    } else if (entity === 'scenarios') {
        const featureInput = document.getElementById('scenario-feature-filter');
        const featureName = featureInput.dataset.selectedValue || featureInput.value;
        loadScenarios(featureName, page);
    } else if (entity === 'mockapis') {
        const scenarioInput = document.getElementById('mockapi-scenario-filter');
        const scenarioName = scenarioInput.dataset.selectedValue || scenarioInput.value;
        const searchInput = document.getElementById('mockapi-search-query');
        const searchQuery = searchInput ? searchInput.value.trim() : '';
        loadMockAPIs(scenarioName, page, searchQuery);
    } else if (entity === 'loadtest') {
        const searchInput = document.getElementById('loadtest-search-query');
        const searchQuery = searchInput ? searchInput.value.trim() : '';
        loadLoadTestScenarios(page, searchQuery);
    }
}

async function populateFeatureFilters() {
    const scenarioFilter = document.getElementById('scenario-feature-filter');

    // Check if searchable select is already initialized
    if (!scenarioFilter.classList.contains('searchable-select-input')) {
        scenarioFilter.placeholder = 'Search features...';
        createSearchableSelect('scenario-feature-filter', {
            searchType: 'feature',
            placeholder: 'Search features...',
            onChange: (featureName) => {
                loadScenarios(featureName);
            }
        });
    }

    // Add event listener to accountId filter to re-render table when it changes
    const accountIdFilter = document.getElementById('scenario-account-id-filter');
    accountIdFilter.addEventListener('input', async () => {
        const accountId = accountIdFilter.value.trim();
        const featureName = scenarioFilter.dataset.selectedValue;

        if (featureName && accountId) {
            await fetchActiveScenario(featureName, accountId);
        } else {
            activeScenarioId = null;
        }

        renderScenariosTable();
    });
}

async function populateMockAPIFilters() {
    const featureFilter = document.getElementById('mockapi-feature-filter');
    const scenarioFilter = document.getElementById('mockapi-scenario-filter');

    // Initialize feature searchable select
    if (!featureFilter.classList.contains('searchable-select-input')) {
        featureFilter.placeholder = 'Search features...';
        createSearchableSelect('mockapi-feature-filter', {
            searchType: 'feature',
            placeholder: 'Search features...',
            onChange: async (featureName) => {
                // Update scenario filter to search within this feature
                scenarioFilter.dataset.featureName = featureName;
                scenarioFilter.value = '';
                scenarioFilter.dataset.selectedValue = '';
                scenarioFilter.placeholder = 'Search scenarios...';

                // Clear mock APIs
                document.getElementById('mockapis-table-body').innerHTML =
                    `<tr><td colspan="8" class="loading">${t('mockapi.loading')}</td></tr>`;
            }
        });
    }

    // Initialize scenario searchable select
    if (!scenarioFilter.classList.contains('searchable-select-input')) {
        scenarioFilter.placeholder = 'Search scenarios...';
        createSearchableSelect('mockapi-scenario-filter', {
            searchType: 'scenario',
            placeholder: 'Search scenarios...',
            onChange: (scenarioName) => {
                loadMockAPIs(scenarioName);
            }
        });
    }

    // Add search query input handler (Enter key support)
    const searchInput = document.getElementById('mockapi-search-query');
    if (searchInput && !searchInput.dataset.listenerAdded) {
        searchInput.dataset.listenerAdded = 'true';

        // Trigger search on Enter key
        searchInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                searchMockAPIs();
            }
        });
    }
}

// Search Mock APIs by name or path
function searchMockAPIs() {
    const searchInput = document.getElementById('mockapi-search-query');
    const scenarioInput = document.getElementById('mockapi-scenario-filter');
    const scenarioName = scenarioInput.dataset.selectedValue || scenarioInput.value;
    const query = searchInput.value.trim();

    if (!scenarioName) {
        showError(t('search.error.noScenario'));
        return;
    }

    if (!query) {
        showError(t('search.error.noTerm'));
        return;
    }

    loadMockAPIs(scenarioName, 1, query);
}

// Clear Mock API search
function clearMockAPISearch() {
    const searchInput = document.getElementById('mockapi-search-query');
    const scenarioInput = document.getElementById('mockapi-scenario-filter');
    const scenarioName = scenarioInput.dataset.selectedValue || scenarioInput.value;

    // Clear the search input
    searchInput.value = '';

    // Reload all Mock APIs for the current scenario
    if (scenarioName) {
        loadMockAPIs(scenarioName, 1, '');
    }
}

// Search Load Test Scenarios by name
function searchLoadTestScenarios() {
    const searchInput = document.getElementById('loadtest-search-query');
    const query = searchInput.value.trim();

    if (!query) {
        showError(t('search.error.noTerm'));
        return;
    }

    loadLoadTestScenarios(1, query);
}

// Clear Load Test Scenario search
function clearLoadTestSearch() {
    const searchInput = document.getElementById('loadtest-search-query');

    // Clear the search input
    searchInput.value = '';

    // Reload all load test scenarios
    loadLoadTestScenarios(1, '');
}

// Search Features by name
function searchFeatures() {
    const searchInput = document.getElementById('feature-search-query');
    const query = searchInput.value.trim();

    if (!query) {
        showError(t('search.error.noTerm'));
        return;
    }

    loadFeatures(1, query);
}

// Clear Feature search
function clearFeatureSearch() {
    const searchInput = document.getElementById('feature-search-query');

    // Clear the search input
    searchInput.value = '';

    // Reload all features
    loadFeatures(1, '');
}

function showCreateFeatureModal() {
    document.getElementById('feature-modal-title').textContent = t('feature.modal.create');
    document.getElementById('feature-id').value = '';
    document.getElementById('feature-name').value = '';
    document.getElementById('feature-description').value = '';
    document.getElementById('feature-active').checked = true;
    showModal('feature-modal');
}

function editFeature(feature) {
    document.getElementById('feature-modal-title').textContent = t('feature.modal.edit');
    document.getElementById('feature-id').value = feature.id;
    document.getElementById('feature-name').value = feature.name;
    document.getElementById('feature-description').value = feature.description || '';
    document.getElementById('feature-active').checked = feature.is_active;
    showModal('feature-modal');
}

async function saveFeature() {
    const id = document.getElementById('feature-id').value;
    const name = document.getElementById('feature-name').value.trim();
    const description = document.getElementById('feature-description').value.trim();
    const isActive = document.getElementById('feature-active').checked;

    // Clear previous errors
    clearFieldError('feature-name');

    if (!name) {
        showFieldError('feature-name', t('feature.error.nameRequired'));
        return;
    }

    if (name.includes(' ')) {
        showFieldError('feature-name', t('feature.error.nameHasSpaces'));
        return;
    }

    if (name.length > 100) {
        showFieldError('feature-name', t('feature.error.nameTooLong'));
        return;
    }

    var data = {
        description: description,
        is_active: isActive
    };
    if (id != null) {
        data.name = name
    }
    try {
        const url = id ? `${API_BASE_URL}/features/${id}` : `${API_BASE_URL}/features`;
        const method = id ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        if (!response.ok) throw new Error('Failed to save feature');

        closeModal('feature-modal');
        showSuccess(id ? t('feature.success.updated') : t('feature.success.created'));
        loadFeatures();
    } catch (error) {
        console.error('Error saving feature:', error);
        showError(t('feature.error.save'));
    }
}

async function showCreateScenarioModal() {
    document.getElementById('scenario-modal-title').textContent = t('scenario.modal.create');
    document.getElementById('scenario-id').value = '';
    document.getElementById('scenario-name').value = '';
    document.getElementById('scenario-description').value = '';

    const featureSelect = document.getElementById('scenario-feature');

    // Initialize searchable select if not already done
    if (!featureSelect.classList.contains('searchable-select-input')) {
        featureSelect.placeholder = 'Search features...';
        createSearchableSelect('scenario-feature', {
            searchType: 'feature',
            placeholder: 'Search features...'
        });
    } else {
        // Reset the input
        featureSelect.value = '';
        featureSelect.dataset.selectedValue = '';
    }

    showModal('scenario-modal');
}

async function editScenario(scenario) {
    document.getElementById('scenario-modal-title').textContent = t('scenario.modal.edit');
    document.getElementById('scenario-id').value = scenario.id;
    document.getElementById('scenario-name').value = scenario.name;
    document.getElementById('scenario-description').value = scenario.description || '';

    const featureSelect = document.getElementById('scenario-feature');

    // Initialize searchable select if not already done
    if (!featureSelect.classList.contains('searchable-select-input')) {
        featureSelect.placeholder = 'Search features...';
        createSearchableSelect('scenario-feature', {
            searchType: 'feature',
            placeholder: 'Search features...'
        });
    }

    // Set the selected value
    featureSelect.value = scenario.feature_name;
    featureSelect.dataset.selectedValue = scenario.feature_name;
    featureSelect.dataset.selectedText = scenario.feature_name;

    showModal('scenario-modal');
}

async function saveScenario() {
    const id = document.getElementById('scenario-id').value;
    const featureInput = document.getElementById('scenario-feature');
    const featureName = featureInput.dataset.selectedValue || featureInput.value;
    const name = document.getElementById('scenario-name').value.trim();
    const description = document.getElementById('scenario-description').value.trim();

    // Clear previous errors
    clearFieldError('scenario-feature');
    clearFieldError('scenario-name');

    if (!featureName) {
        showFieldError('scenario-feature', t('scenario.error.featureRequired'));
        return;
    }

    if (!name) {
        showFieldError('scenario-name', t('scenario.error.nameRequired'));
        return;
    }

    if (name.includes(' ')) {
        showFieldError('scenario-name', t('scenario.error.nameHasSpaces'));
        return;
    }

    if (name.length > 100) {
        showFieldError('scenario-name', t('scenario.error.nameTooLong'));
        return;
    }
    const data = {
        feature_name: featureName,
        description: description
    };
    if (id != null) {
        data.name = name
    }
    try {
        const url = id ? `${API_BASE_URL}/scenarios/${id}` : `${API_BASE_URL}/scenarios`;
        const method = id ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        if (!response.ok) throw new Error('Failed to save scenario');

        closeModal('scenario-modal');
        showSuccess(id ? t('scenario.success.updated') : t('scenario.success.created'));

        const featureInput = document.getElementById('scenario-feature-filter');
        const currentFeature = featureInput.dataset.selectedValue || featureInput.value;
        if (currentFeature) {
            loadScenarios(currentFeature);
        }
    } catch (error) {
        console.error('Error saving scenario:', error);
        showError(t('scenario.error.save'));
    }
}

async function activateScenario(scenarioId, accountId, prevScenarioId) {
    try {
        const data = {
            prev_scenario_id: prevScenarioId,
        };
        const url = `${API_BASE_URL}/scenarios/${scenarioId}/activate?account_id=${encodeURIComponent(accountId)}`;
        const response = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        if (!response.ok) throw new Error('Failed to activate scenario');

        showSuccess(t('scenario.success.activated', accountId));

        // Update active scenario ID and reload
        activeScenarioId = scenarioId;
        const featureInput = document.getElementById('scenario-feature-filter');
        const currentFeature = featureInput.dataset.selectedValue || featureInput.value;
        if (currentFeature) {
            loadScenarios(currentFeature);
        }
    } catch (error) {
        console.error('Error activating scenario:', error);
        showError(t('scenario.error.activate'));
    }
}

async function activateScenarioGlobal(scenarioId, prevScenarioId) {
    try {
        const data = {
            prev_scenario_id: prevScenarioId,
        };
        const url = `${API_BASE_URL}/scenarios/${scenarioId}/activate`;
        const response = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        if (!response.ok) throw new Error('Failed to activate scenario');

        showSuccess(t('scenario.success.activatedGlobal'));

        // Clear active scenario ID since this is a global activation
        activeScenarioId = null;
        const featureInput = document.getElementById('scenario-feature-filter');
        const currentFeature = featureInput.dataset.selectedValue || featureInput.value;
        if (currentFeature) {
            loadScenarios(currentFeature);
        }
    } catch (error) {
        console.error('Error activating scenario:', error);
        showError(t('scenario.error.activate'));
    }
}

async function showCreateMockAPIModal() {
    document.getElementById('mockapi-modal-title').textContent = t('mockapi.modal.create');
    document.getElementById('mockapi-id').value = '';
    document.getElementById('mockapi-name').value = '';
    document.getElementById('mockapi-description').value = '';
    document.getElementById('mockapi-path').value = '';
    document.getElementById('mockapi-method').value = 'GET';
    document.getElementById('mockapi-regex-path').value = '';
    document.getElementById('mockapi-hash-input').value = '';
    document.getElementById('mockapi-output').value = '';
    document.getElementById('mockapi-latency').value = '0';
    document.getElementById('mockapi-active').checked = true;

    const featureSelect = document.getElementById('mockapi-feature');
    const scenarioSelect = document.getElementById('mockapi-scenario');

    // Initialize feature searchable select if not already done
    if (!featureSelect.classList.contains('searchable-select-input')) {
        featureSelect.placeholder = 'Search features...';
        createSearchableSelect('mockapi-feature', {
            searchType: 'feature',
            placeholder: 'Search features...',
            onChange: (featureName) => {
                // Update scenario filter
                scenarioSelect.dataset.featureName = featureName;
                scenarioSelect.value = '';
                scenarioSelect.dataset.selectedValue = '';
                scenarioSelect.placeholder = 'Search scenarios...';
            }
        });
    } else {
        featureSelect.value = '';
        featureSelect.dataset.selectedValue = '';
    }

    // Initialize scenario searchable select if not already done
    if (!scenarioSelect.classList.contains('searchable-select-input')) {
        scenarioSelect.placeholder = 'Search scenarios...';
        createSearchableSelect('mockapi-scenario', {
            searchType: 'scenario',
            placeholder: 'Search scenarios...'
        });
    } else {
        scenarioSelect.value = '';
        scenarioSelect.dataset.selectedValue = '';
        scenarioSelect.dataset.featureName = '';
    }

    resetJsonTreeMode('mockapi-hash-input');
    resetJsonTreeMode('mockapi-output');
    showModal('mockapi-modal');
}

async function editMockAPI(api) {
    document.getElementById('mockapi-modal-title').textContent = t('mockapi.modal.edit');
    document.getElementById('mockapi-id').value = api.id;
    document.getElementById('mockapi-name').value = api.name;
    document.getElementById('mockapi-description').value = api.description || '';
    document.getElementById('mockapi-path').value = api.path;
    document.getElementById('mockapi-method').value = api.method || 'GET';
    document.getElementById('mockapi-regex-path').value = api.regex_path || '';

    // Populate input if it exists
    try {
        if (api.input && api.input !== null) {
            document.getElementById('mockapi-hash-input').value =
                typeof api.input === 'string' ? api.input : JSON.stringify(api.input, null, 2);
        } else {
            document.getElementById('mockapi-hash-input').value = '';
        }
    } catch (e) {
        document.getElementById('mockapi-hash-input').value = '';
    }

    // Populate output
    try {
        document.getElementById('mockapi-output').value =
            typeof api.output === 'string' ? api.output : JSON.stringify(api.output, null, 2);
    } catch (e) {
        document.getElementById('mockapi-output').value = '';
    }

    // Populate headers if they exist
    try {
        if (api.headers && api.headers !== null) {
            let headerStr = typeof api.headers === 'string' ? api.headers : JSON.stringify(api.headers, null, 2);
            // Remove outer braces and trim
            headerStr = headerStr.replace(/^\{\n?/, '').replace(/\n?\}$/, '').trim();
            document.getElementById('mockapi-output-header').value = headerStr;
        } else {
            document.getElementById('mockapi-output-header').value = '';
        }
    } catch (e) {
        document.getElementById('mockapi-output-header').value = '';
    }

    document.getElementById('mockapi-latency').value = api.latency || 0;
    document.getElementById('mockapi-active').checked = api.is_active;

    const featureSelect = document.getElementById('mockapi-feature');
    const scenarioSelect = document.getElementById('mockapi-scenario');

    // Initialize feature searchable select if not already done
    if (!featureSelect.classList.contains('searchable-select-input')) {
        featureSelect.placeholder = 'Search features...';
        createSearchableSelect('mockapi-feature', {
            searchType: 'feature',
            placeholder: 'Search features...',
            onChange: (featureName) => {
                scenarioSelect.dataset.featureName = featureName;
                scenarioSelect.value = '';
                scenarioSelect.dataset.selectedValue = '';
            }
        });
    }

    // Set feature value
    featureSelect.value = api.feature_name;
    featureSelect.dataset.selectedValue = api.feature_name;
    featureSelect.dataset.selectedText = api.feature_name;

    // Initialize scenario searchable select if not already done
    if (!scenarioSelect.classList.contains('searchable-select-input')) {
        scenarioSelect.placeholder = 'Search scenarios...';
        createSearchableSelect('mockapi-scenario', {
            searchType: 'scenario',
            placeholder: 'Search scenarios...'
        });
    }

    // Set scenario value
    scenarioSelect.dataset.featureName = api.feature_name;
    scenarioSelect.value = api.scenario_name;
    scenarioSelect.dataset.selectedValue = api.scenario_name;
    scenarioSelect.dataset.selectedText = api.scenario_name;

    resetJsonTreeMode('mockapi-hash-input');
    resetJsonTreeMode('mockapi-output');
    showModal('mockapi-modal');
}

async function duplicateMockAPI(api) {
    document.getElementById('mockapi-modal-title').textContent = t('mockapi.modal.duplicate');

    // Clear ID so it creates a new record
    document.getElementById('mockapi-id').value = '';

    // Generate versioned name
    let newName = api.name;
    const versionRegex = / v(\d+)$/;
    const match = newName.match(versionRegex);

    if (match) {
        // Increment existing version number
        const currentVersion = parseInt(match[1]);
        newName = newName.replace(versionRegex, ` v${currentVersion + 1}`);
    } else {
        // Append v2 for first duplicate
        newName = `${newName} v2`;
    }

    document.getElementById('mockapi-name').value = newName;
    document.getElementById('mockapi-description').value = api.description || '';
    document.getElementById('mockapi-path').value = api.path;
    document.getElementById('mockapi-method').value = api.method || 'GET';
    document.getElementById('mockapi-regex-path').value = api.regex_path || '';

    // Populate input if it exists
    try {
        if (api.input && api.input !== null) {
            document.getElementById('mockapi-hash-input').value =
                typeof api.input === 'string' ? api.input : JSON.stringify(api.input, null, 2);
        } else {
            document.getElementById('mockapi-hash-input').value = '';
        }
    } catch (e) {
        document.getElementById('mockapi-hash-input').value = '';
    }

    // Populate output
    try {
        document.getElementById('mockapi-output').value =
            typeof api.output === 'string' ? api.output : JSON.stringify(api.output, null, 2);
    } catch (e) {
        document.getElementById('mockapi-output').value = '';
    }

    // Populate headers if they exist
    try {
        if (api.headers && api.headers !== null) {
            let headerStr = typeof api.headers === 'string' ? api.headers : JSON.stringify(api.headers, null, 2);
            // Remove outer braces and trim
            headerStr = headerStr.replace(/^\{\n?/, '').replace(/\n?\}$/, '').trim();
            document.getElementById('mockapi-output-header').value = headerStr;
        } else {
            document.getElementById('mockapi-output-header').value = '';
        }
    } catch (e) {
        document.getElementById('mockapi-output-header').value = '';
    }

    document.getElementById('mockapi-latency').value = api.latency || 0;
    document.getElementById('mockapi-active').checked = api.is_active;

    const featureSelect = document.getElementById('mockapi-feature');
    const scenarioSelect = document.getElementById('mockapi-scenario');

    // Initialize feature searchable select if not already done
    if (!featureSelect.classList.contains('searchable-select-input')) {
        featureSelect.placeholder = 'Search features...';
        createSearchableSelect('mockapi-feature', {
            searchType: 'feature',
            placeholder: 'Search features...',
            onChange: (featureName) => {
                scenarioSelect.dataset.featureName = featureName;
                scenarioSelect.value = '';
                scenarioSelect.dataset.selectedValue = '';
            }
        });
    }

    // Set feature value
    featureSelect.value = api.feature_name;
    featureSelect.dataset.selectedValue = api.feature_name;
    featureSelect.dataset.selectedText = api.feature_name;

    // Initialize scenario searchable select if not already done
    if (!scenarioSelect.classList.contains('searchable-select-input')) {
        scenarioSelect.placeholder = 'Search scenarios...';
        createSearchableSelect('mockapi-scenario', {
            searchType: 'scenario',
            placeholder: 'Search scenarios...'
        });
    }

    // Set scenario value
    scenarioSelect.dataset.featureName = api.feature_name;
    scenarioSelect.value = api.scenario_name;
    scenarioSelect.dataset.selectedValue = api.scenario_name;
    scenarioSelect.dataset.selectedText = api.scenario_name;

    resetJsonTreeMode('mockapi-hash-input');
    resetJsonTreeMode('mockapi-output');
    showModal('mockapi-modal');
}

async function saveMockAPI() {
    const id = document.getElementById('mockapi-id').value;
    const featureInput = document.getElementById('mockapi-feature');
    const scenarioInput = document.getElementById('mockapi-scenario');
    const featureName = featureInput.dataset.selectedValue || featureInput.value;
    const scenarioName = scenarioInput.dataset.selectedValue || scenarioInput.value;
    const name = document.getElementById('mockapi-name').value.trim();
    const description = document.getElementById('mockapi-description').value.trim();
    const path = document.getElementById('mockapi-path').value.trim();
    const method = document.getElementById('mockapi-method').value;
    const regexPath = document.getElementById('mockapi-regex-path').value.trim();
    const hashInputStr = getJsonFieldValue('mockapi-hash-input');
    const outputStr = getJsonFieldValue('mockapi-output');
    const outputHeaders = document.getElementById('mockapi-output-header').value.trim();
    const latency = parseInt(document.getElementById('mockapi-latency').value, 10) || 0;
    const isActive = document.getElementById('mockapi-active').checked;

    // Clear previous errors
    clearFieldError('mockapi-feature');
    clearFieldError('mockapi-scenario');
    clearFieldError('mockapi-name');
    clearFieldError('mockapi-path');
    clearFieldError('mockapi-output');

    if (!featureName) {
        showFieldError('mockapi-feature', t('mockapi.error.featureRequired'));
        return;
    }

    if (!scenarioName) {
        showFieldError('mockapi-scenario', t('mockapi.error.scenarioRequired'));
        return;
    }

    if (!name) {
        showFieldError('mockapi-name', t('mockapi.error.nameRequired'));
        return;
    }

    if (name.includes(' ')) {
        showFieldError('mockapi-name', t('mockapi.error.nameHasSpaces'));
        return;
    }

    if (name.length > 100) {
        showFieldError('mockapi-name', t('mockapi.error.nameTooLong'));
        return;
    }

    if (!path) {
        showFieldError('mockapi-path', t('mockapi.error.pathRequired'));
        return;
    }

    if (!method || !outputStr) {
        showError(t('mockapi.error.methodOutputRequired'));
        return;
    }

    // Parse output JSON - will be automatically minified when sent to server
    let output;
    try {
        output = JSON.parse(outputStr);
    } catch (e) {
        showError(t('mockapi.error.invalidOutput'));
        return;
    }

    // Parse input JSON if provided - will be automatically minified when sent to server
    let input = null;
    if (hashInputStr) {
        try {
            input = JSON.parse(hashInputStr);
        } catch (e) {
            showError(t('mockapi.error.invalidInput'));
            return;
        }
    }

    const data = {
        feature_name: featureName,
        scenario_name: scenarioName,
        name: name,
        description: description,
        path: path,
        method: method,
        regex_path: regexPath,
        input: input,
        output: output,
        headers: outputHeaders,
        latency: latency,
        is_active: isActive
    };

    try {
        const url = id ? `${API_BASE_URL}/mockapis/${id}` : `${API_BASE_URL}/mockapis`;
        const method = id ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        if (!response.ok) throw new Error('Failed to save mock API');

        closeModal('mockapi-modal');
        showSuccess(id ? t('mockapi.success.updated') : t('mockapi.success.created'));

        const scenarioInput = document.getElementById('mockapi-scenario-filter');
        const currentScenario = scenarioInput.dataset.selectedValue || scenarioInput.value;
        if (currentScenario) {
            loadMockAPIs(currentScenario);
        }
    } catch (error) {
        console.error('Error saving mock API:', error);
        showError(t('mockapi.error.save'));
    }
}

function showModal(modalId) {
    document.getElementById(modalId).classList.add('show');
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('show');
}

function formatDate(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
}

function showError(message) {
    alert(message);
}

function showSuccess(message) {
    alert(message);
}

function showFieldError(fieldId, message) {
    // Remove any existing error message for this field
    clearFieldError(fieldId);

    const field = document.getElementById(fieldId);
    if (!field) return;

    // Create error message element
    const errorDiv = document.createElement('div');
    errorDiv.className = 'field-error-message';
    errorDiv.id = `${fieldId}-error`;
    errorDiv.textContent = message;
    errorDiv.style.color = '#e53e3e';
    errorDiv.style.fontSize = '0.875rem';
    errorDiv.style.marginTop = '0.25rem';
    errorDiv.style.marginBottom = '0.5rem';

    // Add red border to the input field
    field.style.borderColor = '#e53e3e';

    // Insert error message after the field
    field.parentNode.insertBefore(errorDiv, field.nextSibling);
}

function clearFieldError(fieldId) {
    const field = document.getElementById(fieldId);
    if (!field) return;

    // Remove error message
    const errorDiv = document.getElementById(`${fieldId}-error`);
    if (errorDiv) {
        errorDiv.remove();
    }

    // Reset border color
    field.style.borderColor = '';
}

function clearAllFieldErrors() {
    const errorMessages = document.querySelectorAll('.field-error-message');
    errorMessages.forEach(error => error.remove());

    const inputs = document.querySelectorAll('input[style*="border-color"]');
    inputs.forEach(input => input.style.borderColor = '');
}

// Add event listeners to clear errors when user starts typing
function setupFieldErrorClearOnInput(fieldIds) {
    fieldIds.forEach(fieldId => {
        const field = document.getElementById(fieldId);
        if (field) {
            field.addEventListener('input', function() {
                clearFieldError(fieldId);
            });
        }
    });
}

// Setup error clearing on page load
document.addEventListener('DOMContentLoaded', function() {
    setupFieldErrorClearOnInput([
        'feature-name',
        'scenario-name',
        'mockapi-name',
        'loadtest-name'
    ]);
});

async function deleteFeature(id) {
    if (!confirm(t('feature.confirm.delete'))) return;

    try {
        const response = await fetch(`${API_BASE_URL}/features/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete feature');

        showSuccess(t('feature.success.deleted'));
        loadFeatures();
    } catch (error) {
        console.error('Error deleting feature:', error);
        showError(t('feature.error.delete'));
    }
}

async function deleteScenario(id) {
    if (!confirm(t('scenario.confirm.delete'))) return;

    try {
        const response = await fetch(`${API_BASE_URL}/scenarios/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete scenario');

        showSuccess(t('scenario.success.deleted'));
        const featureInput = document.getElementById('scenario-feature-filter');
        const currentFeature = featureInput.dataset.selectedValue || featureInput.value;
        if (currentFeature) {
            loadScenarios(currentFeature);
        }
    } catch (error) {
        console.error('Error deleting scenario:', error);
        showError(t('scenario.error.delete'));
    }
}

async function deleteMockAPI(id) {
    if (!confirm(t('mockapi.confirm.delete'))) return;

    try {
        const response = await fetch(`${API_BASE_URL}/mockapis/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete mock API');

        showSuccess(t('mockapi.success.deleted'));
        const scenarioInput = document.getElementById('mockapi-scenario-filter');
        const currentScenario = scenarioInput.dataset.selectedValue || scenarioInput.value;
        if (currentScenario) {
            loadMockAPIs(currentScenario);
        }
    } catch (error) {
        console.error('Error deleting mock API:', error);
        showError(t('mockapi.error.delete'));
    }
}

// === Proto Parser ===

function initProtoFileInput() {
    document.getElementById('proto-file-input').addEventListener('change', function(e) {
        const file = e.target.files[0];
        if (!file) return;
        const reader = new FileReader();
        reader.onload = function(event) {
            document.getElementById('proto-text-input').value = event.target.result;
        };
        reader.readAsText(file);
    });
}

function showProtoParserModal() {
    document.getElementById('proto-file-input').value = '';
    document.getElementById('proto-text-input').value = '';
    document.getElementById('proto-results').style.display = 'none';
    document.getElementById('proto-results').innerHTML = '';
    selectedProtoInput = null;
    selectedProtoOutput = null;
    showModal('proto-parser-modal');
}

function parseProtoInput() {
    const text = document.getElementById('proto-text-input').value.trim();
    if (!text) {
        showError(t('proto.error.empty'));
        return;
    }

    parsedEnums = {};
    parsedMessages = {};

    const cleaned = text
        .replace(/\/\/.*$/gm, '')
        .replace(/\/\*[\s\S]*?\*\//g, '');

    extractEnums(cleaned);
    extractMessages(cleaned);
    renderProtoResults();
}

function findMatchingBrace(text, start) {
    let depth = 1;
    let i = start;
    while (i < text.length && depth > 0) {
        if (text[i] === '{') depth++;
        if (text[i] === '}') depth--;
        i++;
    }
    return i;
}

function extractEnums(text) {
    const regex = /\benum\s+(\w+)\s*\{/g;
    let match;
    while ((match = regex.exec(text)) !== null) {
        const bodyStart = match.index + match[0].length;
        const bodyEnd = findMatchingBrace(text, bodyStart);
        const body = text.substring(bodyStart, bodyEnd - 1);

        const values = [];
        const valRegex = /(\w+)\s*=\s*(\d+)/g;
        let valMatch;
        while ((valMatch = valRegex.exec(body)) !== null) {
            values.push({ name: valMatch[1], number: parseInt(valMatch[2]) });
        }
        values.sort((a, b) => a.number - b.number);
        if (values.length > 0) {
            parsedEnums[match[1]] = values;
        }
    }
}

function extractMessages(text) {
    const regex = /\bmessage\s+(\w+)\s*\{/g;
    let match;
    while ((match = regex.exec(text)) !== null) {
        const bodyStart = match.index + match[0].length;
        const bodyEnd = findMatchingBrace(text, bodyStart);
        const body = text.substring(bodyStart, bodyEnd - 1);

        parsedMessages[match[1]] = parseFields(getTopLevelContent(body));
    }
}

// Strips nested message/enum blocks; inlines oneof field declarations
function getTopLevelContent(body) {
    let result = '';
    let i = 0;
    while (i < body.length) {
        const ahead = body.substring(i);

        const skipMatch = ahead.match(/^(message|enum)\s+\w+\s*\{/);
        if (skipMatch) {
            i += skipMatch[0].length;
            i = findMatchingBrace(body, i);
            continue;
        }

        const oneofMatch = ahead.match(/^oneof\s+\w+\s*\{/);
        if (oneofMatch) {
            i += oneofMatch[0].length;
            let depth = 1;
            while (i < body.length && depth > 0) {
                if (body[i] === '{') depth++;
                else if (body[i] === '}') depth--;
                if (depth > 0) result += body[i];
                i++;
            }
            continue;
        }

        result += body[i];
        i++;
    }
    return result;
}

function parseFields(text) {
    const fields = [];
    const regex = /(?:(repeated|optional)\s+)?(?:map\s*<\s*(\w+)\s*,\s*(\w+)\s*>|(\w+(?:\.\w+)*))\s+(\w+)\s*=\s*(\d+)/g;
    let match;
    while ((match = regex.exec(text)) !== null) {
        fields.push({
            modifier: match[1] || null,
            mapKeyType: match[2] || null,
            mapValueType: match[3] || null,
            type: match[4] || null,
            name: match[5],
            number: parseInt(match[6])
        });
    }
    return fields;
}

const _protoIntTypes = ['int32', 'int64', 'uint32', 'uint64', 'sint32', 'sint64',
    'fixed32', 'fixed64', 'sfixed32', 'sfixed64'];

const _protoWellKnown = {
    'google.protobuf.Timestamp': '2024-01-01T00:00:00Z',
    'google.protobuf.Duration': '0s',
    'google.protobuf.Struct': {},
    'google.protobuf.Value': null,
    'google.protobuf.ListValue': [],
    'google.protobuf.Any': { '@type': '' }
};

function getDefaultValue(type, depth) {
    if (type === 'string') return '';
    if (type === 'bool') return false;
    if (type === 'bytes') return '';
    if (_protoIntTypes.includes(type)) return 0;
    if (type === 'float' || type === 'double') return 0;
    if (Object.prototype.hasOwnProperty.call(_protoWellKnown, type)) {
        return JSON.parse(JSON.stringify(_protoWellKnown[type]));
    }
    if (parsedEnums[type] && parsedEnums[type].length > 0) {
        return parsedEnums[type][0].name;
    }
    if (depth < 3 && parsedMessages[type]) {
        return generateTemplate(parsedMessages[type], depth + 1);
    }
    return {};
}

function generateTemplate(fields, depth = 0) {
    const obj = {};
    for (const field of fields) {
        if (field.mapKeyType) {
            obj[field.name] = { 'key': getDefaultValue(field.mapValueType, depth) };
        } else if (field.modifier === 'repeated') {
            obj[field.name] = [getDefaultValue(field.type, depth)];
        } else {
            obj[field.name] = getDefaultValue(field.type, depth);
        }
    }
    return obj;
}

function escapeHtml(str) {
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

function renderProtoResults() {
    const container = document.getElementById('proto-results');
    const names = Object.keys(parsedMessages);

    if (names.length === 0) {
        container.innerHTML = `<p style="color: #718096; font-style: italic;">${t('proto.noMessages')}</p>`;
        container.style.display = 'block';
        return;
    }

    protoTemplates = names.map(name => ({
        name,
        json: JSON.stringify(generateTemplate(parsedMessages[name]), null, 2)
    }));

    let html = '<div class="proto-results-grid">';
    protoTemplates.forEach((item, index) => {
        html += `
            <div class="proto-result-card">
                <div class="proto-result-name">${escapeHtml(item.name)}</div>
                <pre class="proto-result-json">${escapeHtml(item.json)}</pre>
                <div class="proto-result-actions">
                    <button class="btn btn-proto-input btn-proto-select" onclick="selectProtoTemplate(${index}, 'input')">${t('proto.btn.useAsInput')}</button>
                    <button class="btn btn-proto-output btn-proto-select" onclick="selectProtoTemplate(${index}, 'output')">${t('proto.btn.useAsOutput')}</button>
                </div>
            </div>`;
    });
    html += '</div>';
    html += '<div class="proto-apply-row">';
    html += `<button class="btn btn-primary" id="proto-apply-btn" style="display: none;" onclick="applyProtoTemplates()">${t('proto.btn.createMockAPI')}</button>`;
    html += '</div>';

    container.innerHTML = html;
    container.style.display = 'block';
}

function selectProtoTemplate(index, target) {
    document.querySelectorAll(`.btn-proto-${target}`).forEach(btn => btn.classList.remove('selected'));
    document.querySelectorAll(`.btn-proto-${target}`)[index]?.classList.add('selected');

    if (target === 'input') {
        selectedProtoInput = protoTemplates[index]?.json || null;
    } else {
        selectedProtoOutput = protoTemplates[index]?.json || null;
    }

    document.getElementById('proto-apply-btn').style.display =
        (selectedProtoInput || selectedProtoOutput) ? 'inline-block' : 'none';
}

async function applyProtoTemplates() {
    closeModal('proto-parser-modal');
    await showCreateMockAPIModal();

    if (selectedProtoInput) {
        document.getElementById('mockapi-hash-input').value = selectedProtoInput;
    }
    if (selectedProtoOutput) {
        document.getElementById('mockapi-output').value = selectedProtoOutput;
    }

    selectedProtoInput = null;
    selectedProtoOutput = null;
}

// === JSON Tree Editor ===

function getJsonType(value) {
    if (value === null || value === undefined) return 'null';
    if (Array.isArray(value)) return 'array';
    return typeof value;
}

function createTreeNode(key, value, isRoot, isArrayItem, arrayIndex) {
    const type = getJsonType(value);
    const node = document.createElement('div');
    node.className = 'tree-node';
    node.dataset.type = type;

    const header = document.createElement('div');
    header.className = 'tree-node-header';

    // Expand/collapse toggle (containers) or spacer (primitives)
    if (type === 'object' || type === 'array') {
        const expandBtn = document.createElement('button');
        expandBtn.className = 'tree-expand-btn';
        expandBtn.textContent = '▼';
        expandBtn.onclick = function () {
            const ch = node.querySelector(':scope > .tree-children');
            if (ch.style.display === 'none') {
                ch.style.display = 'block';
                expandBtn.textContent = '▼';
            } else {
                ch.style.display = 'none';
                expandBtn.textContent = '▶';
            }
        };
        header.appendChild(expandBtn);
    } else {
        const spacer = document.createElement('span');
        spacer.className = 'tree-expand-spacer';
        header.appendChild(spacer);
    }

    // Key input (object property) or index badge (array item)
    if (!isRoot) {
        if (isArrayItem) {
            const badge = document.createElement('span');
            badge.className = 'tree-index-badge';
            badge.textContent = '[' + arrayIndex + ']';
            header.appendChild(badge);
        } else {
            const keyInput = document.createElement('input');
            keyInput.className = 'tree-key-input';
            keyInput.value = (key != null) ? String(key) : '';
            keyInput.placeholder = 'key';
            header.appendChild(keyInput);
        }
        const colon = document.createElement('span');
        colon.className = 'tree-colon';
        colon.textContent = ':';
        header.appendChild(colon);
    }

    // Type selector
    const typeOptions = isRoot ? ['object', 'array'] : ['string', 'number', 'boolean', 'null', 'object', 'array'];
    const typeSelect = document.createElement('select');
    typeSelect.className = 'tree-type-select';
    typeOptions.forEach(function (t) {
        const opt = document.createElement('option');
        opt.value = t;
        opt.textContent = t;
        if (t === type) opt.selected = true;
        typeSelect.appendChild(opt);
    });
    typeSelect.onchange = function (e) {
        changeNodeType(node, e.target.value, isRoot);
    };
    header.appendChild(typeSelect);

    // Value control (primitives only)
    if (type === 'string' || type === 'number') {
        const input = document.createElement('input');
        input.className = 'tree-value-input';
        input.value = String(value);
        input.placeholder = type === 'number' ? '0' : 'value';
        header.appendChild(input);
    } else if (type === 'boolean') {
        const sel = document.createElement('select');
        sel.className = 'tree-value-input';
        ['true', 'false'].forEach(function (v) {
            const opt = document.createElement('option');
            opt.value = v;
            opt.textContent = v;
            if (v === String(value)) opt.selected = true;
            sel.appendChild(opt);
        });
        header.appendChild(sel);
    } else if (type === 'null') {
        const lbl = document.createElement('span');
        lbl.className = 'tree-null-label';
        lbl.textContent = 'null';
        header.appendChild(lbl);
    }

    // Add-child button (containers only)
    if (type === 'object' || type === 'array') {
        const addBtn = document.createElement('button');
        addBtn.className = 'tree-add-btn';
        addBtn.textContent = '+';
        addBtn.title = type === 'object' ? 'Add property' : 'Add item';
        addBtn.onclick = function () { addTreeChild(node); };
        header.appendChild(addBtn);
    }

    // Duplicate button (non-root only)
    if (!isRoot) {
        const dupBtn = document.createElement('button');
        dupBtn.className = 'tree-duplicate-btn';
        dupBtn.textContent = '⧉';
        dupBtn.title = 'Duplicate';
        dupBtn.onclick = function () { duplicateTreeNode(node); };
        header.appendChild(dupBtn);
    }

    // Remove button (non-root only)
    if (!isRoot) {
        const rmBtn = document.createElement('button');
        rmBtn.className = 'tree-remove-btn';
        rmBtn.textContent = '×';
        rmBtn.title = 'Remove';
        rmBtn.onclick = function () { removeTreeNode(node); };
        header.appendChild(rmBtn);
    }

    node.appendChild(header);

    // Children container (containers only)
    if (type === 'object' || type === 'array') {
        const childrenDiv = document.createElement('div');
        childrenDiv.className = 'tree-children';
        if (type === 'object') {
            Object.keys(value).forEach(function (k) {
                childrenDiv.appendChild(createTreeNode(k, value[k], false, false, null));
            });
        } else {
            value.forEach(function (item, idx) {
                childrenDiv.appendChild(createTreeNode(null, item, false, true, idx));
            });
        }
        node.appendChild(childrenDiv);
    }

    return node;
}

function renderJsonTree(containerId, data) {
    const container = document.getElementById(containerId);
    container.innerHTML = '';
    container.appendChild(createTreeNode(null, data, true, false, null));
}

function readTreeValue(node) {
    const type = node.dataset.type;

    if (type === 'object') {
        const obj = {};
        const ch = node.querySelector(':scope > .tree-children');
        if (ch) {
            ch.querySelectorAll(':scope > .tree-node').forEach(function (child) {
                const ki = child.querySelector(':scope > .tree-node-header > .tree-key-input');
                const key = ki ? ki.value.trim() : '';
                if (key) obj[key] = readTreeValue(child);
            });
        }
        return obj;
    }

    if (type === 'array') {
        const arr = [];
        const ch = node.querySelector(':scope > .tree-children');
        if (ch) {
            ch.querySelectorAll(':scope > .tree-node').forEach(function (child) {
                arr.push(readTreeValue(child));
            });
        }
        return arr;
    }

    if (type === 'null') return null;

    const valueEl = node.querySelector(':scope > .tree-node-header > .tree-value-input');
    if (type === 'boolean') return valueEl ? valueEl.value === 'true' : false;
    if (type === 'number') {
        const n = Number(valueEl ? valueEl.value : '0');
        return isNaN(n) ? 0 : n;
    }
    return valueEl ? valueEl.value : '';
}

function addTreeChild(node) {
    const type = node.dataset.type;
    let ch = node.querySelector(':scope > .tree-children');
    if (!ch) {
        ch = document.createElement('div');
        ch.className = 'tree-children';
        node.appendChild(ch);
    }
    if (type === 'object') {
        const newNode = createTreeNode('', '', false, false, null);
        ch.appendChild(newNode);
        newNode.querySelector('.tree-key-input').focus();
    } else if (type === 'array') {
        const idx = ch.querySelectorAll(':scope > .tree-node').length;
        ch.appendChild(createTreeNode(null, '', false, true, idx));
    }
}

function removeTreeNode(node) {
    const parent = node.parentNode;
    const grandparent = parent ? parent.parentNode : null;
    node.remove();
    if (grandparent && grandparent.dataset.type === 'array') {
        reindexArrayChildren(grandparent);
    }
}

function duplicateTreeNode(node) {
    const parent = node.parentNode;
    const grandparent = parent ? parent.parentNode : null;

    // Read the value of the current node (including all children)
    const value = readTreeValue(node);

    // Get the key from the node header
    const header = node.querySelector(':scope > .tree-node-header');
    const ki = header.querySelector('.tree-key-input');
    const badge = header.querySelector('.tree-index-badge');

    const isArrayItem = !!badge;
    let key = null;

    if (ki) {
        // For object properties, copy the key and append "_copy" if it's a duplicate
        key = ki.value + '_copy';
    }

    // Create a new node with the same value
    const newNode = createTreeNode(key, value, false, isArrayItem, 0);

    // Insert the new node after the current node
    if (node.nextSibling) {
        parent.insertBefore(newNode, node.nextSibling);
    } else {
        parent.appendChild(newNode);
    }

    // Reindex array children if we're in an array
    if (grandparent && grandparent.dataset.type === 'array') {
        reindexArrayChildren(grandparent);
    }
}

function reindexArrayChildren(parentNode) {
    const ch = parentNode.querySelector(':scope > .tree-children');
    if (!ch) return;
    ch.querySelectorAll(':scope > .tree-node').forEach(function (child, idx) {
        const badge = child.querySelector(':scope > .tree-node-header > .tree-index-badge');
        if (badge) badge.textContent = '[' + idx + ']';
    });
}

function changeNodeType(node, newType, isRoot) {
    const header = node.querySelector(':scope > .tree-node-header');
    const ki = header.querySelector('.tree-key-input');
    const badge = header.querySelector('.tree-index-badge');
    const isArrayItem = !!badge;
    const key = ki ? ki.value : null;
    const arrayIndex = badge ? parseInt(badge.textContent.match(/\d+/)?.[0] || '0') : null;

    let defaultValue;
    switch (newType) {
        case 'string':  defaultValue = ''; break;
        case 'number':  defaultValue = 0; break;
        case 'boolean': defaultValue = false; break;
        case 'null':    defaultValue = null; break;
        case 'object':  defaultValue = {}; break;
        case 'array':   defaultValue = []; break;
        default:        defaultValue = '';
    }

    const newNode = createTreeNode(key, defaultValue, isRoot, isArrayItem, arrayIndex);
    node.parentNode.replaceChild(newNode, node);
}

function switchJsonMode(fieldId, mode) {
    const textarea = document.getElementById(fieldId);
    const treeContainer = document.getElementById(fieldId + '-tree');
    const formGroup = textarea.closest('.form-group');
    formGroup.querySelectorAll('.mode-btn').forEach(function (btn) {
        btn.classList.toggle('active', btn.dataset.mode === mode);
    });

    if (mode === 'tree') {
        let data = {};
        const raw = textarea.value.trim();
        if (raw) {
            try { data = JSON.parse(raw); } catch (e) { /* invalid JSON — start with empty object */ }
        }
        renderJsonTree(fieldId + '-tree', data);
        textarea.style.display = 'none';
        treeContainer.style.display = 'block';
    } else {
        const root = treeContainer.querySelector('.tree-node');
        if (root) {
            textarea.value = JSON.stringify(readTreeValue(root), null, 2);
        }
        textarea.style.display = 'block';
        treeContainer.style.display = 'none';
    }
}

function getJsonFieldValue(fieldId) {
    const textarea = document.getElementById(fieldId);
    const treeContainer = document.getElementById(fieldId + '-tree');
    if (treeContainer && treeContainer.style.display !== 'none') {
        const root = treeContainer.querySelector('.tree-node');
        return root ? JSON.stringify(readTreeValue(root), null, 2) : '';
    }
    return textarea.value.trim();
}

function resetJsonTreeMode(fieldId) {
    const textarea = document.getElementById(fieldId);
    const treeContainer = document.getElementById(fieldId + '-tree');
    textarea.style.display = 'block';
    treeContainer.style.display = 'none';
    textarea.closest('.form-group').querySelectorAll('.mode-btn').forEach(function (btn) {
        btn.classList.toggle('active', btn.dataset.mode === 'raw');
    });
}

// Helper function to recursively sort JSON object keys alphabetically
function sortJsonKeys(obj) {
    if (obj === null || typeof obj !== 'object') {
        return obj;
    }

    if (Array.isArray(obj)) {
        return obj.map(sortJsonKeys);
    }

    const sorted = {};
    Object.keys(obj)
        .sort()
        .forEach(key => {
            sorted[key] = sortJsonKeys(obj[key]);
        });

    return sorted;
}

function formatJson(fieldId) {
    const textarea = document.getElementById(fieldId);
    if (!textarea) {
        console.error('Textarea not found:', fieldId);
        showError('Field not found');
        return;
    }

    const treeContainer = document.getElementById(fieldId + '-tree');

    // Check if we're in tree mode or raw mode
    const isTreeMode = treeContainer && treeContainer.style.display !== 'none';

    if (isTreeMode) {
        // Switch to raw mode first, then format
        switchJsonMode(fieldId, 'raw');
        // The textarea value is already set by switchJsonMode, so we can format it
    }

    const rawValue = textarea.value.trim();

    if (!rawValue) {
        showError(t('json.format.error.empty'));
        return;
    }

    try {
        // Parse the JSON
        const parsed = JSON.parse(rawValue);

        // Sort the JSON keys recursively
        const sortedJson = sortJsonKeys(parsed);

        // Re-stringify with proper indentation
        const formatted = JSON.stringify(sortedJson, null, 2);

        // Update the textarea
        textarea.value = formatted;

        showSuccess(t('json.format.success'));
    } catch (e) {
        // Provide more helpful error message
        let errorMsg = t('json.format.error.invalid');
        if (e.message) {
            errorMsg += ':\n' + e.message;
        }

        // Try to identify common issues
        if (rawValue.includes('}{')) {
            errorMsg += t('json.format.error.tip.braces');
        } else if (rawValue.match(/,\s*[}\]]/)) {
            errorMsg += t('json.format.error.tip.trailing');
        } else if (!rawValue.startsWith('{') && !rawValue.startsWith('[')) {
            errorMsg += t('json.format.error.tip.start');
        }

        showError(errorMsg);
        console.error('JSON parse error:', e);
    }
}

// ==================== Searchable Select ====================

let searchDebounceTimers = {};

function createSearchableSelect(inputId, options = {}) {
    const input = document.getElementById(inputId);
    if (!input) return;

    const wrapper = document.createElement('div');
    wrapper.className = 'searchable-select-wrapper';

    const dropdown = document.createElement('div');
    dropdown.className = 'searchable-select-dropdown';
    dropdown.id = inputId + '-dropdown';

    input.parentNode.insertBefore(wrapper, input);
    wrapper.appendChild(input);
    wrapper.appendChild(dropdown);

    input.classList.add('searchable-select-input');
    input.setAttribute('autocomplete', 'off');
    input.dataset.selectedValue = '';
    input.dataset.searchType = options.searchType || 'feature';

    // Handle input focus - show dropdown
    input.addEventListener('focus', function() {
        if (input.value.trim() === '' || input.value === input.placeholder) {
            input.value = '';
            loadInitialOptions(inputId, options);
        }
        dropdown.classList.add('show');
    });

    // Handle input typing - search
    input.addEventListener('input', function() {
        const query = input.value.trim();
        input.dataset.selectedValue = '';

        clearTimeout(searchDebounceTimers[inputId]);
        searchDebounceTimers[inputId] = setTimeout(() => {
            searchOptions(inputId, query, options);
        }, 300);
    });

    // Handle clicking outside to close dropdown
    document.addEventListener('click', function(e) {
        if (!wrapper.contains(e.target)) {
            dropdown.classList.remove('show');
            // Restore selected value text if nothing was selected
            if (input.dataset.selectedValue) {
                input.value = input.dataset.selectedText || input.dataset.selectedValue;
            } else if (input.value.trim() !== '') {
                input.value = '';
                input.placeholder = options.placeholder || 'Select...';
            }
        }
    });

    return { input, dropdown, wrapper };
}

async function loadInitialOptions(inputId, options) {
    const input = document.getElementById(inputId);
    const dropdown = document.getElementById(inputId + '-dropdown');
    const searchType = input.dataset.searchType;

    try {
        let items = [];
        if (searchType === 'feature') {
            const response = await fetch(`${API_BASE_URL}/features?page_size=10`);
            if (response.ok) {
                const result = await response.json();
                items = result.data || [];
            }
        } else if (searchType === 'scenario') {
            const featureName = options.featureName || input.dataset.featureName;
            if (featureName) {
                const response = await fetch(`${API_BASE_URL}/scenarios?feature_name=${featureName}&page_size=10`);
                if (response.ok) {
                    const result = await response.json();
                    items = result.data || [];
                }
            }
        }

        renderDropdownOptions(inputId, items, options);
    } catch (error) {
        console.error('Error loading options:', error);
    }
}

async function searchOptions(inputId, query, options) {
    const input = document.getElementById(inputId);
    const dropdown = document.getElementById(inputId + '-dropdown');
    const searchType = input.dataset.searchType;

    if (!query) {
        loadInitialOptions(inputId, options);
        return;
    }

    try {
        let items = [];
        if (searchType === 'feature') {
            const response = await fetch(`${API_BASE_URL}/features/search?q=${encodeURIComponent(query)}`);
            if (response.ok) {
                const result = await response.json();
                items = result.data || result || [];
            }
        } else if (searchType === 'scenario') {
            const featureName = options.featureName || input.dataset.featureName;
            const url = featureName
                ? `${API_BASE_URL}/scenarios/search?q=${encodeURIComponent(query)}&feature_name=${featureName}`
                : `${API_BASE_URL}/scenarios/search?q=${encodeURIComponent(query)}`;
            const response = await fetch(url);
            if (response.ok) {
                const result = await response.json();
                items = result.data || result || [];
            }
        }

        renderDropdownOptions(inputId, items, options);
    } catch (error) {
        console.error('Error searching:', error);
        dropdown.innerHTML = `<div class="searchable-select-option disabled">${t('dropdown.error')}</div>`;
    }
}

function renderDropdownOptions(inputId, items, options) {
    const input = document.getElementById(inputId);
    const dropdown = document.getElementById(inputId + '-dropdown');

    if (!items || items.length === 0) {
        dropdown.innerHTML = `<div class="searchable-select-option disabled">${t('dropdown.noResults')}</div>`;
        return;
    }

    dropdown.innerHTML = '';
    items.forEach(item => {
        const option = document.createElement('div');
        option.className = 'searchable-select-option';
        option.textContent = item.name;
        option.dataset.value = item.name;

        if (input.dataset.selectedValue === item.name) {
            option.classList.add('selected');
        }

        option.addEventListener('click', function() {
            input.dataset.selectedValue = item.name;
            input.dataset.selectedText = item.name;
            input.value = item.name;
            dropdown.classList.remove('show');

            // Trigger change event
            if (options.onChange) {
                options.onChange(item.name);
            }

            // Update all options to show selected state
            dropdown.querySelectorAll('.searchable-select-option').forEach(opt => {
                opt.classList.remove('selected');
            });
            option.classList.add('selected');
        });

        dropdown.appendChild(option);
    });
}

// ==================== Load Test Scenarios ====================

async function loadLoadTestScenarios(page = 1, searchQuery = '') {
    try {
        let url;
        if (searchQuery && searchQuery.trim() !== '') {
            // Use search endpoint
            url = `${API_BASE_URL}/loadtest/scenarios/search?q=${encodeURIComponent(searchQuery)}&page=${page}&page_size=10`;
        } else {
            // Use regular list endpoint
            url = `${API_BASE_URL}/loadtest/scenarios?page=${page}&page_size=10`;
        }

        const response = await fetch(url);
        if (!response.ok) throw new Error('Failed to load load test scenarios');

        const result = await response.json();
        loadTestScenarios = result.data;
        pagination.loadtest = { page: result.page, totalPages: result.total_pages };
        renderLoadTestTable();
        renderPagination('loadtest', result.page, result.total_pages);
    } catch (error) {
        console.error('Error loading load test scenarios:', error);
        showError(t('loadtest.error.load'));
    }
}

async function renderLoadTestTable() {
    const tbody = document.getElementById('loadtest-table-body');

    if (!loadTestScenarios || loadTestScenarios.length === 0) {
        tbody.innerHTML = `<tr><td colspan="7" class="loading">${t('loadtest.noResults')}</td></tr>`;
        return;
    }

    // Calculate account counts from the accounts string
    const accountCounts = {};
    for (const scenario of loadTestScenarios) {
        if (scenario.accounts && scenario.accounts.trim()) {
            // Count comma-separated username-password pairs
            accountCounts[scenario.id] = scenario.accounts.split(',').filter(a => a.trim()).length;
        } else {
            accountCounts[scenario.id] = 0;
        }
    }

    tbody.innerHTML = loadTestScenarios.map(scenario => `
        <tr>
            <td><strong>${scenario.name || 'N/A'}</strong></td>
            <td>${scenario.description || '-'}</td>
            <td>${t('loadtest.table.stepsCount', scenario.steps ? scenario.steps.length : 0)}</td>
            <td>${t('loadtest.table.accountsCount', accountCounts[scenario.id] || 0)}</td>
            <td><span class="status-badge ${scenario.is_active ? 'status-active' : 'status-inactive'}">
                ${scenario.is_active ? t('common.active') : t('common.inactive')}
            </span></td>
            <td>${formatDate(scenario.created_at)}</td>
            <td class="actions">
                <button class="btn btn-primary" onclick='runLoadTestScenario("${scenario.id}")' style="background: #10b981;">${t('loadtest.btn.run')}</button>
                <button class="btn btn-edit" onclick='editLoadTestScenario("${scenario.id}")'>${t('common.edit')}</button>
                <button class="btn btn-duplicate" onclick='duplicateLoadTestScenario("${scenario.id}")'>${t('common.duplicate')}</button>
                <button class="btn btn-outline" onclick='exportScenarioToJSON("${scenario.id}")'>${t('loadtest.btn.export')}</button>
                <button class="btn btn-delete" onclick="deleteLoadTestScenario('${scenario.id}')">${t('common.delete')}</button>
            </td>
        </tr>
    `).join('');
}

function showCreateLoadTestModal() {
    document.getElementById('loadtest-modal-title').textContent = t('loadtest.modal.create');
    document.getElementById('loadtest-id').value = '';
    document.getElementById('loadtest-name').value = '';
    document.getElementById('loadtest-description').value = '';
    document.getElementById('loadtest-accounts-input').value = '098888888-Test123456,09575757-Test123456,0966666666-Password789';
    document.getElementById('loadtest-active').checked = true;

    // Update account count display
    updateAccountCount();

    // Clear existing steps
    const container = document.getElementById('loadtest-steps-container');
    container.innerHTML = '';

    // Add one empty step by default
    addLoadTestStep();

    showModal('loadtest-modal');
}

function addLoadTestStep() {
    const container = document.getElementById('loadtest-steps-container');
    const template = document.getElementById('loadtest-step-template');
    const clone = template.content.cloneNode(true);

    // Update step number
    const stepNumber = container.children.length + 1;
    clone.querySelector('.step-number').textContent = stepNumber;

    // Apply i18n translations to cloned template content
    applyTranslations(clone);

    container.appendChild(clone);
}

function updateAccountCount() {
    const accountsInput = document.getElementById('loadtest-accounts-input');
    const countSpan = document.getElementById('loadtest-accounts-count');
    
    if (accountsInput && countSpan) {
        const accounts = accountsInput.value.trim();
        if (accounts) {
            const count = accounts.split(',').filter(a => a.trim()).length;
            countSpan.textContent = count;
        } else {
            countSpan.textContent = '0';
        }
    }
}

function removeLoadTestStep(button) {
    const stepDiv = button.closest('.loadtest-step');
    stepDiv.remove();

    // Renumber remaining steps
    const container = document.getElementById('loadtest-steps-container');
    const steps = container.querySelectorAll('.loadtest-step');
    steps.forEach((step, index) => {
        step.querySelector('.step-number').textContent = index + 1;
    });
}

function addStepVariable(button) {
    const container = button.previousElementSibling;
    const template = document.getElementById('variable-row-template');
    const clone = template.content.cloneNode(true);
    applyTranslations(clone);
    container.appendChild(clone);
}

function populateVariablesInStep(stepDiv, variables) {
    const container = stepDiv.querySelector('.step-variables-container');
    container.innerHTML = '';

    if (!variables || Object.keys(variables).length === 0) {
        return;
    }

    // variables is a simple map: { varName: jsonPath }
    for (const [varName, jsonPath] of Object.entries(variables)) {
        const template = document.getElementById('variable-row-template');
        const clone = template.content.cloneNode(true);
        applyTranslations(clone);

        clone.querySelector('.var-name').value = varName || '';
        clone.querySelector('.var-jsonpath').value = jsonPath || '';

        container.appendChild(clone);
    }
}

function getStepsFromForm() {
    const container = document.getElementById('loadtest-steps-container');
    const stepDivs = container.querySelectorAll('.loadtest-step');
    const steps = [];

    console.log('Total step divs found:', stepDivs.length);

    stepDivs.forEach((stepDiv, index) => {
        const variables = getSimpleVariablesFromStep(stepDiv);

        const stepName = stepDiv.querySelector('.step-name').value.trim();
        const stepPath = stepDiv.querySelector('.step-path').value.trim();

        console.log(`Step ${index + 1}:`, { name: stepName, path: stepPath });

        const step = {
            name: stepName,
            method: stepDiv.querySelector('.step-method').value,
            path: stepPath,
            body: stepDiv.querySelector('.step-body').value.trim(),
            save_variables: variables,
            expect_status: parseInt(stepDiv.querySelector('.step-expectstatus').value) || 200
        };

        // Parse headers if provided
        const headersStr = stepDiv.querySelector('.step-headers').value.trim();
        if (headersStr) {
            try {
                step.headers = JSON.parse(headersStr);
            } catch (e) {
                step.headers = {};
            }
        }

        // Add retry_for_seconds if provided and > 0
        const retryForSeconds = parseInt(stepDiv.querySelector('.step-retry-for-seconds').value) || 0;
        if (retryForSeconds > 0) {
            step.retry_for_seconds = retryForSeconds;
        }

        // Add max_retry_times if provided and > 0
        const maxRetryTimes = parseInt(stepDiv.querySelector('.step-max-retry-times').value) || 0;
        if (maxRetryTimes > 0) {
            step.max_retry_times = maxRetryTimes;
        }

        // Add wait_after_seconds if provided and > 0
        const waitAfterSeconds = parseInt(stepDiv.querySelector('.step-wait-after-seconds').value) || 0;
        if (waitAfterSeconds > 0) {
            step.wait_after_seconds = waitAfterSeconds;
        }

        // Add condition if provided
        const condition = stepDiv.querySelector('.step-condition').value.trim();
        if (condition) {
            step.condition = condition;
        }

        // Always push steps that have at least a name or path
        if (step.name || step.path) {
            steps.push(step);
            console.log(`Step ${index + 1} added to list`);
        } else {
            console.log(`Step ${index + 1} skipped - missing name and path`);
        }
    });

    console.log('Total steps to send:', steps.length);
    return steps;
}

function getSimpleVariablesFromStep(stepDiv) {
    const varRows = stepDiv.querySelectorAll('.variable-row');
    const variables = {};

    varRows.forEach(row => {
        const name = row.querySelector('.var-name').value.trim();
        const jsonPath = row.querySelector('.var-jsonpath').value.trim();

        if (name && jsonPath) {
            variables[name] = jsonPath;
        }
    });

    return variables;
}

function populateStepsInForm(steps) {
    const container = document.getElementById('loadtest-steps-container');
    container.innerHTML = '';

    if (!steps || steps.length === 0) {
        addLoadTestStep();
        return;
    }

    steps.forEach((step, index) => {
        addLoadTestStep();
        const stepDivs = container.querySelectorAll('.loadtest-step');
        const stepDiv = stepDivs[index];

        stepDiv.querySelector('.step-name').value = step.name || '';
        // Use empty string for "Auto" option, otherwise use the method value
        stepDiv.querySelector('.step-method').value = step.method || '';
        stepDiv.querySelector('.step-path').value = step.path || '';
        stepDiv.querySelector('.step-body').value = step.body || '';
        stepDiv.querySelector('.step-expectstatus').value = step.expect_status || 200;

        // Populate retry and wait fields
        stepDiv.querySelector('.step-retry-for-seconds').value = step.retry_for_seconds || 0;
        stepDiv.querySelector('.step-max-retry-times').value = step.max_retry_times || 0;
        stepDiv.querySelector('.step-wait-after-seconds').value = step.wait_after_seconds || 0;

        // Populate condition field
        stepDiv.querySelector('.step-condition').value = step.condition || '';

        if (step.headers && Object.keys(step.headers).length > 0) {
            stepDiv.querySelector('.step-headers').value = JSON.stringify(step.headers, null, 2);
        } else {
            stepDiv.querySelector('.step-headers').value = '';
        }

        // Populate saved variables
        if (step.save_variables && Object.keys(step.save_variables).length > 0) {
            populateVariablesInStep(stepDiv, step.save_variables);
        }
    });
}

async function runLoadTestScenario(id) {
    if (!confirm(t('loadtest.confirm.run'))) return;

    try {
        // showSuccess('Running load test... This may take a while.');
        
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios/${id}/run`, {
            method: 'POST'
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || 'Failed to run load test');
        }

        const result = await response.json();

        // Display results
        displayLoadTestResults(result);
        // showSuccess('Load test completed successfully!');
    } catch (error) {
        console.error('Error running load test:', error);
        showError(t('loadtest.error.run') + ': ' + error.message);
    }
}

function displayLoadTestResults(result) {
    const summary = `
${t('loadtest.results.title')}
================
${t('loadtest.results.scenario')}: ${result.ScenarioName || 'N/A'}
${t('loadtest.results.totalAccounts')}: ${result.TotalAccounts || 0}
${t('loadtest.results.success')}: ${result.SuccessCount || 0}
${t('loadtest.results.failure')}: ${result.FailureCount || 0}
${t('loadtest.results.totalDuration')}: ${result.TotalDuration || 0}ms
${t('loadtest.results.avgDuration')}: ${result.AvgDuration || 0}ms

${result.AccountResults && result.AccountResults.length > 0 ?
    t('loadtest.results.accountResults') + ':\n' + result.AccountResults.map(acc =>
        `- ${acc.Username}: ${acc.Success ? t('loadtest.results.successMark') : t('loadtest.results.failMark')} (${acc.TotalTime}ms)${acc.FailedStep ? ' - ' + t('loadtest.results.failedAt') + ': ' + acc.FailedStep : ''}`
    ).join('\n') : ''}`;

    alert(summary);
    console.log('Full Load Test Results:', result);
}

async function viewLoadTestScenario(id) {
    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios/${id}`);
        if (!response.ok) throw new Error('Failed to load scenario');

        const scenario = await response.json();

        // Create a formatted view
        let stepsHtml = '';
        if (scenario.steps) {
            stepsHtml = scenario.steps.map((step, index) => {
                let varsHtml = '';
                if (step.save_variables && Object.keys(step.save_variables).length > 0) {
                    varsHtml = Object.entries(step.save_variables).map(([name, path]) =>
                        `  - {{${name}}} = ${path}`
                    ).join('\n');
                    varsHtml = `\nSave Variables:\n${varsHtml}`;
                }

                return `
Step ${index + 1}: ${step.name}
  ${step.method} ${step.path}
  ${step.body ? `Body: ${step.body.substring(0, 100)}...` : ''}${varsHtml}
  Expected: ${step.expect_status}
`;
            }).join('\n');
        }

        const details = `
=== ${scenario.name} ===
Description: ${scenario.description || '-'}
Base URL: ${scenario.base_url}
Concurrency: ${scenario.concurrency}

--- Steps (${scenario.steps ? scenario.steps.length : 0}) ---
${stepsHtml}
`;

        alert(details);
    } catch (error) {
        console.error('Error loading scenario:', error);
        showError(t('loadtest.error.load'));
    }
}

async function editLoadTestScenario(id) {
    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios/${id}`);
        if (!response.ok) throw new Error('Failed to load scenario');

        const scenario = await response.json();

        document.getElementById('loadtest-modal-title').textContent = t('loadtest.modal.edit');
        document.getElementById('loadtest-id').value = scenario.id;
        document.getElementById('loadtest-name').value = scenario.name || '';
        document.getElementById('loadtest-description').value = scenario.description || '';
        document.getElementById('loadtest-accounts-input').value = scenario.accounts || '';
        document.getElementById('loadtest-active').checked = scenario.is_active;

        // Update account count display
        updateAccountCount();

        populateStepsInForm(scenario.steps);

        showModal('loadtest-modal');
    } catch (error) {
        console.error('Error loading scenario:', error);
        showError(t('loadtest.error.load'));
    }
}

async function duplicateLoadTestScenario(id) {
    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios/${id}`);
        if (!response.ok) throw new Error('Failed to load scenario');

        const scenario = await response.json();

        document.getElementById('loadtest-modal-title').textContent = t('loadtest.modal.duplicate');

        // Clear ID so it creates a new record
        document.getElementById('loadtest-id').value = '';

        // Generate versioned name
        let newName = scenario.name || '';
        const versionRegex = / v(\d+)$/;
        const match = newName.match(versionRegex);

        if (match) {
            // Increment existing version number
            const currentVersion = parseInt(match[1]);
            newName = newName.replace(versionRegex, ` v${currentVersion + 1}`);
        } else {
            // Append v2 for first duplicate
            newName = `${newName} v2`;
        }

        document.getElementById('loadtest-name').value = newName;
        document.getElementById('loadtest-description').value = scenario.description || '';
        document.getElementById('loadtest-accounts-input').value = scenario.accounts || '';
        document.getElementById('loadtest-active').checked = scenario.is_active;

        // Update account count display
        updateAccountCount();

        populateStepsInForm(scenario.steps);

        showModal('loadtest-modal');
    } catch (error) {
        console.error('Error loading scenario:', error);
        showError(t('loadtest.error.load'));
    }
}

async function saveLoadTestScenario() {
    const id = document.getElementById('loadtest-id').value;
    const name = document.getElementById('loadtest-name').value.trim();
    const description = document.getElementById('loadtest-description').value.trim();
    const accounts = document.getElementById('loadtest-accounts-input').value.trim();
    const concurrency = 10; // Default concurrency
    const isActive = document.getElementById('loadtest-active').checked;
    const steps = getStepsFromForm();

    // Clear previous errors
    clearFieldError('loadtest-name');

    if (!name) {
        showFieldError('loadtest-name', t('loadtest.error.nameRequired'));
        return;
    }

    if (name.includes(' ')) {
        showFieldError('loadtest-name', t('loadtest.error.nameHasSpaces'));
        return;
    }

    if (name.length > 100) {
        showFieldError('loadtest-name', t('loadtest.error.nameTooLong'));
        return;
    }

    if (steps.length === 0) {
        showError(t('loadtest.error.stepsRequired'));
        return;
    }

    const data = {
        name: name,
        description: description,
        accounts: accounts,
        concurrency: concurrency,
        steps: steps,
        is_active: isActive
    };

    try {
        const url = id ? `${API_BASE_URL}/loadtest/scenarios/${id}` : `${API_BASE_URL}/loadtest/scenarios`;
        const method = id ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        if (!response.ok) throw new Error('Failed to save load test scenario');

        closeModal('loadtest-modal');
        showSuccess(id ? t('loadtest.success.updated') : t('loadtest.success.created'));
        loadLoadTestScenarios();
    } catch (error) {
        console.error('Error saving load test scenario:', error);
        showError(t('loadtest.error.save'));
    }
}

async function deleteLoadTestScenario(id) {
    if (!confirm(t('loadtest.confirm.delete'))) return;

    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete load test scenario');

        showSuccess(t('loadtest.success.deleted'));
        loadLoadTestScenarios();
    } catch (error) {
        console.error('Error deleting load test scenario:', error);
        showError(t('loadtest.error.delete'));
    }
}

// ==================== Account Management ====================

async function showAccountsModal(scenarioId, scenarioName) {
    document.getElementById('accounts-scenario-id').value = scenarioId;
    document.getElementById('accounts-modal-title').textContent = `Manage Accounts - ${scenarioName}`;
    document.getElementById('accounts-text').value = '';

    await refreshAccountsList(scenarioId);
    showModal('accounts-modal');
}

async function refreshAccountsList(scenarioId) {
    const listDiv = document.getElementById('accounts-list');
    const countSpan = document.getElementById('accounts-count');

    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios/${scenarioId}/accounts`);
        if (!response.ok) throw new Error('Failed to load accounts');

        const accounts = await response.json();
        countSpan.textContent = accounts.length;

        if (!accounts || accounts.length === 0) {
            listDiv.innerHTML = '<p style="color: #718096; text-align: center;">No accounts yet. Add accounts using the text box above.</p>';
            return;
        }

        listDiv.innerHTML = accounts.map((acc, index) => `
            <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px; background: ${index % 2 === 0 ? '#f8fafc' : '#fff'}; border-radius: 4px; margin-bottom: 2px;">
                <div>
                    <strong>${acc.username}</strong>
                    <span style="color: #718096; margin-left: 10px;">****${acc.password.slice(-4)}</span>
                </div>
                <button class="btn btn-delete" onclick="deleteAccount('${acc.id}')" style="padding: 2px 8px; font-size: 11px;">Remove</button>
            </div>
        `).join('');
    } catch (error) {
        console.error('Error loading accounts:', error);
        listDiv.innerHTML = '<p style="color: #e53e3e;">Failed to load accounts</p>';
    }
}

async function addAccountsFromText() {
    const scenarioId = document.getElementById('accounts-scenario-id').value;
    const text = document.getElementById('accounts-text').value.trim();

    if (!text) {
        showError('Please enter accounts in the format: username-password,username-password');
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios/${scenarioId}/accounts`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ accounts_text: text })
        });

        if (!response.ok) {
            const err = await response.json();
            throw new Error(err.message || 'Failed to add accounts');
        }

        const result = await response.json();
        showSuccess(`Added ${result.count} account(s) successfully`);
        document.getElementById('accounts-text').value = '';
        await refreshAccountsList(scenarioId);
        loadLoadTestScenarios(); // Refresh the main table to update account count
    } catch (error) {
        console.error('Error adding accounts:', error);
        showError('Failed to add accounts: ' + error.message);
    }
}

async function deleteAccount(accountId) {
    if (!confirm('Are you sure you want to remove this account?')) return;

    const scenarioId = document.getElementById('accounts-scenario-id').value;

    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/accounts/${accountId}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete account');

        await refreshAccountsList(scenarioId);
        loadLoadTestScenarios(); // Refresh the main table to update account count
    } catch (error) {
        console.error('Error deleting account:', error);
        showError('Failed to delete account');
    }
}

async function clearAllAccounts() {
    if (!confirm('Are you sure you want to remove ALL accounts for this scenario?')) return;

    const scenarioId = document.getElementById('accounts-scenario-id').value;

    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios/${scenarioId}/accounts`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to clear accounts');

        showSuccess('All accounts cleared');
        await refreshAccountsList(scenarioId);
        loadLoadTestScenarios(); // Refresh the main table to update account count
    } catch (error) {
        console.error('Error clearing accounts:', error);
        showError('Failed to clear accounts');
    }
}

/* ==================== JSON Import/Export ==================== */

async function handleJSONImport(event) {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = async function(e) {
        try {
            const jsonText = e.target.result;
            const scenario = JSON.parse(jsonText);

            // Validate required fields
            if (!scenario.name) {
                throw new Error(t('import.error.noName'));
            }
            if (!scenario.steps || scenario.steps.length === 0) {
                throw new Error(t('import.error.noSteps'));
            }

            // Check if scenario with same name already exists
            const existingScenario = loadTestScenarios.find(s =>
                s.name.toLowerCase() === scenario.name.toLowerCase()
            );

            if (existingScenario) {
                const confirmImport = confirm(t('import.duplicateName', scenario.name));

                if (!confirmImport) {
                    showError(t('import.cancelled'));
                    event.target.value = '';
                    return;
                }

                // Optionally append timestamp to make name unique
                scenario.name = `${scenario.name} (imported ${new Date().toLocaleString()})`;
            }

            // Ensure is_active is set
            if (scenario.is_active === undefined) {
                scenario.is_active = true;
            }

            // Remove id if present (will be generated by backend)
            delete scenario.id;
            delete scenario._id;
            delete scenario.created_at;
            delete scenario.updated_at;

            // Create the scenario via API
            const response = await fetch(`${API_BASE_URL}/loadtest/scenarios`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(scenario)
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.message || 'Failed to import scenario');
            }

            showSuccess(t('import.success', scenario.name));
            loadLoadTestScenarios();

            // Reset file input
            event.target.value = '';
        } catch (error) {
            console.error('Error importing JSON:', error);
            showError(t('import.error') + ': ' + error.message);
            event.target.value = '';
        }
    };
    reader.readAsText(file);
}

async function exportScenarioToJSON(scenarioId) {
    try {
        const scenario = loadTestScenarios.find(s => s.id === scenarioId);
        if (!scenario) {
            showError(t('loadtest.noResults'));
            return;
        }

        // Create a clean copy without id and timestamps
        const exportData = {
            name: scenario.name,
            description: scenario.description || '',
            accounts: scenario.accounts || '',
            steps: scenario.steps || [],
            is_active: scenario.is_active !== undefined ? scenario.is_active : true
        };

        // Convert to JSON with pretty formatting
        const jsonContent = JSON.stringify(exportData, null, 2);

        // Create download link
        const blob = new Blob([jsonContent], { type: 'application/json;charset=utf-8;' });
        const link = document.createElement('a');
        const url = URL.createObjectURL(blob);

        link.setAttribute('href', url);
        link.setAttribute('download', `${scenario.name}.json`);
        link.style.visibility = 'hidden';

        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);

        showSuccess(t('loadtest.success.exported'));
    } catch (error) {
        console.error('Error exporting scenario:', error);
        showError(t('loadtest.error.run'));
    }
}
