export function chatPanel(sessionToken: string): string {
  return `<style>
  #c2-coach {
    position: fixed; top: 0; right: 0; bottom: 0; width: 400px;
    background: #161b22; border-left: 1px solid #30363d;
    display: none; flex-direction: column; z-index: 9999;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  }
  #c2-coach.c2-open { display: flex; }
  @media (min-width: 900px) { body.c2-shift { margin-right: 400px; } }
  @media (max-width: 480px) { #c2-coach { width: 100%; } body.c2-shift { margin-right: 0; } }
  #c2-head {
    display: flex; align-items: center; justify-content: space-between;
    padding: 12px 14px; border-bottom: 1px solid #30363d; flex-shrink: 0;
  }
  #c2-head .c2-title { color: #f0f6fc; font-weight: 600; font-size: 14px; }
  #c2-head .c2-title::before { content: "\\2022"; color: #3fb950; margin-right: 6px; }
  .c2-head-actions { display: flex; gap: 6px; }
  .c2-head-actions button {
    background: #21262d; color: #8b949e; border: 1px solid #30363d;
    border-radius: 6px; padding: 4px 10px; font-size: 12px; cursor: pointer;
  }
  .c2-head-actions button:hover { color: #c9d1d9; border-color: #8b949e; }
  #c2-log { flex: 1; overflow-y: auto; padding: 14px; display: flex; flex-direction: column; gap: 10px; }
  .c2-msg {
    font-size: 13px; line-height: 1.5; padding: 9px 12px; border-radius: 10px;
    max-width: 92%; white-space: pre-wrap; word-wrap: break-word;
  }
  .c2-coach-msg { background: #21262d; color: #c9d1d9; align-self: flex-start; border-top-left-radius: 2px; }
  .c2-user-msg { background: #1f6feb; color: #fff; align-self: flex-end; border-top-right-radius: 2px; }
  .c2-msg code { background: #0d1117; padding: 1px 5px; border-radius: 4px; font-size: 12px; }
  .c2-event { font-size: 11px; color: #6e7681; font-style: italic; align-self: flex-start; padding-left: 2px; }
  #c2-chips { display: flex; flex-wrap: wrap; gap: 6px; padding: 0 14px 10px; flex-shrink: 0; }
  #c2-chips.c2-hidden { display: none; }
  .c2-chip {
    background: #0d1117; color: #58a6ff; border: 1px solid #30363d;
    border-radius: 14px; padding: 5px 11px; font-size: 12px; cursor: pointer;
  }
  .c2-chip:hover { border-color: #58a6ff; }
  #c2-form { display: flex; gap: 8px; padding: 12px 14px; border-top: 1px solid #30363d; flex-shrink: 0; }
  #c2-input {
    flex: 1; background: #0d1117; color: #c9d1d9; border: 1px solid #30363d;
    border-radius: 8px; padding: 9px 12px; font-size: 13px; outline: none;
  }
  #c2-input:focus { border-color: #58a6ff; }
  #c2-send { background: #238636; color: #fff; border: none; border-radius: 8px; padding: 9px 16px; font-size: 13px; cursor: pointer; }
  #c2-send:disabled { opacity: 0.5; cursor: default; }
  #c2-fab {
    position: fixed; bottom: 20px; right: 20px; z-index: 9999;
    background: #238636; color: #fff; border: none; border-radius: 20px;
    padding: 10px 18px; font-size: 14px; font-weight: 600; cursor: pointer;
    box-shadow: 0 2px 10px rgba(0,0,0,0.4);
  }
  #c2-fab.c2-hidden { display: none; }
</style>
<div id="c2-coach" class="c2-open">
  <div id="c2-head">
    <span class="c2-title">Coach</span>
    <div class="c2-head-actions">
      <button id="c2-notes-btn" title="Show saved notes">Notes</button>
      <button id="c2-toggle" title="Collapse">Hide</button>
    </div>
  </div>
  <div id="c2-log">
    <div class="c2-msg c2-coach-msg">Ask me about this report &mdash; goal pace, training load, a specific piece, recovery. Type <code>/note ...</code> to save something to memory.</div>
  </div>
  <div id="c2-chips">
    <button class="c2-chip">Am I on pace for my goal?</button>
    <button class="c2-chip">Is my volume ramping safely?</button>
    <button class="c2-chip">How is my intensity distribution?</button>
    <button class="c2-chip">Review my most recent hard piece</button>
  </div>
  <form id="c2-form">
    <input id="c2-input" autocomplete="off" placeholder="Ask the coach…" />
    <button id="c2-send" type="submit">Send</button>
  </form>
</div>
<button id="c2-fab" class="c2-hidden" title="Open coach">Coach</button>
<script>
(function(){
  var TOKEN = ${JSON.stringify(sessionToken)};
  var panel = document.getElementById('c2-coach');
  var log = document.getElementById('c2-log');
  var form = document.getElementById('c2-form');
  var input = document.getElementById('c2-input');
  var send = document.getElementById('c2-send');
  var chips = document.getElementById('c2-chips');
  var fab = document.getElementById('c2-fab');
  var toggle = document.getElementById('c2-toggle');
  var notesBtn = document.getElementById('c2-notes-btn');
  document.body.classList.add('c2-shift');

  function scroll(){ log.scrollTop = log.scrollHeight; }
  function add(role, text){
    var el = document.createElement('div');
    el.className = 'c2-msg ' + (role === 'user' ? 'c2-user-msg' : 'c2-coach-msg');
    el.textContent = text;
    log.appendChild(el); scroll(); return el;
  }
  function addEvent(t){
    var el = document.createElement('div');
    el.className = 'c2-event';
    el.textContent = '\\u00b7 ' + t;
    log.appendChild(el); scroll();
  }
  function setBusy(b){ input.disabled = b; send.disabled = b; if(!b) input.focus(); }
  function hideChips(){ chips.classList.add('c2-hidden'); }

  async function postJSON(path, body){
    var r = await fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-C2-Token': TOKEN }, body: JSON.stringify(body) });
    return r.json();
  }

  async function run(message){
    hideChips();
    if (message.indexOf('/note ') === 0){
      var note = message.slice(6).trim();
      if (!note) return;
      add('user', message);
      try { var d = await postJSON('/api/notes', { note: note }); addEvent(d.saved ? 'saved to coach memory' : (d.error || 'could not save')); }
      catch(e){ addEvent('error: ' + e.message); }
      return;
    }
    add('user', message);
    setBusy(true);
    var thinking = add('coach', 'thinking\\u2026');
    try {
      var d = await postJSON('/api/chat', { message: message });
      thinking.remove();
      (d.events || []).forEach(addEvent);
      add('coach', d.reply || d.error || '(no response)');
    } catch(e){ thinking.remove(); add('coach', 'Error: ' + e.message); }
    setBusy(false);
  }

  async function showNotes(){
    try {
      var r = await fetch('/api/notes'); var d = await r.json();
      var notes = d.notes || [];
      if (!notes.length){ addEvent('no saved notes yet'); return; }
      var lines = notes.map(function(n){ return '\\u2022 [' + n.date.slice(0,10) + '] ' + n.note; });
      add('coach', 'Saved notes:\\n' + lines.join('\\n'));
    } catch(e){ addEvent('error: ' + e.message); }
  }

  form.addEventListener('submit', function(e){
    e.preventDefault();
    var v = input.value.trim();
    if (!v) return;
    input.value = '';
    run(v);
  });
  var chipButtons = chips.querySelectorAll('.c2-chip');
  for (var i = 0; i < chipButtons.length; i++){
    chipButtons[i].addEventListener('click', function(){ run(this.textContent); });
  }
  notesBtn.addEventListener('click', showNotes);
  toggle.addEventListener('click', function(){ panel.classList.remove('c2-open'); fab.classList.remove('c2-hidden'); document.body.classList.remove('c2-shift'); });
  fab.addEventListener('click', function(){ panel.classList.add('c2-open'); fab.classList.add('c2-hidden'); document.body.classList.add('c2-shift'); input.focus(); });
})();
</script>`;
}
