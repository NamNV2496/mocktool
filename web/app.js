const API_BASE_URL = 'http://localhost:8081/api/v1/mocktool';

let features = [];
let scenarios = [];
let mockAPIs = [];
let loadTestScenarios = [];
let activeScenarioId = null; // Track the active scenario ID for the current accountId

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

function switchTab(tabName) {
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));

    document.querySelector(`[data-tab="${tabName}"]`).classList.add('active');
    document.getElementById(tabName).classList.add('active');

    if (tabName === 'features') {
        loadFeatures();
    } else if (tabName === 'scenarios') {
        populateFeatureFilters();
    } else if (tabName === 'mockapis') {
        populateMockAPIFilters();
    } else if (tabName === 'loadtest') {
        loadLoadTestScenarios();
    }
}

async function loadFeatures(page = 1) {
    try {
        const response = await fetch(`${API_BASE_URL}/features?page=${page}&page_size=10`);
        if (!response.ok) throw new Error('Failed to load features');

        const result = await response.json();
        features = result.data;
        pagination.features = { page: result.page, totalPages: result.total_pages };
        renderFeaturesTable();
        renderPagination('features', result.page, result.total_pages);
    } catch (error) {
        console.error('Error loading features:', error);
        showError('Failed to load features');
    }
}

function renderFeaturesTable() {
    const tbody = document.getElementById('features-table-body');

    if (!features || features.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="loading">No features found</td></tr>';
        return;
    }

    tbody.innerHTML = features.map(feature => `
        <tr>
            <td><strong>${feature.name || 'N/A'}</strong></td>
            <td>${feature.description || '-'}</td>
            <td><span class="status-badge ${feature.is_active ? 'status-active' : 'status-inactive'}">
                ${feature.is_active ? 'Active' : 'Inactive'}
            </span></td>
            <td>${formatDate(feature.created_at)}</td>
            <td class="actions">
                <button class="btn btn-edit" onclick='editFeature(${JSON.stringify(feature).replace(/'/g, "&#39;")})'>Edit</button>
                <button class="btn btn-delete" onclick="deleteFeature('${feature.id}')">Delete</button>
            </td>
        </tr>
    `).join('');
}

