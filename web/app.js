const API_BASE_URL = 'http://localhost:8081/api/v1/mocktool';

let features = [];
let scenarios = [];
let mockAPIs = [];

document.addEventListener('DOMContentLoaded', function() {
    initializeTabs();
    loadFeatures();
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
    }
}

async function loadFeatures() {
    try {
        const response = await fetch(`${API_BASE_URL}/features`);
        if (!response.ok) throw new Error('Failed to load features');

        features = await response.json();
        renderFeaturesTable();
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

async function loadScenarios(featureName) {
    if (!featureName) {
        document.getElementById('scenarios-table-body').innerHTML =
            '<tr><td colspan="6" class="loading">Select a feature to view scenarios</td></tr>';
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/scenarios?feature_name=${featureName}`);
        if (!response.ok) throw new Error('Failed to load scenarios');

        scenarios = await response.json();
        renderScenariosTable();
    } catch (error) {
        console.error('Error loading scenarios:', error);
        showError('Failed to load scenarios');
    }
}

function renderScenariosTable() {
    const tbody = document.getElementById('scenarios-table-body');

    if (!scenarios || scenarios.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="loading">No scenarios found</td></tr>';
        return;
    }

    tbody.innerHTML = scenarios.map(scenario => `
        <tr>
            <td>${scenario.feature_name || 'N/A'}</td>
            <td><strong>${scenario.name || 'N/A'}</strong></td>
            <td>${scenario.description || '-'}</td>
            <td><span class="status-badge ${scenario.is_active ? 'status-active' : 'status-inactive'}">
                ${scenario.is_active ? 'Active' : 'Inactive'}
            </span></td>
            <td>${formatDate(scenario.created_at)}</td>
            <td class="actions">
                <button class="btn btn-edit" onclick='editScenario(${JSON.stringify(scenario)})'>Edit</button>
                <button class="btn btn-delete" onclick="deleteScenario('${scenario.id}')">Delete</button>
            </td>
        </tr>
    `).join('');
}

async function loadMockAPIs(scenarioName) {
    if (!scenarioName) {
        document.getElementById('mockapis-table-body').innerHTML =
            '<tr><td colspan="7" class="loading">Select a scenario to view mock APIs</td></tr>';
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/mockapis?scenario_name=${scenarioName}`);
        if (!response.ok) throw new Error('Failed to load mock APIs');

        mockAPIs = await response.json();
        renderMockAPIsTable();
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

function populateFeatureFilters() {
    const scenarioFilter = document.getElementById('scenario-feature-filter');
    scenarioFilter.innerHTML = '<option value="">Select a feature...</option>';

    features.forEach(feature => {
        const option = document.createElement('option');
        option.value = feature.name;
        option.textContent = feature.name;
        scenarioFilter.appendChild(option);
    });

    scenarioFilter.onchange = (e) => loadScenarios(e.target.value);
}

function populateMockAPIFilters() {
    const featureFilter = document.getElementById('mockapi-feature-filter');
    featureFilter.innerHTML = '<option value="">Select a feature...</option>';

    features.forEach(feature => {
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
            const response = await fetch(`${API_BASE_URL}/scenarios/active?feature_name=${featureName}`);
            const scenario = await response.json();

            const scenarioFilter = document.getElementById('mockapi-scenario-filter');
            scenarioFilter.innerHTML = '<option value="">Select a scenario...</option>';

            const option = document.createElement('option');
            option.value = scenario.name;
            option.textContent = scenario.name;
            scenarioFilter.appendChild(option);
        
        } catch (error) {
            console.error('Error loading scenarios:', error);
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

function showCreateScenarioModal() {
    document.getElementById('scenario-modal-title').textContent = 'Create Scenario';
    document.getElementById('scenario-id').value = '';
    document.getElementById('scenario-name').value = '';
    document.getElementById('scenario-description').value = '';
    document.getElementById('scenario-active').checked = true;

    const featureSelect = document.getElementById('scenario-feature');
    featureSelect.innerHTML = '<option value="">Select a feature...</option>';
    features.forEach(feature => {
        const option = document.createElement('option');
        option.value = feature.name;
        option.textContent = feature.name;
        featureSelect.appendChild(option);
    });

    showModal('scenario-modal');
}

function editScenario(scenario) {
    document.getElementById('scenario-modal-title').textContent = 'Edit Scenario';
    document.getElementById('scenario-id').value = scenario.id;
    document.getElementById('scenario-name').value = scenario.name;
    document.getElementById('scenario-description').value = scenario.description || '';
    document.getElementById('scenario-active').checked = scenario.is_active;

    const featureSelect = document.getElementById('scenario-feature');
    featureSelect.innerHTML = '<option value="">Select a feature...</option>';
    features.forEach(feature => {
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
    const isActive = document.getElementById('scenario-active').checked;

    if (!featureName || !name) {
        showError('Feature and scenario name are required');
        return;
    }

    const data = {
        feature_name: featureName,
        name: name,
        description: description,
        is_active: isActive
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

function showCreateMockAPIModal() {
    document.getElementById('mockapi-modal-title').textContent = 'Create Mock API';
    document.getElementById('mockapi-id').value = '';
    document.getElementById('mockapi-name').value = '';
    document.getElementById('mockapi-path').value = '';
    document.getElementById('mockapi-method').value = 'GET';
    document.getElementById('mockapi-regex-path').value = '';
    document.getElementById('mockapi-hash-input').value = '';
    document.getElementById('mockapi-output').value = '';
    document.getElementById('mockapi-active').checked = true;

    const featureSelect = document.getElementById('mockapi-feature');
    featureSelect.innerHTML = '<option value="">Select a feature...</option>';
    features.forEach(feature => {
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
            const response = await fetch(`${API_BASE_URL}/scenarios?feature_name=${featureName}`);
            const scenarios = await response.json();

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

    showModal('mockapi-modal');
}

function editMockAPI(api) {
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

    const featureSelect = document.getElementById('mockapi-feature');
    featureSelect.innerHTML = '<option value="">Select a feature...</option>';
    features.forEach(feature => {
        const option = document.createElement('option');
        option.value = feature.name;
        option.textContent = feature.name;
        if (feature.name === api.feature_name) {
            option.selected = true;
        }
        featureSelect.appendChild(option);
    });

    loadScenariosForEdit(api.feature_name, api.scenario_name);
    showModal('mockapi-modal');
}

async function loadScenariosForEdit(featureName, selectedScenario) {
    try {
        const response = await fetch(`${API_BASE_URL}/scenarios?feature_name=${featureName}`);
        const scenarios = await response.json();

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
    } catch (error) {
        console.error('Error loading scenarios:', error);
    }
}

async function saveMockAPI() {
    const id = document.getElementById('mockapi-id').value;
    const featureName = document.getElementById('mockapi-feature').value;
    const scenarioName = document.getElementById('mockapi-scenario').value;
    const name = document.getElementById('mockapi-name').value.trim();
    const path = document.getElementById('mockapi-path').value.trim();
    const method = document.getElementById('mockapi-method').value;
    const regexPath = document.getElementById('mockapi-regex-path').value.trim();
    const hashInputStr = document.getElementById('mockapi-hash-input').value.trim();
    const outputStr = document.getElementById('mockapi-output').value.trim();
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
