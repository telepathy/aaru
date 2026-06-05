import { api } from '../api.js';
import { showLoading, escapeHtml, crAssignValueColors, summarizeValue } from '../utils.js';
import { showDiffModal } from '../modal.js';

// ===== Deploy Units Browse =====
let duSelectedCode = null;
let duSiloOptions = [];   // derived from DevOps API
let duSystemOptions = []; // derived from DevOps API
let duCache = {};         // code -> DevOpsDUItem
let duAll = [];           // all DUs from API
export let duSnapshots = [];     // last compared snapshots

async function renderDeployUnits(body) {
  showLoading(body);
  try {
    // Fetch all DUs from DevOps API to derive filter options (no DMDB)
    const allData = await api('/du-list');
    duAll = allData.deploy_units||[];

    // Extract unique silos and systems from DU list
    const siloSet = new Set();
    const sysSet = new Set();
    duAll.forEach(d => {
      if (d.silo) siloSet.add(d.silo);
      if (d.system) sysSet.add(d.system);
    });
    duSiloOptions = [...siloSet].sort();
    duSystemOptions = [...sysSet].sort();

    body.innerHTML = `<div id="du-grid" style="display:grid;grid-template-columns:1fr 2fr;gap:20px;height:calc(100vh - 140px)">
      <div style="display:flex;flex-direction:column;min-height:0;min-width:0">
        <div class="card" style="flex:1;display:flex;flex-direction:column;min-height:0;min-width:0">
          <div class="card-header"><div class="card-title">部署单元列表</div></div>
          <div class="filter-bar" style="padding:12px 20px">
            <select class="form-control" id="du-silo" onchange="loadDUList()"><option value="">全部竖井</option>${duSiloOptions.map(c=>`<option value="${escapeHtml(c)}">${escapeHtml(c)}</option>`).join('')}</select>
            <select class="form-control" id="du-system" onchange="loadDUList()"><option value="">全部系统</option>${duSystemOptions.map(c=>`<option value="${escapeHtml(c)}">${escapeHtml(c)}</option>`).join('')}</select>
          </div>
          <div id="du-list" style="flex:1;overflow-y:auto;padding:0 20px 20px"><div class="loading-state"><div class="spinner"></div></div></div>
        </div>
      </div>
      <div style="min-height:0;min-width:0">
        <div class="card" style="height:100%;display:flex;flex-direction:column;min-width:0;overflow:hidden">
          <div class="card-header"><div class="card-title" id="du-detail-title">部署单元详情</div></div>
          <div id="du-detail" style="flex:1;overflow-y:auto;padding:20px;min-width:0"><div class="empty-state"><p>点击左侧部署单元查看详情</p></div></div>
        </div>
      </div>
    </div>`;
    renderDUList(duAll);
  } catch(e) { body.innerHTML = '<div class="empty-state"><p>加载失败: '+escapeHtml(e.message)+'</p></div>'; }
}

function renderDUList(dus) {
  const container = document.getElementById('du-list');
  if (!container) return;
  // Update cache
  duCache = {};
  (dus||[]).forEach(d => { duCache[d.code] = d; });
  if (!dus||dus.length===0) { container.innerHTML = '<div class="empty-state"><p>无匹配的部署单元</p></div>'; return; }
  container.innerHTML = dus.map(d=>{
    const sel = d.code===duSelectedCode?'selected':'';
    return `<div class="du-list-item ${sel}" onclick="loadDUDetail('${escapeHtml(d.code)}')" data-code="${escapeHtml(d.code)}">
      <div class="du-item-code">${escapeHtml(d.code)}</div>
      <div class="du-item-meta">Silo: ${escapeHtml(d.silo||'-')} / System: ${escapeHtml(d.system||'-')}</div>
    </div>`;
  }).join('');
}

function loadDUList() {
  const silo = document.getElementById('du-silo')?.value||'';
  const system = document.getElementById('du-system')?.value||'';
  let dus = duAll;
  if (silo) dus = dus.filter(d => d.silo === silo);
  if (system) dus = dus.filter(d => d.system === system);
  renderDUList(dus);
}