async function loadScenarios(featureName, page = 1) {
    if (!featureName) {
        document.getElementById('scenarios-table-body').innerHTML =
            '<tr><td colspan="6" class="loading">Select a feature to view scenarios</td></tr>';
        document.getElementById('scenarios-pagination').innerHTML = '';
        activeScenarioId = null;
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

        renderScenariosTable();
        renderPagination('scenarios', result.page, result.total_pages);
    } catch (error) {
        console.error('Error loading scenarios:', error);
        showError('Failed to load scenarios');
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

function renderScenariosTable() {
    const tbody = document.getElementById('scenarios-table-body');

    if (!scenarios || scenarios.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="loading">No scenarios found</td></tr>';
        return;
    }

    // Get the accountId filter value
    const filterAccountId = document.getElementById('scenario-account-id-filter')?.value.trim();

    tbody.innerHTML = scenarios.map(scenario => {
        // Check if this scenario is the active one
        const isActive = activeScenarioId && scenario.id === activeScenarioId;
        const rowStyle = isActive ? 'background-color: #e6ffed;' : '';

        return `
            <tr style="${rowStyle}">
                <td>${scenario.feature_name || 'N/A'}</td>
                <td>
                    <strong>${scenario.name || 'N/A'}</strong>
                    ${isActive ? '<span class="status-badge status-active" style="margin-left: 8px;">ACTIVE</span>' : ''}
                </td>
                <td>${scenario.description || '-'}</td>
                <td>${formatDate(scenario.created_at)}</td>
                <td class="actions">
                    <button class="btn btn-edit" onclick='editScenario(${JSON.stringify(scenario).replace(/'/g, "&#39;")})'>Edit</button>
                    <button class="btn btn-delete" onclick="deleteScenario('${scenario.id}')">Delete</button>
                    ${filterAccountId ? `<button class="btn btn-primary" onclick="activateScenario('${scenario.id}', '${filterAccountId}')">Activate for ${filterAccountId}</button>` : `<button class="btn btn-primary" onclick="activateScenarioGlobal('${scenario.id}')">Activate Globally</button>`}
                </td>
            </tr>
        `;
    }).join('');
}

async function loadMockAPIs(scenarioName, page = 1) {
    if (!scenarioName) {
        document.getElementById('mockapis-table-body').innerHTML =
            '<tr><td colspan="7" class="loading">Select a scenario to view mock APIs</td></tr>';
        document.getElementById('mockapis-pagination').innerHTML = '';
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/mockapis?scenario_name=${scenarioName}&page=${page}&page_size=10`);
        if (!response.ok) throw new Error('Failed to load mock APIs');

        const result = await response.json();
        mockAPIs = result.data;
        pagination.mockapis = { page: result.page, totalPages: result.total_pages };
        renderMockAPIsTable();
        renderPagination('mockapis', result.page, result.total_pages);
    } catch (error) {
        console.error('Error loading mock APIs:', error);
        showError('Failed to load mock APIs');
    }
}

function renderMockAPIsTable() {
    const tbody = document.getElementById('mockapis-table-body');

    if (!mockAPIs || mockAPIs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" class="loading">No mock APIs found</td></tr>';
        return;
    }

    tbody.innerHTML = mockAPIs.map(api => `
        <tr>
            <td>${api.feature_name || 'N/A'}</td>
            <td>${api.scenario_name || 'N/A'}</td>
            <td><strong>${api.name || 'N/A'}</strong></td>
            <td><span class="status-badge" style="background-color: #4299e1; color: white;">${api.method || 'GET'}</span></td>
            <td><code>${api.path || 'N/A'}</code></td>
            <td><span class="status-badge ${api.is_active ? 'status-active' : 'status-inactive'}">
                ${api.is_active ? 'Active' : 'Inactive'}
            </span></td>
            <td>${formatDate(api.created_at)}</td>
            <td class="actions">
                <button class="btn btn-edit" onclick='editMockAPI(${JSON.stringify(api)})'>Edit</button>
                <button class="btn btn-delete" onclick="deleteMockAPI('${api.id}')">Delete</button>
            </td>
        </tr>
    `).join('');
}

async function fetchAllFeatures() {
    try {
        const response = await fetch(`${API_BASE_URL}/features?page_size=1000`);
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
        const response = await fetch(`${API_BASE_URL}/scenarios?feature_name=${featureName}&page_size=1000`);
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
    html += `<button class="page-btn" ${currentPage === 1 ? 'disabled' : ''} onclick="loadPage('${entity}', ${currentPage - 1})">Prev</button>`;

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
    html += `<button class="page-btn" ${currentPage === totalPages ? 'disabled' : ''} onclick="loadPage('${entity}', ${currentPage + 1})">Next</button>`;

    html += '</div>';
    container.innerHTML = html;
}

function loadPage(entity, page) {
    if (entity === 'features') {
        loadFeatures(page);
    } else if (entity === 'scenarios') {
        const featureName = document.getElementById('scenario-feature-filter').value;
        loadScenarios(featureName, page);
    } else if (entity === 'mockapis') {
        const scenarioName = document.getElementById('mockapi-scenario-filter').value;
        loadMockAPIs(scenarioName, page);
    } else if (entity === 'loadtest') {
        loadLoadTestScenarios(page);
    }
}

async function populateFeatureFilters() {
    const allFeatures = await fetchAllFeatures();

    const scenarioFilter = document.getElementById('scenario-feature-filter');
    scenarioFilter.innerHTML = '<option value="">Select a feature...</option>';

    allFeatures.forEach(feature => {
        const option = document.createElement('option');
        option.value = feature.name;
        option.textContent = feature.name;
        scenarioFilter.appendChild(option);
    });

    scenarioFilter.onchange = (e) => loadScenarios(e.target.value);

    // Add event listener to accountId filter to re-render table when it changes
    const accountIdFilter = document.getElementById('scenario-account-id-filter');
    accountIdFilter.addEventListener('input', async () => {
        const accountId = accountIdFilter.value.trim();
        const featureName = scenarioFilter.value;

        if (featureName && accountId) {
            await fetchActiveScenario(featureName, accountId);
        } else {
            activeScenarioId = null;
        }

        renderScenariosTable();
    });
}

async function populateMockAPIFilters() {
    const allFeatures = await fetchAllFeatures();

    const featureFilter = document.getElementById('mockapi-feature-filter');
    featureFilter.innerHTML = '<option value="">Select a feature...</option>';

    allFeatures.forEach(feature => {
        const option = document.createElement('option');
        option.value = feature.name;
        option.textContent = feature.name;
        featureFilter.appendChild(option);
    });

    featureFilter.onchange = async (e) => {
        const featureName = e.target.value;
        if (!featureName) {
            document.getElementById('mockapi-scenario-filter').innerHTML = '<option value="">Select a scenario...</option>';
            return;
        }

        try {
            const scenarios = await fetchAllScenarios(featureName);

            const scenarioFilter = document.getElementById('mockapi-scenario-filter');
            scenarioFilter.innerHTML = '<option value="">Select a scenario...</option>';

            scenarios.forEach(scenario => {
                const option = document.createElement('option');
                option.value = scenario.name;
                option.textContent = scenario.name;
                scenarioFilter.appendChild(option);
            });
        } catch (error) {
            console.error('Error loading scenarios:', error);
            showError('Failed to load scenarios');
        }
    };

    const scenarioFilter = document.getElementById('mockapi-scenario-filter');
    scenarioFilter.onchange = (e) => loadMockAPIs(e.target.value);
}

function showCreateFeatureModal() {
    document.getElementById('feature-modal-title').textContent = 'Create Feature';
    document.getElementById('feature-id').value = '';
    document.getElementById('feature-name').value = '';
    document.getElementById('feature-description').value = '';
    document.getElementById('feature-active').checked = true;
    showModal('feature-modal');
}

function editFeature(feature) {
    document.getElementById('feature-modal-title').textContent = 'Edit Feature';
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

    if (!name) {
        showError('Feature name is required');
        return;
    }

    const data = {
        name: name,
        description: description,
        is_active: isActive
    };

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
        showSuccess(id ? 'Feature updated successfully' : 'Feature created successfully');
        loadFeatures();
    } catch (error) {
        console.error('Error saving feature:', error);
        showError('Failed to save feature');
    }
}

async function showCreateScenarioModal() {
    document.getElementById('scenario-modal-title').textContent = 'Create Scenario';
    document.getElementById('scenario-id').value = '';
    document.getElementById('scenario-name').value = '';
    document.getElementById('scenario-description').value = '';

    const allFeatures = await fetchAllFeatures();
    const featureSelect = document.getElementById('scenario-feature');
    featureSelect.innerHTML = '<option value="">Select a feature...</option>';
    allFeatures.forEach(feature => {
        const option = document.createElement('option');
        option.value = feature.name;
        option.textContent = feature.name;
        featureSelect.appendChild(option);
    });

    showModal('scenario-modal');
}

async function editScenario(scenario) {
    document.getElementById('scenario-modal-title').textContent = 'Edit Scenario';
    document.getElementById('scenario-id').value = scenario.id;
    document.getElementById('scenario-name').value = scenario.name;
    document.getElementById('scenario-description').value = scenario.description || '';

    const allFeatures = await fetchAllFeatures();
    const featureSelect = document.getElementById('scenario-feature');
    featureSelect.innerHTML = '<option value="">Select a feature...</option>';
    allFeatures.forEach(feature => {
        const option = document.createElement('option');
        option.value = feature.name;
        option.textContent = feature.name;
        if (feature.name === scenario.feature_name) {
            option.selected = true;
        }
        featureSelect.appendChild(option);
    });

    showModal('scenario-modal');
}

async function saveScenario() {
    const id = document.getElementById('scenario-id').value;
    const featureName = document.getElementById('scenario-feature').value;
    const name = document.getElementById('scenario-name').value.trim();
    const description = document.getElementById('scenario-description').value.trim();

    if (!featureName || !name) {
        showError('Feature and scenario name are required');
        return;
    }

    const data = {
        feature_name: featureName,
        name: name,
        description: description
    };

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
        showSuccess(id ? 'Scenario updated successfully' : 'Scenario created successfully');

        const currentFeature = document.getElementById('scenario-feature-filter').value;
        if (currentFeature) {
            loadScenarios(currentFeature);
        }
    } catch (error) {
        console.error('Error saving scenario:', error);
        showError('Failed to save scenario');
    }
}

async function activateScenario(scenarioId, accountId) {
    try {
        const url = `${API_BASE_URL}/scenarios/${scenarioId}/activate?account_id=${encodeURIComponent(accountId)}`;
        const response = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });

        if (!response.ok) throw new Error('Failed to activate scenario');

        showSuccess(`Scenario activated for account ${accountId}`);

        // Update active scenario ID and reload
        activeScenarioId = scenarioId;
        const currentFeature = document.getElementById('scenario-feature-filter').value;
        if (currentFeature) {
            loadScenarios(currentFeature);
        }
    } catch (error) {
        console.error('Error activating scenario:', error);
        showError('Failed to activate scenario');
    }
}

async function activateScenarioGlobal(scenarioId) {
    try {
        const url = `${API_BASE_URL}/scenarios/${scenarioId}/activate`;
        const response = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });

        if (!response.ok) throw new Error('Failed to activate scenario');

        showSuccess('Scenario activated globally');

        // Clear active scenario ID since this is a global activation
        activeScenarioId = null;
        const currentFeature = document.getElementById('scenario-feature-filter').value;
        if (currentFeature) {
            loadScenarios(currentFeature);
        }
    } catch (error) {
        console.error('Error activating scenario:', error);
        showError('Failed to activate scenario');
    }
}

async function showCreateMockAPIModal() {
    document.getElementById('mockapi-modal-title').textContent = 'Create Mock API';
    document.getElementById('mockapi-id').value = '';
    document.getElementById('mockapi-name').value = '';
    document.getElementById('mockapi-path').value = '';
    document.getElementById('mockapi-method').value = 'GET';
    document.getElementById('mockapi-regex-path').value = '';
    document.getElementById('mockapi-hash-input').value = '';
    document.getElementById('mockapi-output').value = '';
    document.getElementById('mockapi-active').checked = true;

    const allFeatures = await fetchAllFeatures();
    const featureSelect = document.getElementById('mockapi-feature');
    featureSelect.innerHTML = '<option value="">Select a feature...</option>';
    allFeatures.forEach(feature => {
        const option = document.createElement('option');
        option.value = feature.name;
        option.textContent = feature.name;
        featureSelect.appendChild(option);
    });

    featureSelect.onchange = async (e) => {
        const featureName = e.target.value;
        const scenarioSelect = document.getElementById('mockapi-scenario');
        scenarioSelect.innerHTML = '<option value="">Select a scenario...</option>';

        if (!featureName) return;

        try {
            const scenarios = await fetchAllScenarios(featureName);

            scenarios.forEach(scenario => {
                const option = document.createElement('option');
                option.value = scenario.name;
                option.textContent = scenario.name;
                scenarioSelect.appendChild(option);
            });
        } catch (error) {
            console.error('Error loading scenarios:', error);
        }
    };

    resetJsonTreeMode('mockapi-hash-input');
    resetJsonTreeMode('mockapi-output');
    showModal('mockapi-modal');
}

async function editMockAPI(api) {
    document.getElementById('mockapi-modal-title').textContent = 'Edit Mock API';
    document.getElementById('mockapi-id').value = api.id;
    document.getElementById('mockapi-name').value = api.name;
    document.getElementById('mockapi-path').value = api.path;
    document.getElementById('mockapi-method').value = api.method || 'GET';
    document.getElementById('mockapi-regex-path').value = api.regex_path || '';

    // Populate hash_input if it exists
    try {
        if (api.hash_input && api.hash_input !== null) {
            document.getElementById('mockapi-hash-input').value =
                typeof api.hash_input === 'string' ? api.hash_input : JSON.stringify(api.hash_input, null, 2);
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

    document.getElementById('mockapi-active').checked = api.is_active;

    const allFeatures = await fetchAllFeatures();
    const featureSelect = document.getElementById('mockapi-feature');
    featureSelect.innerHTML = '<option value="">Select a feature...</option>';
    allFeatures.forEach(feature => {
        const option = document.createElement('option');
        option.value = feature.name;
        option.textContent = feature.name;
        if (feature.name === api.feature_name) {
            option.selected = true;
        }
        featureSelect.appendChild(option);
    });

    loadScenariosForEdit(api.feature_name, api.scenario_name);
    resetJsonTreeMode('mockapi-hash-input');
    resetJsonTreeMode('mockapi-output');
    showModal('mockapi-modal');
}

async function loadScenariosForEdit(featureName, selectedScenario) {
    const scenarios = await fetchAllScenarios(featureName);

    const scenarioSelect = document.getElementById('mockapi-scenario');
    scenarioSelect.innerHTML = '<option value="">Select a scenario...</option>';

    scenarios.forEach(scenario => {
        const option = document.createElement('option');
        option.value = scenario.name;
        option.textContent = scenario.name;
        if (scenario.name === selectedScenario) {
            option.selected = true;
        }
        scenarioSelect.appendChild(option);
    });
}

async function saveMockAPI() {
    const id = document.getElementById('mockapi-id').value;
    const featureName = document.getElementById('mockapi-feature').value;
    const scenarioName = document.getElementById('mockapi-scenario').value;
    const name = document.getElementById('mockapi-name').value.trim();
    const path = document.getElementById('mockapi-path').value.trim();
    const method = document.getElementById('mockapi-method').value;
    const regexPath = document.getElementById('mockapi-regex-path').value.trim();
    const hashInputStr = getJsonFieldValue('mockapi-hash-input');
    const outputStr = getJsonFieldValue('mockapi-output');
    const outputHeaders = document.getElementById('mockapi-output-header').value.trim();
    const isActive = document.getElementById('mockapi-active').checked;

    if (!featureName || !scenarioName || !name || !path || !method || !outputStr) {
        showError('Feature, scenario, name, path, method, and output are required');
        return;
    }

    // Parse output JSON - will be automatically minified when sent to server
    let output;
    try {
        output = JSON.parse(outputStr);
    } catch (e) {
        showError('Invalid JSON in output field');
        return;
    }

    // Parse hash_input JSON if provided - will be automatically minified when sent to server
    let hashInput = null;
    if (hashInputStr) {
        try {
            hashInput = JSON.parse(hashInputStr);
        } catch (e) {
            showError('Invalid JSON in hash input field');
            return;
        }
    }

    const data = {
        feature_name: featureName,
        scenario_name: scenarioName,
        name: name,
        path: path,
        method: method,
        regex_path: regexPath,
        hash_input: hashInput,
        output: output,
        headers: outputHeaders,
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
        showSuccess(id ? 'Mock API updated successfully' : 'Mock API created successfully');

        const currentScenario = document.getElementById('mockapi-scenario-filter').value;
        if (currentScenario) {
            loadMockAPIs(currentScenario);
        }
    } catch (error) {
        console.error('Error saving mock API:', error);
        showError('Failed to save mock API');
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
    alert('Error: ' + message);
}

function showSuccess(message) {
    alert('Success: ' + message);
}

async function deleteFeature(id) {
    if (!confirm('Are you sure you want to delete this feature?')) return;

    try {
        const response = await fetch(`${API_BASE_URL}/features/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete feature');

        showSuccess('Feature deleted successfully');
        loadFeatures();
    } catch (error) {
        console.error('Error deleting feature:', error);
        showError('Failed to delete feature');
    }
}

async function deleteScenario(id) {
    if (!confirm('Are you sure you want to delete this scenario?')) return;

    try {
        const response = await fetch(`${API_BASE_URL}/scenarios/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete scenario');

        showSuccess('Scenario deleted successfully');
        const currentFeature = document.getElementById('scenario-feature-filter').value;
        if (currentFeature) {
            loadScenarios(currentFeature);
        }
    } catch (error) {
        console.error('Error deleting scenario:', error);
        showError('Failed to delete scenario');
    }
}

async function deleteMockAPI(id) {
    if (!confirm('Are you sure you want to delete this mock API?')) return;

    try {
        const response = await fetch(`${API_BASE_URL}/mockapis/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete mock API');

        showSuccess('Mock API deleted successfully');
        const currentScenario = document.getElementById('mockapi-scenario-filter').value;
        if (currentScenario) {
            loadMockAPIs(currentScenario);
        }
    } catch (error) {
        console.error('Error deleting mock API:', error);
        showError('Failed to delete mock API');
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
        showError('Please enter or upload a .proto file');
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
        container.innerHTML = '<p style="color: #718096; font-style: italic;">No messages found in the .proto file.</p>';
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
                    <button class="btn btn-proto-input btn-proto-select" onclick="selectProtoTemplate(${index}, 'input')">Use as Input</button>
                    <button class="btn btn-proto-output btn-proto-select" onclick="selectProtoTemplate(${index}, 'output')">Use as Output</button>
                </div>
            </div>`;
    });
    html += '</div>';
    html += '<div class="proto-apply-row">';
    html += '<button class="btn btn-primary" id="proto-apply-btn" style="display: none;" onclick="applyProtoTemplates()">Create Mock API</button>';
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

// ==================== Load Test Scenarios ====================

async function loadLoadTestScenarios(page = 1) {
    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios?page=${page}&page_size=10`);
        if (!response.ok) throw new Error('Failed to load load test scenarios');

        const result = await response.json();
        loadTestScenarios = result.data;
        pagination.loadtest = { page: result.page, totalPages: result.total_pages };
        renderLoadTestTable();
        renderPagination('loadtest', result.page, result.total_pages);
    } catch (error) {
        console.error('Error loading load test scenarios:', error);
        showError('Failed to load load test scenarios');
    }
}

async function renderLoadTestTable() {
    const tbody = document.getElementById('loadtest-table-body');

    if (!loadTestScenarios || loadTestScenarios.length === 0) {
        tbody.innerHTML = '<tr><td colspan="9" class="loading">No load test scenarios found</td></tr>';
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
            <td><code>${scenario.base_url || '-'}</code></td>
            <td>${scenario.concurrency || 10}</td>
            <td>${scenario.steps ? scenario.steps.length : 0} steps</td>
            <td>${accountCounts[scenario.id] || 0} accounts</td>
            <td><span class="status-badge ${scenario.is_active ? 'status-active' : 'status-inactive'}">
                ${scenario.is_active ? 'Active' : 'Inactive'}
            </span></td>
            <td>${formatDate(scenario.created_at)}</td>
            <td class="actions">
                <button class="btn btn-primary" onclick='runLoadTestScenario("${scenario.id}")' style="background: #10b981;">▶ Run</button>
                <button class="btn btn-edit" onclick='editLoadTestScenario("${scenario.id}")'>Edit</button>
                <button class="btn btn-delete" onclick="deleteLoadTestScenario('${scenario.id}')">Delete</button>
            </td>
        </tr>
    `).join('');
}

function showCreateLoadTestModal() {
    document.getElementById('loadtest-modal-title').textContent = 'Create Load Test Scenario';
    document.getElementById('loadtest-id').value = '';
    document.getElementById('loadtest-name').value = '';
    document.getElementById('loadtest-description').value = '';
    document.getElementById('loadtest-accounts-input').value = '098888888-Test123456,09575757-Test123456,0966666666-Password789';
    document.getElementById('loadtest-concurrency').value = '10';
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
    if (!confirm('Are you sure you want to run this load test?')) return;

    try {
        showSuccess('Running load test... This may take a while.');
        
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
        showSuccess('Load test completed successfully!');
    } catch (error) {
        console.error('Error running load test:', error);
        showError('Failed to run load test: ' + error.message);
    }
}

function displayLoadTestResults(result) {
    const summary = `
Load Test Results
================
Scenario: ${result.ScenarioName || 'N/A'}
Total Accounts: ${result.TotalAccounts || 0}
Success: ${result.SuccessCount || 0}
Failure: ${result.FailureCount || 0}
Total Duration: ${result.TotalDuration || 0}ms
Average Duration: ${result.AvgDuration || 0}ms

${result.AccountResults && result.AccountResults.length > 0 ? 
    'Account Results:\n' + result.AccountResults.map(acc => 
        `- ${acc.Username}: ${acc.Success ? '✓ Success' : '✗ Failed'} (${acc.TotalTime}ms)${acc.FailedStep ? ' - Failed at: ' + acc.FailedStep : ''}`
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
        showError('Failed to load scenario details');
    }
}

async function editLoadTestScenario(id) {
    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios/${id}`);
        if (!response.ok) throw new Error('Failed to load scenario');

        const scenario = await response.json();

        document.getElementById('loadtest-modal-title').textContent = 'Edit Load Test Scenario';
        document.getElementById('loadtest-id').value = scenario.id;
        document.getElementById('loadtest-name').value = scenario.name || '';
        document.getElementById('loadtest-description').value = scenario.description || '';
        document.getElementById('loadtest-accounts-input').value = scenario.accounts || '';
        document.getElementById('loadtest-concurrency').value = scenario.concurrency || 10;
        document.getElementById('loadtest-active').checked = scenario.is_active;

        // Update account count display
        updateAccountCount();

        populateStepsInForm(scenario.steps);

        showModal('loadtest-modal');
    } catch (error) {
        console.error('Error loading scenario:', error);
        showError('Failed to load scenario for editing');
    }
}

async function saveLoadTestScenario() {
    const id = document.getElementById('loadtest-id').value;
    const name = document.getElementById('loadtest-name').value.trim();
    const description = document.getElementById('loadtest-description').value.trim();
    const accounts = document.getElementById('loadtest-accounts-input').value.trim();
    const concurrency = parseInt(document.getElementById('loadtest-concurrency').value) || 10;
    const isActive = document.getElementById('loadtest-active').checked;
    const steps = getStepsFromForm();

    if (!name) {
        showError('Scenario name is required');
        return;
    }
    if (steps.length === 0) {
        showError('At least one step is required');
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
        showSuccess(id ? 'Load test scenario updated successfully' : 'Load test scenario created successfully');
        loadLoadTestScenarios();
    } catch (error) {
        console.error('Error saving load test scenario:', error);
        showError('Failed to save load test scenario');
    }
}

async function deleteLoadTestScenario(id) {
    if (!confirm('Are you sure you want to delete this load test scenario?')) return;

    try {
        const response = await fetch(`${API_BASE_URL}/loadtest/scenarios/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete load test scenario');

        showSuccess('Load test scenario deleted successfully');
        loadLoadTestScenarios();
    } catch (error) {
        console.error('Error deleting load test scenario:', error);
        showError('Failed to delete load test scenario');
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
