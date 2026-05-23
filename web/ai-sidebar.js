// AI Assistant Sidebar — self-contained module
// Communicates with POST /api/v1/mocktool/chat

(function () {
  'use strict';

  const CHAT_URL = API_BASE_URL + '/chat';

  const SUGGESTIONS = [
    'How many APIs does feature insertAd have?',
    'List all features',
    'What scenarios exist for feature checkoutFlow?',
    'Show active scenario for feature login',
    'Create a mock API for POST /api/users returning {"id":1}',
  ];

  const SLASH_COMMANDS = [
    { cmd: '/list-features', desc: 'List all features' },
    { cmd: '/list-scenarios', desc: 'List scenarios for a feature' },
    { cmd: '/list-apis', desc: 'List APIs for a scenario' },
    { cmd: '/curl', desc: 'Get cURL command for a mock API' },
    { cmd: '/proto', desc: 'Paste a .proto file → generate mock APIs' },
    { cmd: '/mock', desc: 'Paste an API contract → generate all test cases' },
    { cmd: '/search', desc: 'Search mock APIs by name or path' },
    { cmd: '/activate', desc: 'Activate a scenario' },
    { cmd: '/create-api', desc: 'Create a new mock API' },
    { cmd: '/delete', desc: 'Delete a feature, scenario, or API' },
    { cmd: '/help', desc: 'Show what I can do' },
  ];

  // Conversation history kept client-side (stateless backend)
  let history = [];
  let isOpen = false;
  let isThinking = false;

  const CHAT_HISTORY_KEY = 'ai_chat_history';

  // ───────────── localStorage helpers ─────────────

  function saveHistory() {
    try {
      localStorage.setItem(CHAT_HISTORY_KEY, JSON.stringify(history));
    } catch (err) {
      console.warn('Failed to save chat history:', err);
    }
  }

  function loadHistory() {
    try {
      const raw = localStorage.getItem(CHAT_HISTORY_KEY);
      if (!raw) return false;
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        history = parsed;
        return true;
      }
    } catch (err) {
      console.warn('Failed to load chat history:', err);
    }
    return false;
  }

  function clearHistory() {
    localStorage.removeItem(CHAT_HISTORY_KEY);
  }

  // ───────────── DOM helpers ─────────────

  function el(id) { return document.getElementById(id); }

  function createSidebar() {
    const sidebar = document.createElement('div');
    sidebar.id = 'ai-sidebar';
    sidebar.className = 'ai-sidebar';
    
    // Load saved width from localStorage, default 400px
    const savedWidth = localStorage.getItem('ai-sidebar-width');
    if (savedWidth) {
      sidebar.style.width = savedWidth + 'px';
    }
    
    sidebar.innerHTML = `
      <div class="ai-sidebar-resizer" id="ai-sidebar-resizer"></div>
      <div class="ai-sidebar-header">
        <span class="ai-sidebar-title">
          <span class="ai-icon">🤖</span> AI ASSIST
        </span>
        <div class="ai-sidebar-header-actions">
          <button class="ai-icon-btn" id="ai-clear-btn" title="New conversation">✕ New</button>
          <button class="ai-icon-btn ai-close-btn" id="ai-close-btn" title="Close">✕</button>
        </div>
      </div>

      <div class="ai-messages" id="ai-messages">
        <div class="ai-welcome" id="ai-welcome">
          <div class="ai-welcome-section">
            <div class="ai-section-label">Try asking</div>
            <div class="ai-suggestions" id="ai-suggestions"></div>
          </div>
          <div class="ai-welcome-section">
            <div class="ai-section-label">Slash commands</div>
            <div class="ai-commands" id="ai-commands"></div>
          </div>
        </div>
      </div>

      <!-- Proto panel (hidden by default) -->
      <div class="ai-proto-panel" id="ai-proto-panel" style="display:none">
        <div class="ai-proto-panel-header">
          <span class="ai-proto-title">📋 Proto → Mock Generator</span>
          <button class="ai-icon-btn" id="ai-proto-cancel">✕</button>
        </div>
        <div class="ai-proto-fields">
          <input id="ai-proto-feature" class="ai-proto-input" placeholder="Feature name" />
          <input id="ai-proto-scenario" class="ai-proto-input" placeholder="Scenario name" />
        </div>
        <textarea
          id="ai-proto-content"
          class="ai-proto-textarea"
          placeholder="Paste your .proto file content here…"
          rows="8"
          spellcheck="false"
        ></textarea>
        <div class="ai-proto-footer">
          <span class="ai-proto-hint" id="ai-proto-hint"></span>
          <button class="ai-proto-submit" id="ai-proto-submit">⚡ Generate Mocks</button>
        </div>
      </div>

      <!-- Mock panel (hidden by default) -->
      <div class="ai-proto-panel" id="ai-mock-panel" style="display:none">
        <div class="ai-proto-panel-header">
          <span class="ai-proto-title">📄 Contract → Mock Generator</span>
          <button class="ai-icon-btn" id="ai-mock-cancel">✕</button>
        </div>
        <div class="ai-proto-fields">
          <input id="ai-mock-feature" class="ai-proto-input" placeholder="Feature name (optional)" />
          <input id="ai-mock-scenario" class="ai-proto-input" placeholder="Scenario prefix (optional)" />
        </div>
        <textarea
          id="ai-mock-content"
          class="ai-proto-textarea"
          placeholder="Paste your API contract here…

Example:
API-001: GET — /api/referrals
Request: {&quot;vertical&quot;: &quot;JOBS&quot;}
Response: {&quot;data&quot;: {...}, &quot;success&quot;: true}

or

Response: {&quot;data&quot;: {...}, &quot;success&quot;: true}

Error:
{&quot;error_code&quot;: &quot;ERR-001&quot;, &quot;error_message&quot;: &quot;Login required&quot;}"
          rows="10"
          spellcheck="false"
        ></textarea>
        <div class="ai-proto-footer">
          <span class="ai-proto-hint" id="ai-mock-hint"></span>
          <button class="ai-proto-submit" id="ai-mock-submit">⚡ Generate Cases</button>
        </div>
      </div>

      <div class="ai-input-area">
        <div class="ai-input-wrapper">
          <textarea
            id="ai-input"
            class="ai-input"
            placeholder="Ask anything..."
            rows="1"
          ></textarea>
          <button class="ai-send-btn" id="ai-send-btn" title="Send">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
              <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/>
            </svg>
          </button>
        </div>
        <div class="ai-input-hint">Press Enter to send · Shift+Enter for newline</div>
      </div>
    `;
    document.body.appendChild(sidebar);
  }

  function createToggleButton() {
    const btn = document.createElement('button');
    btn.id = 'ai-toggle-btn';
    btn.className = 'ai-toggle-btn';
    btn.innerHTML = '<span class="ai-icon">🤖</span> AI ASSIST';
    btn.setAttribute('title', 'Open AI assistant');
    document.body.appendChild(btn);
  }

  function populateSuggestions() {
    const container = el('ai-suggestions');
    SUGGESTIONS.forEach(s => {
      const chip = document.createElement('button');
      chip.className = 'ai-suggestion-chip';
      chip.textContent = s;
      chip.onclick = () => sendMessage(s);
      container.appendChild(chip);
    });
  }

  function populateCommands() {
    const container = el('ai-commands');
    SLASH_COMMANDS.forEach(({ cmd, desc }) => {
      const row = document.createElement('div');
      row.className = 'ai-command-row';
      row.innerHTML = `<span class="ai-command-name">${cmd}</span><span class="ai-command-desc">${desc}</span>`;
      row.onclick = () => {
        if (cmd === '/proto') {
          showProtoPanel();
        } else if (cmd === '/mock') {
          showMockPanel();
        } else {
          el('ai-input').value = cmd + ' ';
          el('ai-input').focus();
          autoResize(el('ai-input'));
        }
      };
      container.appendChild(row);
    });
  }

  // ───────────── Sidebar open/close ─────────────

  function openSidebar() {
    isOpen = true;
    el('ai-sidebar').classList.add('open');
    el('ai-toggle-btn').style.display = 'none';
    el('ai-input').focus();
  }

  function closeSidebar() {
    isOpen = false;
    el('ai-sidebar').classList.remove('open');
    el('ai-toggle-btn').style.display = '';
  }

  function toggleSidebar() {
    if (isOpen) closeSidebar(); else openSidebar();
  }

  // ───────────── Message rendering ─────────────

  function appendMessage(role, text) {
    const welcome = el('ai-welcome');
    if (welcome) welcome.style.display = 'none';

    const msgs = el('ai-messages');
    const bubble = document.createElement('div');
    bubble.className = 'ai-bubble ai-bubble-' + role;

    const label = document.createElement('div');
    label.className = 'ai-bubble-label';
    label.textContent = role === 'user' ? 'You' : 'AI';

    const content = document.createElement('div');
    content.className = 'ai-bubble-content';
    content.innerHTML = formatMessage(text);

    bubble.appendChild(label);
    bubble.appendChild(content);
    msgs.appendChild(bubble);
    msgs.scrollTop = msgs.scrollHeight;
    return bubble;
  }

  function showThinking() {
    const msgs = el('ai-messages');
    const bubble = document.createElement('div');
    bubble.id = 'ai-thinking-bubble';
    bubble.className = 'ai-bubble ai-bubble-assistant ai-thinking';
    bubble.innerHTML = `
      <div class="ai-bubble-label">AI</div>
      <div class="ai-bubble-content">
        <span class="ai-dot"></span><span class="ai-dot"></span><span class="ai-dot"></span>
      </div>
    `;
    msgs.appendChild(bubble);
    msgs.scrollTop = msgs.scrollHeight;
  }

  function removeThinking() {
    const b = el('ai-thinking-bubble');
    if (b) b.remove();
  }

  function formatMessage(text) {
    // Minimal markdown: code blocks, inline code, bold, newlines
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/```([\s\S]*?)```/g, '<pre class="ai-code">$1</pre>')
      .replace(/`([^`]+)`/g, '<code class="ai-inline-code">$1</code>')
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      .replace(/\n/g, '<br>');
  }

  // ───────────── Send / receive ─────────────

  async function sendMessage(text) {
    text = text.trim();
    if (!text || isThinking) return;

    // Expand slash commands to natural language hints
    text = expandSlashCommand(text);

    appendMessage('user', text);
    el('ai-input').value = '';
    autoResize(el('ai-input'));

    isThinking = true;
    el('ai-send-btn').disabled = true;
    showThinking();

    try {
      const res = await fetch(CHAT_URL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: text, history }),
      });

      const data = await res.json();
      removeThinking();

      if (!res.ok) {
        const errMsg = data.error || 'Unknown error';
        appendMessage('assistant', '⚠️ ' + errMsg);
        return;
      }

      const reply = data.reply || '(no response)';
      appendMessage('assistant', reply);

      // Update client-side history for next turn
      history.push({ role: 'user', content: text });
      history.push({ role: 'assistant', content: reply });
      saveHistory();

      // Keep history bounded at 20 turns to avoid huge payloads
      if (history.length > 40) history = history.slice(-40);

    } catch (err) {
      removeThinking();
      appendMessage('assistant', '⚠️ Network error: ' + err.message);
    } finally {
      isThinking = false;
      el('ai-send-btn').disabled = false;
    }
  }

  function expandSlashCommand(text) {
    const { feature, scenario } = getURLContext();
    const f  = feature  ? ` for feature "${feature}"`                                          : '';
    const fs = feature  ? ` for feature "${feature}"${scenario ? `, scenario "${scenario}"` : ''}` : '';

    const map = {
      '/list-features':  'List all features',
      '/list-scenarios': `List all scenarios${f}`,
      '/list-apis':      `List all APIs${fs}`,
      '/curl':           `Give me the cURL command for mock API${fs}`,
      '/search':         `Search mock APIs${fs}`,
      '/activate':       `Activate a scenario${f}`,
      '/create-api':     `Help me create a new mock API${fs}`,
      '/delete':         `Help me delete something${fs}`,
      '/help':           'What can you help me with?',
    };
    for (const [cmd, expansion] of Object.entries(map)) {
      if (text === cmd) return expansion;
      if (text.startsWith(cmd + ' ')) return expansion + ': ' + text.slice(cmd.length + 1);
    }
    return text;
  }

  // ───────────── Proto panel ─────────────

  function getURLContext() {
    const params = new URLSearchParams(window.location.search);
    return {
      feature:  params.get('feature')  || '',
      scenario: params.get('scenario') || '',
    };
  }

  function showProtoPanel() {
    el('ai-mock-panel').style.display = 'none';
    el('ai-proto-panel').style.display = 'flex';
    el('ai-proto-hint').textContent = '';
    const ctx = getURLContext();
    if (ctx.feature  && !el('ai-proto-feature').value)  el('ai-proto-feature').value  = ctx.feature;
    if (ctx.scenario && !el('ai-proto-scenario').value) el('ai-proto-scenario').value = ctx.scenario;
    el('ai-proto-feature').focus();
  }

  function hideProtoPanel() {
    el('ai-proto-panel').style.display = 'none';
    el('ai-proto-content').value = '';
    el('ai-proto-feature').value = '';
    el('ai-proto-scenario').value = '';
    el('ai-proto-hint').textContent = '';
  }

  // ── Proto parsing helpers (reuse app.js globals when available) ──

  function protoExtractServices(cleaned) {
    const services = [];
    const svcRe = /\bservice\s+(\w+)\s*\{/g;
    let m;
    while ((m = svcRe.exec(cleaned)) !== null) {
      const name = m[1];
      let depth = 1, i = m.index + m[0].length;
      while (i < cleaned.length && depth > 0) {
        if (cleaned[i] === '{') depth++;
        else if (cleaned[i] === '}') depth--;
        i++;
      }
      const body = cleaned.substring(m.index + m[0].length, i - 1);
      const rpcs = [];
      const rpcRe = /\brpc\s+(\w+)\s*\(\s*(?:stream\s+)?(\w+)\s*\)\s*returns\s*\(\s*(?:stream\s+)?(\w+)\s*\)/g;
      let r;
      while ((r = rpcRe.exec(body)) !== null) {
        rpcs.push({ name: r[1], requestType: r[2], responseType: r[3] });
      }
      if (rpcs.length) services.push({ name, rpcs });
    }
    return services;
  }

  function protoGetTemplate(messageType) {
    // Reuse app.js globals if available
    if (typeof generateTemplate === 'function' &&
        typeof parsedMessages !== 'undefined' &&
        parsedMessages[messageType]) {
      try { return generateTemplate(parsedMessages[messageType]); } catch (_) {}
    }
    return {};
  }

  function parseProtoForPanel(protoText) {
    const cleaned = protoText
      .replace(/\/\/.*$/gm, '')
      .replace(/\/\*[\s\S]*?\*\//g, '');

    // Seed app.js message/enum parsers if available
    if (typeof extractEnums === 'function' && typeof extractMessages === 'function') {
      if (typeof parsedEnums !== 'undefined')   Object.keys(parsedEnums).forEach(k => delete parsedEnums[k]);
      if (typeof parsedMessages !== 'undefined') Object.keys(parsedMessages).forEach(k => delete parsedMessages[k]);
      extractEnums(cleaned);
      extractMessages(cleaned);
    }

    return protoExtractServices(cleaned);
  }

  function buildProtoPrompt(feature, scenario, protoText, services) {
    const lines = [
      `I have a proto file. Please create mock APIs for every RPC in feature "${feature}", scenario "${scenario}".`,
      '',
      'Rules:',
      '- Use create_mock_api for EACH RPC.',
      '- HTTP method: POST (gRPC-HTTP uses POST for all RPCs).',
      '- Path: /{ServiceName}/{RpcName}',
      '- API name: {ServiceName}_{RpcName}',
      '- Set the output JSON to the response template below (fill with realistic fake values).',
      '',
    ];

    if (services.length > 0) {
      lines.push('Parsed services and response templates:');
      for (const svc of services) {
        lines.push(`\nService: ${svc.name}`);
        for (const rpc of svc.rpcs) {
          const tpl = protoGetTemplate(rpc.responseType);
          const tplStr = Object.keys(tpl).length
            ? JSON.stringify(tpl, null, 2)
            : `{ /* ${rpc.responseType} fields */ }`;
          lines.push(`  RPC: ${rpc.name}`);
          lines.push(`    request: ${rpc.requestType}  response: ${rpc.responseType}`);
          lines.push(`    path: /${svc.name}/${rpc.name}`);
          lines.push(`    response template:\n${tplStr}`);
        }
      }
      lines.push('');
    }

    lines.push('Proto source:');
    lines.push('```proto');
    lines.push(protoText.trim());
    lines.push('```');
    lines.push('');
    lines.push('Create a mock API for every RPC above. Fill response fields with realistic fake data (e.g. id=1, name="John Doe", status="active", etc.).');

    return lines.join('\n');
  }

  function submitProto() {
    const ctx      = getURLContext();
    const feature  = (el('ai-proto-feature').value  || '').trim() || ctx.feature;
    const scenario = (el('ai-proto-scenario').value || '').trim() || ctx.scenario;
    const proto    = (el('ai-proto-content').value  || '').trim();

    if (!feature)  { el('ai-proto-hint').textContent = '⚠ Feature name is required.';  return; }
    if (!scenario) { el('ai-proto-hint').textContent = '⚠ Scenario name is required.'; return; }
    if (!proto)    { el('ai-proto-hint').textContent = '⚠ Please paste a .proto file.'; return; }

    const services = parseProtoForPanel(proto);
    const rpcCount = services.reduce((n, s) => n + s.rpcs.length, 0);

    if (rpcCount === 0) {
      el('ai-proto-hint').textContent = '⚠ No service/rpc definitions found. Make sure your proto has a "service { rpc … }" block.';
      return;
    }

    el('ai-proto-hint').textContent = `Found ${rpcCount} RPC(s) across ${services.length} service(s). Sending to AI…`;

    const prompt = buildProtoPrompt(feature, scenario, proto, services);
    hideProtoPanel();
    sendMessage(prompt);
  }

  // ───────────── Mock panel ─────────────

  function showMockPanel() {
    el('ai-proto-panel').style.display = 'none';
    el('ai-mock-panel').style.display = 'flex';
    el('ai-mock-hint').textContent = '';
    const ctx = getURLContext();
    if (ctx.feature  && !el('ai-mock-feature').value)  el('ai-mock-feature').value  = ctx.feature;
    if (ctx.scenario && !el('ai-mock-scenario').value) el('ai-mock-scenario').value = ctx.scenario;
    el('ai-mock-feature').focus();
  }

  function hideMockPanel() {
    el('ai-mock-panel').style.display = 'none';
    el('ai-mock-content').value = '';
    el('ai-mock-feature').value = '';
    el('ai-mock-scenario').value = '';
    el('ai-mock-hint').textContent = '';
  }

  function buildMockPrompt(feature, scenario, contract) {
    const lines = [
      'You are a mock-API generator. From the API contract below, create mock APIs that cover every example case.',
      '',
      '## Instructions',
      '1. **One feature** per API (use the provided feature name, or derive it from the API path — e.g. `referrals` from `/api-mf/private/referrals`).',
      '2. Check whether the feature already exists (`list_features`). Create it only if missing.',
      '3. **One scenario per logical case** (success variants, error cases). Name them descriptively:',
      '   - Success cases: `success_<variant>` (e.g. `success_jobs`, `success_pty`)',
      '   - Error cases: `error_<code_or_reason>` (e.g. `error_login_required`, `error_unauthorized`)',
      '   - If the contract shows multiple responses for the same request (a sequence), use the `responses` array with `from`/`to`/`status_code` entries instead of separate scenarios.',
      '4. **Per scenario** call `create_scenario` then `create_mock_api`:',
      '   - `request_body` = the example request JSON (omit if the contract has no request body).',
      '   - `response` = the example response JSON.',
      '   - `status_code` = HTTP status: 200 for success, 4xx/5xx for errors.',
      '   - `method` = from the contract (GET, POST, PUT, DELETE, etc.).',
      '   - `path` = exactly as in the contract.',
      '5. After creating everything, reply with a markdown summary table: scenario | path | method | status.',
      '',
    ];

    if (feature) {
      lines.push('## Requested feature name');
      lines.push('`' + feature + '`');
      lines.push('');
    }
    if (scenario) {
      lines.push('## Requested base scenario name');
      lines.push('`' + scenario + '` (use as prefix or as-is for the primary case)');
      lines.push('');
    }

    lines.push('## API Contract');
    lines.push('```');
    lines.push(contract.trim());
    lines.push('```');
    lines.push('');
    lines.push('Now create the feature, scenarios, and mock APIs. Start by checking if the feature already exists.');

    return lines.join('\n');
  }

  function submitMock() {
    const ctx      = getURLContext();
    const feature  = (el('ai-mock-feature').value  || '').trim() || ctx.feature;
    const scenario = (el('ai-mock-scenario').value || '').trim() || ctx.scenario;
    const contract = (el('ai-mock-content').value  || '').trim();

    if (!contract) {
      el('ai-mock-hint').textContent = '⚠ Please paste an API contract.';
      return;
    }

    el('ai-mock-hint').textContent = 'Sending to AI…';

    const prompt = buildMockPrompt(feature, scenario, contract);
    hideMockPanel();
    sendMessage(prompt);
  }

  // ───────────── Clear conversation ─────────────

  async function clearConversation() {
    // Clear local history
    history = [];
    clearHistory();

    // Call /clear endpoint on server
    try {
      await fetch(CHAT_URL + '/clear', { method: 'POST' });
    } catch (err) {
      console.warn('Failed to notify server of conversation clear:', err);
    }

    const msgs = el('ai-messages');
    // Remove all bubbles but keep the welcome panel
    msgs.querySelectorAll('.ai-bubble').forEach(b => b.remove());
    const welcome = el('ai-welcome');
    if (welcome) welcome.style.display = '';
  }

  // ───────────── Input auto-resize ─────────────

  function autoResize(textarea) {
    textarea.style.height = 'auto';
    textarea.style.height = Math.min(textarea.scrollHeight, 140) + 'px';
  }

  // ───────────── Keyboard shortcuts ─────────────

  function bindInputEvents() {
    const input = el('ai-input');
    input.addEventListener('input', () => autoResize(input));
    function handlePanelShortcuts(val) {
      if (val === '/proto') { showProtoPanel(); return true; }
      if (val === '/mock')  { showMockPanel();  return true; }
      return false;
    }

    input.addEventListener('keydown', e => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        const val = input.value.trim();
        if (handlePanelShortcuts(val)) {
          input.value = '';
          autoResize(input);
        } else {
          sendMessage(input.value);
        }
      }
    });
    el('ai-send-btn').addEventListener('click', () => {
      const val = el('ai-input').value.trim();
      if (handlePanelShortcuts(val)) {
        el('ai-input').value = '';
        autoResize(el('ai-input'));
      } else {
        sendMessage(el('ai-input').value);
      }
    });
    el('ai-clear-btn').addEventListener('click', clearConversation);
    el('ai-close-btn').addEventListener('click', closeSidebar);
    el('ai-toggle-btn').addEventListener('click', toggleSidebar);
    el('ai-proto-submit').addEventListener('click', submitProto);
    el('ai-proto-cancel').addEventListener('click', hideProtoPanel);
    el('ai-mock-submit').addEventListener('click', submitMock);
    el('ai-mock-cancel').addEventListener('click', hideMockPanel);
    bindResizeHandle();
  }

  // ───────────── Sidebar resize ─────────────

  function bindResizeHandle() {
    const resizer = el('ai-sidebar-resizer');
    const sidebar = el('ai-sidebar');
    let isResizing = false;
    let startX = 0;
    let startWidth = 0;

    resizer.addEventListener('mousedown', (e) => {
      isResizing = true;
      startX = e.clientX;
      startWidth = sidebar.offsetWidth;
      document.addEventListener('mousemove', handleResize);
      document.addEventListener('mouseup', stopResize);
      e.preventDefault();
    });

    function handleResize(e) {
      if (!isResizing) return;
      const delta = startX - e.clientX; // drag left = positive delta
      const newWidth = Math.max(280, Math.min(800, startWidth + delta)); // min 280px, max 800px
      sidebar.style.width = newWidth + 'px';
    }

    function stopResize() {
      if (isResizing) {
        isResizing = false;
        // Save width to localStorage
        localStorage.setItem('ai-sidebar-width', sidebar.offsetWidth);
        document.removeEventListener('mousemove', handleResize);
        document.removeEventListener('mouseup', stopResize);
      }
    }
  }

  // ───────────── Init ─────────────

  function init() {
    createToggleButton();
    createSidebar();
    populateSuggestions();
    populateCommands();
    bindInputEvents();

    // Restore chat history from localStorage
    if (loadHistory() && history.length > 0) {
      // Restore messages to UI
      history.forEach(msg => {
        appendMessage(msg.role, msg.content);
      });
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