async function loadDUDetail(code) {
  duSelectedCode = code;
  const grid = document.getElementById('du-grid');
  if (grid) grid.style.gridTemplateColumns = '1fr 2fr';
  const title = document.getElementById('du-detail-title');
  const detail = document.getElementById('du-detail');
  if (title) title.textContent = code;
  if (detail) showLoading(detail);

  // Highlight selected item
  document.querySelectorAll('.du-list-item').forEach(el=>{
    el.classList.toggle('selected', el.dataset.code===code);
  });

  const d = duCache[code];
  if (!d) {
    if (detail) detail.innerHTML = '<div class="empty-state"><p>未找到该部署单元</p></div>';
    return;
  }

  // Build basic info section
  let html = `<div class="du-detail-card">
    <div class="du-detail-field"><label>部署单元编码</label><span><code>${escapeHtml(d.code)}</code></span></div>
    <div class="du-detail-field"><label>所属竖井 (Silo)</label><span>${escapeHtml(d.silo||'-')}</span></div>
    <div class="du-detail-field"><label>所属系统 (System)</label><span>${escapeHtml(d.system||'-')}</span></div>
    <div class="du-detail-field"><label>代码仓库</label><span style="word-break:break-all;font-size:12px">${escapeHtml(d.repo||'-')}</span></div>
  </div>`;

  // Fetch version comparison from DMDB
  try {
    const data = await api('/deploy-units/'+encodeURIComponent(code)+'/compare');
    duSnapshots = data.snapshots||[];
    if (duSnapshots.length > 0) {
      const f = s => k => (s.fields||{})[k]||'-';
      html += `<div style="margin-top:20px;display:flex;align-items:center;justify-content:space-between">
        <h4 style="font-size:13px;font-weight:600;padding-bottom:8px">各环境版本对比</h4>
        <button class="btn btn-sm btn-primary" onclick="showDUCompareDetail()">详细比对</button>
      </div>
        <table class="data-table du-compare-table">
          <thead><tr><th>环境</th><th>制品版本</th><th>节点数</th></tr></thead>
          <tbody>${duSnapshots.map(s=>`<tr>
            <td><strong>${escapeHtml(s.env_name||s.env)}</strong><br><span style="font-size:10px;color:var(--text-muted)">${escapeHtml(s.env)}</span></td>
            <td><code style="background:#f4f4f5;padding:2px 6px;border-radius:4px;font-size:12px">${escapeHtml(f(s)('ArtifactVersion'))}</code></td>
            <td>${escapeHtml(f(s)('NodeCount'))}</td>
          </tr>`).join('')}</tbody>
        </table></div>`;
    } else {
      html += `<div style="margin-top:20px"><div class="empty-state"><p>未在任何DMDB环境中找到此部署单元</p></div></div>`;
    }
  } catch(e) {
    html += `<div style="margin-top:20px"><div class="empty-state"><p>获取版本对比失败: ${escapeHtml(e.message)}</p></div></div>`;
  }

  if (detail) detail.innerHTML = html;
}

let duSelectedEnvs = []; // currently enabled env indices

function showDUCompareDetail() {
  const grid = document.getElementById('du-grid');
  if (grid) grid.style.gridTemplateColumns = '240px 1fr';
  const detail = document.getElementById('du-detail');
  const title = document.getElementById('du-detail-title');
  if (!detail) return;
  if (duSnapshots.length === 0) {
    detail.innerHTML = '<div class="empty-state"><p>无对比数据</p></div>';
    return;
  }

  const code = duSelectedCode || '';
  if (title) title.textContent = code + ' - 详细比对';

  // All envs selected by default
  duSelectedEnvs = duSnapshots.map((_, i) => i);

  // Build env selector bar + table container, then render table
  detail.innerHTML = `<div id="du-compare-toolbar" style="margin-bottom:8px;display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:8px">
    <div style="display:flex;align-items:center;gap:6px;flex-wrap:wrap;font-size:11px">
      <span style="color:var(--text-muted);margin-right:4px">环境:</span>
      ${duSnapshots.map((e,i)=>`<label class="du-env-chip" id="du-env-chip-${i}">
        <input type="checkbox" checked onchange="toggleCompareEnv(${i})">
        <span>${escapeHtml(e.env_name||e.env)}</span>
      </label>`).join('')}
    </div>
    <button class="btn btn-sm btn-secondary" onclick="loadDUDetail('${escapeHtml(code)}')">返回概览</button>
  </div>
  <div id="du-compare-table-wrap" style="overflow-x:auto;width:100%"></div>`;
  renderCompareTable();
}

