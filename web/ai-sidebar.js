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

  const CHAT_HISTORY_COOKIE = 'ai_chat_history';
  const COOKIE_MAX_AGE = 604800; // 7 days in seconds

  // ───────────── Cookie helpers ─────────────

  function saveHistoryToCookie() {
    try {
      const jsonStr = JSON.stringify(history);
      document.cookie = `${CHAT_HISTORY_COOKIE}=${encodeURIComponent(jsonStr)}; max-age=${COOKIE_MAX_AGE}; path=/`;
    } catch (err) {
      console.warn('Failed to save chat history to cookie:', err);
    }
  }

  function loadHistoryFromCookie() {
    try {
      const name = CHAT_HISTORY_COOKIE + '=';
      const decodedCookie = decodeURIComponent(document.cookie);
      const cookieArray = decodedCookie.split(';');
      for (let i = 0; i < cookieArray.length; i++) {
        let cookie = cookieArray[i].trim();
        if (cookie.indexOf(name) === 0) {
          const jsonStr = cookie.substring(name.length);
          const parsed = JSON.parse(decodeURIComponent(jsonStr));
          if (Array.isArray(parsed)) {
            history = parsed;
            return true;
          }
        }
      }
    } catch (err) {
      console.warn('Failed to load chat history from cookie:', err);
    }
    return false;
  }

  function clearHistoryCookie() {
    document.cookie = `${CHAT_HISTORY_COOKIE}=; max-age=0; path=/`;
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
        el('ai-input').value = cmd + ' ';
        el('ai-input').focus();
        autoResize(el('ai-input'));
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
      saveHistoryToCookie();

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
    const map = {
      '/list-features': 'List all features',
      '/list-scenarios': 'List all scenarios',
      '/list-apis': 'List all APIs',
      '/search': 'Search mock APIs',
      '/activate': 'Activate a scenario',
      '/create-api': 'Help me create a new mock API',
      '/delete': 'Help me delete something',
      '/help': 'What can you help me with?',
    };
    for (const [cmd, expansion] of Object.entries(map)) {
      if (text === cmd) return expansion;
      if (text.startsWith(cmd + ' ')) return expansion + ': ' + text.slice(cmd.length + 1);
    }
    return text;
  }

  // ───────────── Clear conversation ─────────────

  async function clearConversation() {
    // Clear local history
    history = [];
    clearHistoryCookie();

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
    input.addEventListener('keydown', e => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage(input.value);
      }
    });
    el('ai-send-btn').addEventListener('click', () => sendMessage(input.value));
    el('ai-clear-btn').addEventListener('click', clearConversation);
    el('ai-close-btn').addEventListener('click', closeSidebar);
    el('ai-toggle-btn').addEventListener('click', toggleSidebar);
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

    // Load chat history from cookie
    if (loadHistoryFromCookie() && history.length > 0) {
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
