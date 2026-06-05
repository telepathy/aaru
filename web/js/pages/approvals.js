import { api } from '../api.js';
import { showLoading, escapeHtml, fmtTime } from '../utils.js';

// ===== Approvals =====
async function renderApprovals(body) {
  showLoading(body);
  try {
    const data = await api('/approvals/pending');
    const stages = data.stages||[];
    if (stages.length===0) { body.innerHTML = '<div class="empty-state"><svg width="40" height="40" viewBox="0 0 40 40" fill="none"><path d="M12 22l6 6 10-10" stroke="#d1d5db" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><circle cx="20" cy="20" r="14" stroke="#d1d5db" stroke-width="2" fill="none"/></svg><p>暂无待审批的发布</p></div>'; return; }
    body.innerHTML = `<div class="card"><div class="card-header"><div class="card-title">待审批 (${stages.length})</div></div>
      <table class="data-table"><thead><tr><th>发布单</th><th>环境</th><th>部署单元</th><th>版本</th><th>申请时间</th><th>操作</th></tr></thead><tbody>${stages.map(s=>`<tr>
        <td><a href="#" class="text-link" onclick="loadPage('release-detail',${s.release_id});return false">#${s.release_id}</a></td>
        <td>${escapeHtml(s.env_name||s.env_code)}</td>
        <td>${escapeHtml(s.release?.deploy_unit_code||'-')}</td>
        <td><code style="background:#f4f4f5;padding:2px 6px;border-radius:4px;font-size:12px">${escapeHtml(s.release?.version||'-')}</code></td>
        <td>${fmtTime(s.created_at)}</td>
        <td class="action-group">
          <button class="btn btn-sm btn-success" onclick="approveStage(${s.id})">通过</button>
          <button class="btn btn-sm btn-danger" onclick="rejectStage(${s.id})">驳回</button>
        </td>
      </tr>`).join('')}</tbody></table></div>`;
  } catch(e) { body.innerHTML = '<div class="empty-state"><p>加载失败: '+escapeHtml(e.message)+'</p></div>'; }
}

export { renderApprovals };
