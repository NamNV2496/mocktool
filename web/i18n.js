// ==================== i18n / Internationalization ====================

const I18N_STORAGE_KEY = 'mocktool_lang';

const translations = {
    en: {
        // App
        'app.title': 'Mock API Manager',
        'app.subtitle': 'Manage your features, scenarios, and mock APIs',

        // Tabs
        'tab.features': 'Features',
        'tab.scenarios': 'Scenarios',
        'tab.mockapis': 'Mock APIs',
        'tab.loadtest': 'Load Test Tool',

        // Common
        'common.active': 'Active',
        'common.inactive': 'Inactive',
        'common.edit': 'Edit',
        'common.delete': 'Delete',
        'common.duplicate': 'Duplicate',
        'common.save': 'Save',
        'common.cancel': 'Cancel',
        'common.actions': 'Actions',
        'common.search': 'Search',
        'common.createdAt': 'Created At',
        'common.description': 'Description',
        'common.active.checkbox': 'Active',

        // Features tab
        'feature.title': 'Features',
        'feature.search.placeholder': 'Search by feature name...',
        'feature.new': '+ New Feature',
        'feature.table.name': 'Features Name',
        'feature.table.description': 'Description',
        'feature.table.active': 'Active',
        'feature.table.createdAt': 'Created At',
        'feature.table.actions': 'Actions',
        'feature.loading': 'Loading features...',
        'feature.noResults': 'No features found',

        // Feature modal
        'feature.modal.create': 'Create Feature',
        'feature.modal.edit': 'Edit Feature',
        'feature.form.name.label': 'Feature Name (Cannot change)',
        'feature.form.name.placeholder': 'Enter feature name',
        'feature.form.description.placeholder': 'Enter description',

        // Feature validation
        'feature.error.nameRequired': 'Feature name is required',
        'feature.error.nameHasSpaces': 'Feature name cannot contain spaces',
        'feature.error.nameTooLong': 'Feature name is too long (max 100 characters)',

        // Feature CRUD
        'feature.success.created': 'Feature created successfully',
        'feature.success.updated': 'Feature updated successfully',
        'feature.success.deleted': 'Feature deleted successfully',
        'feature.error.load': 'Failed to load features',
        'feature.error.save': 'Failed to save feature',
        'feature.error.delete': 'Failed to delete feature',
        'feature.confirm.delete': 'Are you sure you want to delete this feature?',

        // Scenarios tab
        'scenario.title': 'Scenarios',
        'scenario.accountId.placeholder': 'Account ID to check active status',
        'scenario.feature.placeholder': 'Search features...',
        'scenario.new': '+ New Scenario',
        'scenario.table.feature': 'Feature',
        'scenario.table.name': 'Scenarios Name',
        'scenario.table.description': 'Description',
        'scenario.table.createdAt': 'Created At',
        'scenario.table.actions': 'Actions',
        'scenario.loading': 'Select a feature to view scenarios',
        'scenario.noResults': 'No scenarios found',
        'scenario.badge.active': 'ACTIVE',
        'scenario.badge.activeGlobal': 'ACTIVE GLOBAL',
        'scenario.btn.activateForAccount': 'ACTIVE FOR ACCOUNT_ID',
        'scenario.btn.activateGlobally': 'Activate Globally',

        // Scenario modal
        'scenario.modal.create': 'Create Scenario',
        'scenario.modal.edit': 'Edit Scenario',
        'scenario.form.feature.label': 'Feature',
        'scenario.form.feature.placeholder': 'Search features...',
        'scenario.form.name.label': 'Scenario Name (Cannot change)',
        'scenario.form.name.placeholder': 'Enter scenario name',
        'scenario.form.description.placeholder': 'Enter description',

        // Scenario validation
        'scenario.error.featureRequired': 'Feature is required',
        'scenario.error.nameRequired': 'Scenario name is required',
        'scenario.error.nameHasSpaces': 'Scenario name cannot contain spaces',
        'scenario.error.nameTooLong': 'Scenario name is too long (max 100 characters)',

        // Scenario CRUD
        'scenario.success.created': 'Scenario created successfully',
        'scenario.success.updated': 'Scenario updated successfully',
        'scenario.success.deleted': 'Scenario deleted successfully',
        'scenario.success.activated': 'Scenario activated for account {0}',
        'scenario.success.activatedGlobal': 'Scenario activated globally',
        'scenario.error.load': 'Failed to load scenarios',
        'scenario.error.save': 'Failed to save scenario',
        'scenario.error.delete': 'Failed to delete scenario',
        'scenario.error.activate': 'Failed to activate scenario',
        'scenario.confirm.delete': 'Are you sure you want to delete this scenario?',

        // Mock APIs tab
        'mockapi.title': 'Mock APIs',
        'mockapi.feature.placeholder': 'Search features...',
        'mockapi.scenario.placeholder': 'Search scenarios...',
        'mockapi.search.placeholder': 'Search by name or path...',
        'mockapi.new': '+ New Mock API',
        'mockapi.parseProto': 'Parse .proto',
        'mockapi.table.feature': 'Feature',
        'mockapi.table.scenario': 'Scenario',
        'mockapi.table.name': 'Mock APIs Name',
        'mockapi.table.description': 'Description',
        'mockapi.table.method': 'Method',
        'mockapi.table.path': 'Path',
        'mockapi.table.active': 'Active',
        'mockapi.table.createdAt': 'Created At',
        'mockapi.table.actions': 'Actions',
        'mockapi.loading': 'Select a scenario to view mock APIs',
        'mockapi.noResults': 'No mock APIs found',

        // Mock API modal
        'mockapi.modal.create': 'Create Mock API',
        'mockapi.modal.edit': 'Edit Mock API',
        'mockapi.modal.duplicate': 'Duplicate Mock API',
        'mockapi.form.feature.label': 'Feature',
        'mockapi.form.scenario.label': 'Scenario',
        'mockapi.form.name.label': 'API Name',
        'mockapi.form.name.placeholder': 'Enter API name',
        'mockapi.form.description.label': 'Description (Optional)',
        'mockapi.form.description.placeholder': 'Enter API description',
        'mockapi.form.path.label': 'Path',
        'mockapi.form.path.placeholder': '/api/example',
        'mockapi.form.method.label': 'HTTP Method',
        'mockapi.form.regexPath.label': 'Regex Path (Optional)',
        'mockapi.form.regexPath.placeholder': '/api/users/[0-9]+ example: /api/users/123456',
        'mockapi.form.hashInput.label': 'Request Input - JSON (Optional)',
        'mockapi.form.hashInput.hint': 'JSON object to generate unique hash for request matching. Keys are automatically sorted for consistent hashing.',
        'mockapi.form.output.label': 'Response Output - JSON (Required)',
        'mockapi.form.output.hint': 'JSON response to return when this mock API is matched',
        'mockapi.form.outputHeader.label': 'Response Output header',
        'mockapi.form.outputHeader.hint': 'Response header',
        'mockapi.form.latency.label': 'Latency (seconds)',
        'mockapi.form.latency.hint': 'Delay response by this many seconds. 0 = no delay.',

        // Mock API validation
        'mockapi.error.featureRequired': 'Feature is required',
        'mockapi.error.scenarioRequired': 'Scenario is required',
        'mockapi.error.nameRequired': 'Mock API name is required',
        'mockapi.error.nameHasSpaces': 'Mock API name cannot contain spaces',
        'mockapi.error.nameTooLong': 'Mock API name is too long (max 100 characters)',
        'mockapi.error.pathRequired': 'Path is required',
        'mockapi.error.methodOutputRequired': 'Method and output are required',
        'mockapi.error.invalidOutput': 'Invalid JSON in output field',
        'mockapi.error.invalidInput': 'Invalid JSON in input field',

        // Mock API CRUD
        'mockapi.success.created': 'Mock API created successfully',
        'mockapi.success.updated': 'Mock API updated successfully',
        'mockapi.success.deleted': 'Mock API deleted successfully',
        'mockapi.error.load': 'Failed to load mock APIs',
        'mockapi.error.save': 'Failed to save mock API',
        'mockapi.error.delete': 'Failed to delete mock API',
        'mockapi.confirm.delete': 'Are you sure you want to delete this mock API?',

        // Search
        'search.error.noScenario': 'Please select a scenario first',
        'search.error.noTerm': 'Please enter a search term',

        // Proto parser
        'proto.modal.title': 'Parse .proto File',
        'proto.form.upload.label': 'Upload .proto file',
        'proto.form.paste.label': 'Or paste .proto content',
        'proto.form.paste.placeholder': 'Paste your .proto file content here...',
        'proto.btn.parse': 'Parse',
        'proto.error.empty': 'Please enter or upload a .proto file',
        'proto.noMessages': 'No messages found in the .proto file.',
        'proto.btn.useAsInput': 'Use as Input',
        'proto.btn.useAsOutput': 'Use as Output',
        'proto.btn.createMockAPI': 'Create Mock API',

        // JSON format
        'json.format.btn': '⚡ Format',
        'json.mode.raw': 'Raw',
        'json.mode.tree': 'Tree',
        'json.format.success': 'JSON formatted and sorted successfully!',
        'json.format.error.empty': 'No JSON to format',
        'json.format.error.invalid': 'Invalid JSON',
        'json.format.error.tip.braces': '\n\nTip: You may have extra braces. Check for duplicate { or } characters.',
        'json.format.error.tip.trailing': '\n\nTip: Remove trailing commas before } or ].',
        'json.format.error.tip.start': '\n\nTip: JSON must start with { or [.',

        // Load test tab
        'loadtest.title': 'Load Test Scenarios',
        'loadtest.search.placeholder': 'Search by scenario name...',
        'loadtest.new': '+ New Scenario',
        'loadtest.import': '📁 Import JSON',
        'loadtest.table.name': 'LoadTest_Name',
        'loadtest.table.description': 'Description',
        'loadtest.table.steps': 'Steps',
        'loadtest.table.accounts': 'Accounts',
        'loadtest.table.active': 'Active',
        'loadtest.table.createdAt': 'Created At',
        'loadtest.table.actions': 'Actions',
        'loadtest.loading': 'Loading load test scenarios...',
        'loadtest.noResults': 'No load test scenarios found',
        'loadtest.table.stepsCount': '{0} steps',
        'loadtest.table.accountsCount': '{0} accounts',

        // Load test modal
        'loadtest.modal.create': 'Create Load Test Scenario',
        'loadtest.modal.edit': 'Edit Load Test Scenario',
        'loadtest.modal.duplicate': 'Duplicate Load Test Scenario',
        'loadtest.form.name.label': 'Scenario Name',
        'loadtest.form.name.placeholder': 'Enter scenario name',
        'loadtest.form.description.label': 'Description',
        'loadtest.form.description.placeholder': 'Enter description',
        'loadtest.form.accounts.label': 'Test Accounts (Format: username-password, comma separated)',
        'loadtest.form.accounts.hint': 'Enter username-password pairs separated by commas. These accounts will be used as {{username}} and {{password}} variables in test steps.',
        'loadtest.form.accounts.count': 'Accounts count:',
        'loadtest.form.steps.label': 'Steps',
        'loadtest.form.addStep': '+ Add Step',

        // Load test step template
        'loadtest.step.title': 'Step',
        'loadtest.step.remove': 'Remove',
        'loadtest.step.name.label': 'Step Name',
        'loadtest.step.name.placeholder': 'e.g., login, get_profile',
        'loadtest.step.condition.label': '⚡ Execution Condition (optional)',
        'loadtest.step.path.label': 'Path or cURL Command',
        'loadtest.step.method.label': 'Method (override cURL)',
        'loadtest.step.method.auto': 'Auto (from cURL or GET)',
        'loadtest.step.status.label': 'Expected Status Code',
        'loadtest.step.retry.label': 'Retry for Seconds (0 = no retry)',
        'loadtest.step.retry.hint': 'Retry failed requests for this many seconds (1s interval)',
        'loadtest.step.maxRetry.label': 'Max Retry Times (0 = unlimited)',
        'loadtest.step.maxRetry.hint': 'Maximum retry attempts (stops when reached or time expires)',
        'loadtest.step.wait.label': 'Wait After Seconds (0 = no wait)',
        'loadtest.step.wait.hint': 'Wait after successful execution before next step',
        'loadtest.step.headers.label': 'Headers (override cURL headers)',
        'loadtest.step.headers.hint': 'Optional. Add or override headers from cURL.',
        'loadtest.step.body.label': 'Request Body (override cURL body)',
        'loadtest.step.body.hint': 'Optional. Override body from cURL if needed.',
        'loadtest.step.variables.label': 'Extract Variables from Response and SAVE FOR FUTURE',
        'loadtest.step.addVariable': '+ Add Variable',
        'loadtest.step.variables.hint': 'Extract values from response to use in next steps with {{variable_name}}',

        // Load test validation
        'loadtest.error.nameRequired': 'Scenario name is required',
        'loadtest.error.nameHasSpaces': 'Load test scenario name cannot contain spaces',
        'loadtest.error.nameTooLong': 'Load test scenario name is too long (max 100 characters)',
        'loadtest.error.stepsRequired': 'At least one step is required',

        // Load test CRUD
        'loadtest.success.created': 'Load test scenario created successfully',
        'loadtest.success.updated': 'Load test scenario updated successfully',
        'loadtest.success.deleted': 'Load test scenario deleted successfully',
        'loadtest.success.exported': 'Scenario exported successfully!',
        'loadtest.error.load': 'Failed to load load test scenarios',
        'loadtest.error.save': 'Failed to save load test scenario',
        'loadtest.error.delete': 'Failed to delete load test scenario',
        'loadtest.error.run': 'Failed to run load test',
        'loadtest.confirm.delete': 'Are you sure you want to delete this load test scenario?',
        'loadtest.confirm.run': 'Are you sure you want to run this load test?',

        // Load test results
        'loadtest.results.title': 'Load Test Results',
        'loadtest.results.scenario': 'Scenario',
        'loadtest.results.totalAccounts': 'Total Accounts',
        'loadtest.results.success': 'Success',
        'loadtest.results.failure': 'Failure',
        'loadtest.results.totalDuration': 'Total Duration',
        'loadtest.results.avgDuration': 'Average Duration',
        'loadtest.results.accountResults': 'Account Results',
        'loadtest.results.successMark': '✓ Success',
        'loadtest.results.failMark': '✗ Failed',
        'loadtest.results.failedAt': 'Failed at',

        // Load test action buttons
        'loadtest.btn.run': '▶ Run',
        'loadtest.btn.export': '📥 Export',

        // Pagination
        'pagination.prev': 'Prev',
        'pagination.next': 'Next',

        // Dropdown
        'dropdown.noResults': 'No results found',
        'dropdown.error': 'Error searching...',

        // Import/Export
        'import.duplicateName': 'A scenario with the name "{0}" already exists.\n\nDo you want to import anyway? This will create a duplicate.\n\nClick OK to import as duplicate, or Cancel to abort.',
        'import.cancelled': 'Import cancelled - scenario name already exists',
        'import.success': 'Scenario "{0}" imported successfully!',
        'import.error': 'Failed to import JSON',
        'import.error.noName': 'Scenario name is required',
        'import.error.noSteps': 'At least one step is required',

        // Variable row template
        'variable.name.placeholder': 'Variable name (e.g., token)',
        'variable.jsonpath.placeholder': 'data.token or header.X-Header-Name',
    },

    vi: {
        // App
        'app.title': 'Quản lý Mock API',
        'app.subtitle': 'Quản lý các tính năng, kịch bản và Mock API của bạn',

        // Tabs
        'tab.features': 'Tính năng',
        'tab.scenarios': 'Kịch bản',
        'tab.mockapis': 'Mock API',
        'tab.loadtest': 'Công cụ kiểm tra tải',

        // Common
        'common.active': 'Hoạt động',
        'common.inactive': 'Không hoạt động',
        'common.edit': 'Sửa',
        'common.delete': 'Xóa',
        'common.duplicate': 'Nhân bản',
        'common.save': 'Lưu',
        'common.cancel': 'Hủy',
        'common.search': 'Tìm kiếm',
        'common.actions': 'Thao tác',
        'common.createdAt': 'Ngày tạo',
        'common.description': 'Mô tả',
        'common.active.checkbox': 'Hoạt động',

        // Features tab
        'feature.title': 'Tính năng',
        'feature.search.placeholder': 'Tìm kiếm theo tên tính năng...',
        'feature.new': '+ Tính năng mới',
        'feature.table.name': 'Tên tính năng',
        'feature.table.description': 'Mô tả',
        'feature.table.active': 'Hoạt động',
        'feature.table.createdAt': 'Ngày tạo',
        'feature.table.actions': 'Thao tác',
        'feature.loading': 'Đang tải tính năng...',
        'feature.noResults': 'Không tìm thấy tính năng',

        // Feature modal
        'feature.modal.create': 'Tạo tính năng',
        'feature.modal.edit': 'Sửa tính năng',
        'feature.form.name.label': 'Tên tính năng (Không thể thay đổi)',
        'feature.form.name.placeholder': 'Nhập tên tính năng',
        'feature.form.description.placeholder': 'Nhập mô tả',

        // Feature validation
        'feature.error.nameRequired': 'Tên tính năng là bắt buộc',
        'feature.error.nameHasSpaces': 'Tên tính năng không được chứa khoảng trắng',
        'feature.error.nameTooLong': 'Tên tính năng quá dài (tối đa 100 ký tự)',

        // Feature CRUD
        'feature.success.created': 'Tạo tính năng thành công',
        'feature.success.updated': 'Cập nhật tính năng thành công',
        'feature.success.deleted': 'Xóa tính năng thành công',
        'feature.error.load': 'Không thể tải tính năng',
        'feature.error.save': 'Không thể lưu tính năng',
        'feature.error.delete': 'Không thể xóa tính năng',
        'feature.confirm.delete': 'Bạn có chắc chắn muốn xóa tính năng này?',

        // Scenarios tab
        'scenario.title': 'Kịch bản',
        'scenario.accountId.placeholder': 'Account ID để kiểm tra trạng thái hoạt động',
        'scenario.feature.placeholder': 'Tìm kiếm tính năng...',
        'scenario.new': '+ Kịch bản mới',
        'scenario.table.feature': 'Tính năng',
        'scenario.table.name': 'Tên kịch bản',
        'scenario.table.description': 'Mô tả',
        'scenario.table.createdAt': 'Ngày tạo',
        'scenario.table.actions': 'Thao tác',
        'scenario.loading': 'Chọn tính năng để xem kịch bản',
        'scenario.noResults': 'Không tìm thấy kịch bản',
        'scenario.badge.active': 'ĐANG DÙNG',
        'scenario.badge.activeGlobal': 'TOÀN CẦU',
        'scenario.btn.activateForAccount': 'KÍCH HOẠT CHO ACCOUNT_ID',
        'scenario.btn.activateGlobally': 'Kích hoạt toàn cầu',

        // Scenario modal
        'scenario.modal.create': 'Tạo kịch bản',
        'scenario.modal.edit': 'Sửa kịch bản',
        'scenario.form.feature.label': 'Tính năng',
        'scenario.form.feature.placeholder': 'Tìm kiếm tính năng...',
        'scenario.form.name.label': 'Tên kịch bản (Không thể thay đổi)',
        'scenario.form.name.placeholder': 'Nhập tên kịch bản',
        'scenario.form.description.placeholder': 'Nhập mô tả',

        // Scenario validation
        'scenario.error.featureRequired': 'Tính năng là bắt buộc',
        'scenario.error.nameRequired': 'Tên kịch bản là bắt buộc',
        'scenario.error.nameHasSpaces': 'Tên kịch bản không được chứa khoảng trắng',
        'scenario.error.nameTooLong': 'Tên kịch bản quá dài (tối đa 100 ký tự)',

        // Scenario CRUD
        'scenario.success.created': 'Tạo kịch bản thành công',
        'scenario.success.updated': 'Cập nhật kịch bản thành công',
        'scenario.success.deleted': 'Xóa kịch bản thành công',
        'scenario.success.activated': 'Đã kích hoạt kịch bản cho tài khoản {0}',
        'scenario.success.activatedGlobal': 'Đã kích hoạt kịch bản toàn cầu',
        'scenario.error.load': 'Không thể tải kịch bản',
        'scenario.error.save': 'Không thể lưu kịch bản',
        'scenario.error.delete': 'Không thể xóa kịch bản',
        'scenario.error.activate': 'Không thể kích hoạt kịch bản',
        'scenario.confirm.delete': 'Bạn có chắc chắn muốn xóa kịch bản này?',

        // Mock APIs tab
        'mockapi.title': 'Mock API',
        'mockapi.feature.placeholder': 'Tìm kiếm tính năng...',
        'mockapi.scenario.placeholder': 'Tìm kiếm kịch bản...',
        'mockapi.search.placeholder': 'Tìm kiếm theo tên hoặc đường dẫn...',
        'mockapi.new': '+ Mock API mới',
        'mockapi.parseProto': 'Phân tích .proto',
        'mockapi.table.feature': 'Tính năng',
        'mockapi.table.scenario': 'Kịch bản',
        'mockapi.table.name': 'Tên Mock API',
        'mockapi.table.description': 'Mô tả',
        'mockapi.table.method': 'Phương thức',
        'mockapi.table.path': 'Đường dẫn',
        'mockapi.table.active': 'Hoạt động',
        'mockapi.table.createdAt': 'Ngày tạo',
        'mockapi.table.actions': 'Thao tác',
        'mockapi.loading': 'Chọn kịch bản để xem Mock API',
        'mockapi.noResults': 'Không tìm thấy Mock API',

        // Mock API modal
        'mockapi.modal.create': 'Tạo Mock API',
        'mockapi.modal.edit': 'Sửa Mock API',
        'mockapi.modal.duplicate': 'Nhân bản Mock API',
        'mockapi.form.feature.label': 'Tính năng',
        'mockapi.form.scenario.label': 'Kịch bản',
        'mockapi.form.name.label': 'Tên API',
        'mockapi.form.name.placeholder': 'Nhập tên API',
        'mockapi.form.description.label': 'Mô tả (Tùy chọn)',
        'mockapi.form.description.placeholder': 'Nhập mô tả API',
        'mockapi.form.path.label': 'Đường dẫn',
        'mockapi.form.path.placeholder': '/api/example',
        'mockapi.form.method.label': 'Phương thức HTTP',
        'mockapi.form.regexPath.label': 'Đường dẫn Regex (Tùy chọn)',
        'mockapi.form.regexPath.placeholder': '/api/users/[0-9]+ ví dụ: /api/users/123456',
        'mockapi.form.hashInput.label': 'Dữ liệu đầu vào - JSON (Tùy chọn)',
        'mockapi.form.hashInput.hint': 'Đối tượng JSON để tạo hash duy nhất cho việc khớp yêu cầu. Các khóa được tự động sắp xếp để băm nhất quán.',
        'mockapi.form.output.label': 'Dữ liệu đầu ra - JSON (Bắt buộc)',
        'mockapi.form.output.hint': 'Phản hồi JSON trả về khi Mock API này được khớp',
        'mockapi.form.outputHeader.label': 'Header phản hồi',
        'mockapi.form.outputHeader.hint': 'Header phản hồi',
        'mockapi.form.latency.label': 'Độ trễ (giây)',
        'mockapi.form.latency.hint': 'Trì hoãn phản hồi bấy nhiêu giây. 0 = không trì hoãn.',

        // Mock API validation
        'mockapi.error.featureRequired': 'Tính năng là bắt buộc',
        'mockapi.error.scenarioRequired': 'Kịch bản là bắt buộc',
        'mockapi.error.nameRequired': 'Tên Mock API là bắt buộc',
        'mockapi.error.nameHasSpaces': 'Tên Mock API không được chứa khoảng trắng',
        'mockapi.error.nameTooLong': 'Tên Mock API quá dài (tối đa 100 ký tự)',
        'mockapi.error.pathRequired': 'Đường dẫn là bắt buộc',
        'mockapi.error.methodOutputRequired': 'Phương thức và dữ liệu đầu ra là bắt buộc',
        'mockapi.error.invalidOutput': 'JSON không hợp lệ trong trường dữ liệu đầu ra',
        'mockapi.error.invalidInput': 'JSON không hợp lệ trong trường dữ liệu đầu vào',

        // Mock API CRUD
        'mockapi.success.created': 'Tạo Mock API thành công',
        'mockapi.success.updated': 'Cập nhật Mock API thành công',
        'mockapi.success.deleted': 'Xóa Mock API thành công',
        'mockapi.error.load': 'Không thể tải Mock API',
        'mockapi.error.save': 'Không thể lưu Mock API',
        'mockapi.error.delete': 'Không thể xóa Mock API',
        'mockapi.confirm.delete': 'Bạn có chắc chắn muốn xóa Mock API này?',

        // Search
        'search.error.noScenario': 'Vui lòng chọn kịch bản trước',
        'search.error.noTerm': 'Vui lòng nhập từ khóa tìm kiếm',

        // Proto parser
        'proto.modal.title': 'Phân tích file .proto',
        'proto.form.upload.label': 'Tải lên file .proto',
        'proto.form.paste.label': 'Hoặc dán nội dung .proto',
        'proto.form.paste.placeholder': 'Dán nội dung file .proto vào đây...',
        'proto.btn.parse': 'Phân tích',
        'proto.error.empty': 'Vui lòng nhập hoặc tải lên file .proto',
        'proto.noMessages': 'Không tìm thấy message trong file .proto.',
        'proto.btn.useAsInput': 'Dùng làm đầu vào',
        'proto.btn.useAsOutput': 'Dùng làm đầu ra',
        'proto.btn.createMockAPI': 'Tạo Mock API',

        // JSON format
        'json.format.btn': '⚡ Định dạng',
        'json.mode.raw': 'Thô',
        'json.mode.tree': 'Cây',
        'json.format.success': 'JSON đã được định dạng và sắp xếp thành công!',
        'json.format.error.empty': 'Không có JSON để định dạng',
        'json.format.error.invalid': 'JSON không hợp lệ',
        'json.format.error.tip.braces': '\n\nGợi ý: Có thể bạn có dấu ngoặc thừa. Kiểm tra các ký tự { hoặc } trùng lặp.',
        'json.format.error.tip.trailing': '\n\nGợi ý: Xóa dấu phẩy thừa trước } hoặc ].',
        'json.format.error.tip.start': '\n\nGợi ý: JSON phải bắt đầu bằng { hoặc [.',

        // Load test tab
        'loadtest.title': 'Kịch bản kiểm tra tải',
        'loadtest.search.placeholder': 'Tìm kiếm theo tên kịch bản...',
        'loadtest.new': '+ Kịch bản mới',
        'loadtest.import': '📁 Nhập JSON',
        'loadtest.table.name': 'Tên kiểm tra tải',
        'loadtest.table.description': 'Mô tả',
        'loadtest.table.steps': 'Bước',
        'loadtest.table.accounts': 'Tài khoản',
        'loadtest.table.active': 'Hoạt động',
        'loadtest.table.createdAt': 'Ngày tạo',
        'loadtest.table.actions': 'Thao tác',
        'loadtest.loading': 'Đang tải kịch bản kiểm tra tải...',
        'loadtest.noResults': 'Không tìm thấy kịch bản kiểm tra tải',
        'loadtest.table.stepsCount': '{0} bước',
        'loadtest.table.accountsCount': '{0} tài khoản',

        // Load test modal
        'loadtest.modal.create': 'Tạo kịch bản kiểm tra tải',
        'loadtest.modal.edit': 'Sửa kịch bản kiểm tra tải',
        'loadtest.modal.duplicate': 'Nhân bản kịch bản kiểm tra tải',
        'loadtest.form.name.label': 'Tên kịch bản',
        'loadtest.form.name.placeholder': 'Nhập tên kịch bản',
        'loadtest.form.description.label': 'Mô tả',
        'loadtest.form.description.placeholder': 'Nhập mô tả',
        'loadtest.form.accounts.label': 'Tài khoản kiểm tra (Định dạng: username-password, cách nhau bằng dấu phẩy)',
        'loadtest.form.accounts.hint': 'Nhập các cặp username-password cách nhau bằng dấu phẩy. Các tài khoản này sẽ được dùng làm biến {{username}} và {{password}} trong các bước kiểm tra.',
        'loadtest.form.accounts.count': 'Số tài khoản:',
        'loadtest.form.steps.label': 'Bước',
        'loadtest.form.addStep': '+ Thêm bước',

        // Load test step template
        'loadtest.step.title': 'Bước',
        'loadtest.step.remove': 'Xóa',
        'loadtest.step.name.label': 'Tên bước',
        'loadtest.step.name.placeholder': 'Ví dụ: login, get_profile',
        'loadtest.step.condition.label': '⚡ Điều kiện thực thi (tùy chọn)',
        'loadtest.step.path.label': 'Đường dẫn hoặc lệnh cURL',
        'loadtest.step.method.label': 'Phương thức (ghi đè cURL)',
        'loadtest.step.method.auto': 'Tự động (từ cURL hoặc GET)',
        'loadtest.step.status.label': 'Mã trạng thái mong đợi',
        'loadtest.step.retry.label': 'Thời gian thử lại (giây, 0 = không thử lại)',
        'loadtest.step.retry.hint': 'Thử lại các yêu cầu thất bại trong khoảng thời gian này (1s mỗi lần)',
        'loadtest.step.maxRetry.label': 'Số lần thử tối đa (0 = không giới hạn)',
        'loadtest.step.maxRetry.hint': 'Số lần thử tối đa (dừng khi đạt hoặc hết thời gian)',
        'loadtest.step.wait.label': 'Chờ sau khi thực thi (giây, 0 = không chờ)',
        'loadtest.step.wait.hint': 'Chờ sau khi thực thi thành công trước bước tiếp theo',
        'loadtest.step.headers.label': 'Headers (ghi đè headers cURL)',
        'loadtest.step.headers.hint': 'Tùy chọn. Thêm hoặc ghi đè headers từ cURL.',
        'loadtest.step.body.label': 'Nội dung yêu cầu (ghi đè nội dung cURL)',
        'loadtest.step.body.hint': 'Tùy chọn. Ghi đè nội dung từ cURL nếu cần.',
        'loadtest.step.variables.label': 'Trích xuất biến từ phản hồi và LƯU CHO SAU',
        'loadtest.step.addVariable': '+ Thêm biến',
        'loadtest.step.variables.hint': 'Trích xuất giá trị từ phản hồi để sử dụng ở các bước tiếp theo với {{tên_biến}}',

        // Load test validation
        'loadtest.error.nameRequired': 'Tên kịch bản là bắt buộc',
        'loadtest.error.nameHasSpaces': 'Tên kịch bản kiểm tra tải không được chứa khoảng trắng',
        'loadtest.error.nameTooLong': 'Tên kịch bản kiểm tra tải quá dài (tối đa 100 ký tự)',
        'loadtest.error.stepsRequired': 'Cần ít nhất một bước',

        // Load test CRUD
        'loadtest.success.created': 'Tạo kịch bản kiểm tra tải thành công',
        'loadtest.success.updated': 'Cập nhật kịch bản kiểm tra tải thành công',
        'loadtest.success.deleted': 'Xóa kịch bản kiểm tra tải thành công',
        'loadtest.success.exported': 'Xuất kịch bản thành công!',
        'loadtest.error.load': 'Không thể tải kịch bản kiểm tra tải',
        'loadtest.error.save': 'Không thể lưu kịch bản kiểm tra tải',
        'loadtest.error.delete': 'Không thể xóa kịch bản kiểm tra tải',
        'loadtest.error.run': 'Không thể chạy kiểm tra tải',
        'loadtest.confirm.delete': 'Bạn có chắc chắn muốn xóa kịch bản kiểm tra tải này?',
        'loadtest.confirm.run': 'Bạn có chắc chắn muốn chạy kiểm tra tải này?',

        // Load test results
        'loadtest.results.title': 'Kết quả kiểm tra tải',
        'loadtest.results.scenario': 'Kịch bản',
        'loadtest.results.totalAccounts': 'Tổng số tài khoản',
        'loadtest.results.success': 'Thành công',
        'loadtest.results.failure': 'Thất bại',
        'loadtest.results.totalDuration': 'Tổng thời gian',
        'loadtest.results.avgDuration': 'Thời gian trung bình',
        'loadtest.results.accountResults': 'Kết quả theo tài khoản',
        'loadtest.results.successMark': '✓ Thành công',
        'loadtest.results.failMark': '✗ Thất bại',
        'loadtest.results.failedAt': 'Thất bại tại',

        // Load test action buttons
        'loadtest.btn.run': '▶ Chạy',
        'loadtest.btn.export': '📥 Xuất',

        // Pagination
        'pagination.prev': 'Trước',
        'pagination.next': 'Tiếp',

        // Dropdown
        'dropdown.noResults': 'Không tìm thấy kết quả',
        'dropdown.error': 'Lỗi tìm kiếm...',

        // Import/Export
        'import.duplicateName': 'Kịch bản tên "{0}" đã tồn tại.\n\nBạn có muốn nhập không? Điều này sẽ tạo bản trùng lặp.\n\nNhấn OK để nhập bản trùng lặp, hoặc Hủy để dừng.',
        'import.cancelled': 'Đã hủy nhập - tên kịch bản đã tồn tại',
        'import.success': 'Nhập kịch bản "{0}" thành công!',
        'import.error': 'Không thể nhập JSON',
        'import.error.noName': 'Tên kịch bản là bắt buộc',
        'import.error.noSteps': 'Cần ít nhất một bước',

        // Variable row template
        'variable.name.placeholder': 'Tên biến (ví dụ: token)',
        'variable.jsonpath.placeholder': 'data.token hoặc header.X-Header-Name',
    }
};

