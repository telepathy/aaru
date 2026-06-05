// ===== API Layer =====
const API = '/api';
export let currentUser = null;

async function api(url, opts = {}) {
  const res = await fetch(API + url, {
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', ...opts.headers },
    ...opts,
  });
  if (res.status === 401) { window.location.href = '/auth/login'; return null; }
  if (!res.ok) { const e = await res.json().catch(()=>({error:res.statusText})); throw new Error(e.error||res.statusText); }
  return res.json();
}

async function checkAuth() {
  try {
    const data = await api('/current-user');
    if (!data) return;
    currentUser = data;
    document.getElementById('user-name').textContent = data.username;
    document.getElementById('user-avatar').textContent = (data.username||'A')[0].toUpperCase();
    const roles = (data.roles||[]).map(r=>r.name);
    document.getElementById('user-role').textContent = roles.join(', ') || '无角色';
    if (roles.includes('admin')) {
      document.getElementById('nav-admin').style.display = '';
    }
  } catch(e) { window.location.href = '/auth/login'; }
}

async function logout() {
  await fetch(API + '/logout', { method:'POST', credentials:'same-origin' });
  window.location.href = '/auth/login';
}

export { API, api, checkAuth, logout };