function toggleCompareEnv(idx) {
  const chip = document.getElementById('du-env-chip-'+idx);
  const checked = chip.querySelector('input').checked;
  if (checked) {
    duSelectedEnvs.push(idx);
    duSelectedEnvs.sort((a,b)=>a-b);
  } else {
    duSelectedEnvs = duSelectedEnvs.filter(i=>i!==idx);
  }
  renderCompareTable();
}

function renderCompareTable() {
  const wrap = document.getElementById('du-compare-table-wrap');
  if (!wrap) return;
  const selected = duSelectedEnvs.map(i=>duSnapshots[i]);
  if (selected.length < 2) {
    wrap.innerHTML = '<div class="empty-state"><p>请至少选择2个环境进行比对</p></div>';
    return;
  }

  const skipKeys = new Set(['id','Env','classCode','biz_serial','SiloCode','System']);
  const allKeys = new Set();
  selected.forEach(s => Object.keys(s.fields||{}).forEach(k => { if (!skipKeys.has(k) && !k.includes('[')) allKeys.add(k); }));

  // 只保留有差异的字段
  const diffKeys = [...allKeys].filter(key => {
    const vals = selected.map(s=>String((s.fields||{})[key]||''));
    return new Set(vals).size > 1;
  }).sort();

  if (diffKeys.length === 0) {
    wrap.innerHTML = '<div class="empty-state"><p>所选环境配置完全一致，无差异</p></div>';
    return;
  }

  wrap.innerHTML = `<div style="margin-bottom:8px;font-size:12px;color:var(--text-muted)">${selected.length} 个环境 / ${diffKeys.length} 个差异项</div>
    <table class="data-table du-compare-table" style="min-width:600px;table-layout:auto">
      <thead><tr><th style="min-width:120px">配置项</th>${selected.map(e=>`<th>${escapeHtml(e.env_name||e.env)}<br><span style="font-weight:400;font-size:10px;color:var(--text-muted)">${escapeHtml(e.env)}</span></th>`).join('')}</tr></thead>
      <tbody>${diffKeys.map(key=>{
        const rawVals = selected.map(s=>String((s.fields||{})[key]||''));
        const colors = crAssignValueColors(rawVals);
        return `<tr>
          <td><strong class="diff-field-link" data-field="${escapeHtml(key)}" title="点击查看详细差异">${escapeHtml(key)}</strong></td>
          ${selected.map((s,i)=>`<td style="font-size:12px;word-break:break-all;max-width:450px;white-space:pre-wrap;background:${colors[i]}">${summarizeValue((s.fields||{})[key])}</td>`).join('')}
        </tr>`;
      }).join('')}</tbody>
    </table>`;

  // 绑定字段名点击事件 → 打开 diff 模态框
  wrap.querySelectorAll('.diff-field-link').forEach(el => {
    el.addEventListener('click', () => {
      const field = el.dataset.field;
      if (field) showDiffModal(field, duSnapshots);
    });
  });
}

// Expose for inline onclick
window.loadDUList = loadDUList;
window.loadDUDetail = loadDUDetail;
window.showDUCompareDetail = showDUCompareDetail;
window.toggleCompareEnv = toggleCompareEnv;

export { renderDeployUnits };