let currentLang = localStorage.getItem(I18N_STORAGE_KEY) || 'en';

/**
 * Translate a key with optional positional arguments.
 * Usage: t('key') or t('key', arg0, arg1, ...)
 * Use {0}, {1}, ... in translation strings for interpolation.
 */
function t(key) {
    const args = Array.prototype.slice.call(arguments, 1);
    const lang = translations[currentLang] || translations['en'];
    let str = lang[key];
    if (str === undefined) {
        str = translations['en'][key];
    }
    if (str === undefined) {
        return key;
    }
    args.forEach(function(arg, i) {
        str = str.replace(new RegExp('\\{' + i + '\\}', 'g'), arg);
    });
    return str;
}

/**
 * Apply translations to all elements with data-i18n* attributes.
 * Accepts a root element/fragment or defaults to the whole document.
 */
function applyTranslations(root) {
    var scope = root || document;
    var query = scope.querySelectorAll ? scope : document;

    query.querySelectorAll('[data-i18n]').forEach(function(el) {
        var key = el.getAttribute('data-i18n');
        if (key) el.textContent = t(key);
    });

    query.querySelectorAll('[data-i18n-placeholder]').forEach(function(el) {
        var key = el.getAttribute('data-i18n-placeholder');
        if (key) el.placeholder = t(key);
    });

    query.querySelectorAll('[data-i18n-title]').forEach(function(el) {
        var key = el.getAttribute('data-i18n-title');
        if (key) el.title = t(key);
    });

    query.querySelectorAll('[data-i18n-html]').forEach(function(el) {
        var key = el.getAttribute('data-i18n-html');
        if (key) el.innerHTML = t(key);
    });
}

/**
 * Switch the active language, persist the choice, and re-render the UI.
 */
function setLanguage(lang) {
    if (!translations[lang]) return;
    currentLang = lang;
    localStorage.setItem(I18N_STORAGE_KEY, lang);
    document.documentElement.lang = lang;
    applyTranslations();

    document.querySelectorAll('.lang-btn').forEach(function(btn) {
        btn.classList.toggle('active', btn.dataset.lang === lang);
    });
}

document.addEventListener('DOMContentLoaded', function() {
    document.documentElement.lang = currentLang;
    applyTranslations();
    document.querySelectorAll('.lang-btn').forEach(function(btn) {
        btn.classList.toggle('active', btn.dataset.lang === currentLang);
    });
});
