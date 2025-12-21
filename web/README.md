# Mock API Manager - Web Interface

A beautiful web interface to manage your mock API features, scenarios, and endpoints.

## Features

- **Feature Management**: Create, edit, and manage API features
- **Scenario Management**: Organize scenarios within features
- **Mock API Management**: Configure mock endpoints with custom responses
- **Modern UI**: Clean, responsive design with smooth animations
- **Real-time Updates**: Instant feedback on all operations

## Getting Started

### 1. Start Your Backend Server

Make sure your Go backend is running on `http://localhost:8080`

```bash
go run main.go service
```

### 2. Open the Web Interface

Simply open `index.html` in your web browser:

```bash
open index.html
# or
firefox index.html
# or just double-click the file
```

### 3. Configure API Base URL (if needed)

If your backend is running on a different port or host, edit `app.js` and change:

```javascript
const API_BASE_URL = 'http://localhost:8080/api/v1/mocktool';
```

## Usage

### Managing Features

1. Click on the **Features** tab
2. Click **+ New Feature** to create a feature
3. Fill in the feature name, description, and active status
4. Click **Save**

### Managing Scenarios

1. Click on the **Scenarios** tab
2. Select a feature from the dropdown
3. Click **+ New Scenario** to create a scenario
4. Fill in the scenario details
5. Click **Save**

### Managing Mock APIs

1. Click on the **Mock APIs** tab
2. Select a feature, then a scenario from the dropdowns
3. Click **+ New Mock API** to create an endpoint
4. Fill in the API details:
   - **Name**: A friendly name for the API
   - **Path**: The API path (e.g., `/api/users`)
   - **Regex Path** (optional): Pattern matching for dynamic paths
   - **Hash Input - JSON** (optional): JSON object to generate unique hash for request matching
   - **Response Output - JSON**: JSON response (must be valid JSON)
5. Click **Save**

## API Configuration

### Hash Input - JSON (Optional)

Provide a JSON object that will be hashed to create a unique identifier for request matching. This is useful when you want to match requests based on specific input parameters.

Example:
```json
{
  "userId": 123,
  "action": "login",
  "timestamp": 1234567890
}
```

The system will:
1. Parse your JSON object
2. Marshal it back to a consistent string format with **sorted keys** (alphabetically)
3. Generate a hash from that normalized string
4. Use the hash to uniquely identify and match this mock API endpoint

**Important**: The keys are automatically sorted alphabetically, so these two inputs will produce the **same hash**:
```json
{"userId": 123, "action": "login"}
{"action": "login", "userId": 123}
```
This ensures consistent hashing regardless of key order in your JSON input.

### Response Output - JSON (Required)

The response output must be valid JSON. This is what will be returned when the mock API is called.

Examples:

```json
{
  "message": "success",
  "data": {
    "id": 123,
    "name": "test"
  }
}
```

```json
{
  "users": [
    {"id": 1, "name": "John"},
    {"id": 2, "name": "Jane"}
  ]
}
```

## File Structure

```
web/
├── index.html      # Main HTML file
├── styles.css      # Styling and design
├── app.js          # JavaScript logic and API calls
└── README.md       # This file
```

## Browser Compatibility

- Chrome (recommended)
- Firefox
- Safari
- Edge

## Troubleshooting

### CORS Issues

If you encounter CORS errors, make sure your backend has CORS enabled:

```go
// In your Echo setup
e.Use(middleware.CORS())
```

### API Connection Issues

- Verify the backend is running on the correct port
- Check the `API_BASE_URL` in `app.js`
- Check browser console for errors (F12)

## Technologies Used

- Pure HTML5
- CSS3 with animations
- Vanilla JavaScript (no frameworks)
- Fetch API for HTTP requests
