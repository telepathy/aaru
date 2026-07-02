import { currentUser, api, checkAuth, logout } from './api.js';
import { toast, setPage } from './utils.js';
import { showDiffModal } from './modal.js';
import { renderApprovals } from './pages/approvals.js';
import { renderAdmin } from './pages/admin.js';
import { renderDeployUnits, duSnapshots } from './pages/deploy-units.js';
import { renderReleaseList, renderReleaseDetail } from './pages/releases.js';
import { renderCreateRelease, crSnapshots } from './pages/create-release.js';
import { renderBatchRelease } from './pages/batch-release.js';
import { loadPageBlueprintList, envCache, roleCache } from './pages/blueprints.js';

// ===== SPA Router =====

// Map page+param to hash fragment (without leading #)
function pageToHash(page, param) {
  switch(page) {
    case 'releases': return '#/releases';
    case 'release-detail': return '#/releases/' + param;
    case 'create-release': return '#/releases/create';
    case 'batch-release': return '#/releases/batch';
    case 'deploy-units': return '#/deploy-units';
    case 'approvals': return '#/approvals';
    case 'blueprints': return '#/blueprints';
    case 'admin': return '#/admin';
    default: return '#/releases';
  }
}

// Parse hash fragment to {page, param}
function hashToPage(hash) {
  const path = (hash || '').replace(/^#\/?/, '');
  if (!path || path === 'releases') return { page: 'releases', param: null };

  const detailMatch = path.match(/^releases\/(\d+)$/);
  if (detailMatch) return { page: 'release-detail', param: parseInt(detailMatch[1], 10) };

  if (path === 'releases/create') return { page: 'create-release', param: null };
  if (path === 'releases/batch')  return { page: 'batch-release', param: null };
  if (path === 'deploy-units')    return { page: 'deploy-units', param: null };
  if (path === 'approvals')       return { page: 'approvals', param: null };
  if (path === 'blueprints')      return { page: 'blueprints', param: null };
  if (path === 'admin')           return { page: 'admin', param: null };

  return { page: 'releases', param: null }; // unrecognized → fallback
}

// Render a page (the actual switch-case)
function renderPage(page, param) {
  const body = document.getElementById('content-body');
  const actions = document.getElementById('header-actions');
  actions.innerHTML = '';
  // 重置 content-body 样式（create-release 页面会修改）
  body.style.display = '';
  body.style.flexDirection = '';
  body.style.overflow = '';

  switch(page) {
    case 'releases': setPage('releases','发布管理','管理应用发布流水线'); renderReleaseList(body,actions); break;
    case 'create-release': setPage('releases','新建发布','创建新的发布单'); renderCreateRelease(body,actions); break;
    case 'batch-release': setPage('releases','批量发布','多个部署单元统一升级版本'); renderBatchRelease(body,actions); break;
    case 'release-detail': setPage('releases','发布详情','查看发布流水线进度'); renderReleaseDetail(body,param); break;
    case 'deploy-units': setPage('deploy-units','部署单元','浏览各环境的部署单元'); renderDeployUnits(body); break;
    case 'approvals': setPage('approvals','审批中心','处理待审批的发布'); renderApprovals(body); break;
    case 'blueprints': loadPageBlueprintList(body); break;
    case 'admin':
      if (!currentUser?.roles?.some(r=>r.name==='admin')) { toast('无权限访问','error'); loadPage('releases'); return; }
      setPage('admin','权限管理','管理用户角色和权限'); renderAdmin(body); break;
    default: loadPage('releases');
  }
}

// Public entry point: called from inline onclick handlers
// Sets the hash; rendering happens via the hashchange event
function loadPage(page, param) {
  const newHash = pageToHash(page, param);
  if (window.location.hash !== newHash) {
    window.location.hash = newHash; // synchronously fires hashchange → loadPageFromHash → renderPage
    return;
  }
  renderPage(page, param);
}

// Called on hashchange and on initial page load
function loadPageFromHash() {
  const { page, param } = hashToPage(window.location.hash);
  renderPage(page, param);
}

// ===== Init =====
document.addEventListener('DOMContentLoaded', async () => {
  await checkAuth();

  // Preload envs for dropdowns; roles only for admin
  try { envCache = (await api('/environments')).envs||[]; } catch(e) {}
  if (currentUser?.roles?.some(r=>r.name==='admin')) {
    try { roleCache = (await api('/admin/roles')).roles||[]; } catch(e) {}
  }

  loadPageFromHash();

  // 监听 hash 变化，支持浏览器前进/后退
  // 忽略裸 #（来自 <a href="#"> 点击），只响应 #/ 开头的业务路由
  window.addEventListener('hashchange', () => {
    if (!window.location.hash || window.location.hash === '#') return;
    loadPageFromHash();
  });

  // 全局事件委托：点击 .diff-field-link 打开 diff 模态框
  document.addEventListener('click', e => {
    const el = e.target.closest('.diff-field-link');
    if (!el) return;
    const field = el.dataset.field;
    // 优先使用 duSnapshots（部署单元页面），其次 crSnapshots（创建发布向导）
    const snaps = (duSnapshots.length > 0) ? duSnapshots :
                  (crSnapshots.length > 0) ? crSnapshots : null;
    if (field && snaps) showDiffModal(field, snaps);
  });
});

// ===== Expose functions for inline onclick handlers =====
window.loadPage = loadPage;
window.logout = logout;
