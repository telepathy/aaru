import { API, currentUser, api } from './api.js';

// ===== UI Helpers =====
function toast(msg, type = 'info') {
  const c = document.getElementById('toast-container');
  const t = document.createElement('div');
  t.className = 'toast toast-' + type;
  t.textContent = msg;
  c.appendChild(t);
  setTimeout(() => t.remove(), 3000);
}

function showLoading(container) {
  container.innerHTML = '<div class="loading-state"><div class="spinner"></div><p>加载中...</p></div>';
}

function statusHTML(status) {
  const names = {draft:'草稿',in_progress:'进行中',approved:'已通过',pushing:'推送中',completed:'已完成',rejected:'已驳回',failed:'失败',rolled_back:'已回滚',deprecated:'已废弃',pending:'待处理',skipped:'已跳过'};
  return `<span class="status-badge status-${status}">${names[status]||status}</span>`;
}

function setPage(name, title, subtitle) {
  document.querySelectorAll('.nav-item').forEach(n=>n.classList.remove('active'));
  const nav = document.querySelector(`.nav-item[data-page="${name}"]`);
  if (nav) nav.classList.add('active');
  document.getElementById('page-title').textContent = title;
  document.getElementById('page-subtitle').textContent = subtitle || '';
}

function escapeHtml(s) {
  if (s==null) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

function fmtTime(t) {
  if (!t) return '-';
  try { return new Date(t).toLocaleString('zh-CN',{month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit'}); }
  catch(e) { return String(t); }
}

// 根据用户 allowed_silos 过滤 DU 列表
function filterDUsByPermission(dus) {
  if (!currentUser || !currentUser.allowed_silos) return [];
  if (currentUser.allowed_silos === '*') return dus;
  const allowed = currentUser.allowed_silos.split(',').map(s=>s.trim()).filter(Boolean);
  if (allowed.length === 0) return [];
  return dus.filter(d => allowed.includes(d.silo));
}

// ===== Value Display =====
const VALUE_COLORS = [
  '#dbeafe', // 蓝
  '#dcfce7', // 绿
  '#fef9c3', // 黄
  '#fce7f3', // 粉
  '#ede9fe', // 紫
  '#ffedd5', // 橙
  '#ccfbf1', // 青
  '#e0e7ff', // 靛
  '#fae8ff', // 洋红
  '#f0f9ff', // 天蓝
];

// 为一行的值分配颜色：相同值→相同颜色，不同值→不同颜色
function crAssignValueColors(vals) {
  const valColorMap = new Map();
  let colorIdx = 0;
  return vals.map(v => {
    const key = String(v);
    if (!valColorMap.has(key)) {
      valColorMap.set(key, VALUE_COLORS[colorIdx % VALUE_COLORS.length]);
      colorIdx++;
    }
    return valColorMap.get(key);
  });
}

function crArrSummary(v) {
  if (!v||v==='null'||v==='[]') return '<span class="same-val">空</span>';
  try {
    const a=JSON.parse(v);
    if(Array.isArray(a)) {
      if (a.length===0) return '<span class="same-val">空</span>';
      const preview = a.slice(0,3).map(item=>{
        if (typeof item==='object'&&item!==null) {
          const keys = Object.keys(item).slice(0,3).map(k=>`${k}:${JSON.stringify(item[k]).substring(0,20)}`).join(', ');
          return `{${keys}${Object.keys(item).length>3?', ...':''}}`;
        }
        return String(item).substring(0,30);
      }).join(', ');
      return `<span style="font-size:11px">${escapeHtml(preview)}${a.length>3?', ...':''}</span> <span style="font-size:10px;color:var(--text-muted)">(${a.length}项)</span>`;
    }
  } catch(e) {}
  return escapeHtml(String(v).substring(0,80));
}

// 摘要展示复杂值：JSON对象/数组显示预览，长字符串截断
function summarizeValue(v) {
  if (v === null || v === undefined || v === '') return '-';
  const s = String(v);
  // JSON对象
  if (s.startsWith('{') && s.length > 80) {
    try {
      const obj = JSON.parse(s);
      const keys = Object.keys(obj);
      if (keys.length === 0) return '{}';
      const preview = keys.slice(0, 3).map(k => {
        const val = obj[k];
        const vs = typeof val === 'object' ? (Array.isArray(val) ? `[...]` : '{...}') : JSON.stringify(val);
        return `${k}: ${String(vs).substring(0, 24)}`;
      }).join(', ');
      return `<span style="font-size:11px">{${preview}${keys.length > 3 ? ', ...' : ''}}</span> <span style="font-size:10px;color:var(--text-muted)">(${keys.length}个字段)</span>`;
    } catch(e) {}
  }
  // JSON数组
  if (s.startsWith('[') && s.length > 80) {
    try {
      const arr = JSON.parse(s);
      if (Array.isArray(arr)) {
        if (arr.length === 0) return '[]';
        const first = arr[0];
        if (typeof first === 'object' && first !== null) {
          const keys = Object.keys(first).slice(0, 2).map(k => `${k}:${JSON.stringify(first[k]).substring(0, 16)}`).join(', ');
          return `<span style="font-size:11px">[{${keys}}, ...] <span style="color:var(--text-muted)">(${arr.length}项)</span></span>`;
        }
        return `<span style="font-size:11px">[${JSON.stringify(first).substring(0, 30)}${arr.length > 1 ? ', ...' : ''}] <span style="color:var(--text-muted)">(${arr.length}项)</span></span>`;
      }
    } catch(e) {}
  }
  // 长字符串截断
  if (s.length > 100) {
    return `<span style="font-size:11px" title="${escapeHtml(s)}">${escapeHtml(s.substring(0, 80))}...</span> <span style="font-size:10px;color:var(--text-muted)">(${s.length}字符)</span>`;
  }
  return escapeHtml(s);
}

// 格式化 JSON 字符串（用于显示）
function crFormatJson(v) {
  if (!v || v === 'null' || v === '[]') return v || '';
  try { return JSON.stringify(JSON.parse(v), null, 2); } catch(e) {}
  return String(v);
}

// 预览页专用：复杂值按实际格式展示，简单值直接显示
function crFormatPreviewValue(v) {
  if (v === null || v === undefined || v === '') return '<span style="color:var(--text-muted)">-</span>';
  const s = String(v);
  // JSON 数组或对象 → 格式化后以等宽块展示
  if ((s.startsWith('[') || s.startsWith('{')) && s.length > 1) {
    try {
      const parsed = JSON.parse(s);
      const formatted = JSON.stringify(parsed, null, 2);
      return `<pre style="margin:0;font-size:11px;font-family:monospace;white-space:pre-wrap;word-break:break-all;max-height:300px;overflow:auto;background:#f9fafb;padding:4px 6px;border-radius:4px">${escapeHtml(formatted)}</pre>`;
    } catch(e) {}
  }
  // 多行字符串
  if (s.includes('\n')) {
    return `<pre style="margin:0;font-size:11px;font-family:monospace;white-space:pre-wrap;word-break:break-all">${escapeHtml(s)}</pre>`;
  }
  return escapeHtml(s);
}

// ===== InitDb URL Auto-sync =====
const CR_INIT_DB_FIELDS = ['initDb', 'initDbAuth', 'initDbFinal', 'ImportData'];

// 替换 git blob URL 中的 tag 部分
// URL 格式: https://git.example.com/repo/blob/TAG/path/to/file
function replaceUrlTag(url, newTag) {
  const idx = url.indexOf('/blob/');
  if (idx < 0) return url;
  const after = url.substring(idx + 6); // skip '/blob/'
  const slashIdx = after.indexOf('/');
  if (slashIdx < 0) return url;
  return url.substring(0, idx + 6) + newTag + after.substring(slashIdx);
}

// 对 initDb 类数组字段，将所有 source URL 中的 tag 替换为新版本
// 返回更新后的 JSON 字符串，如果无变化返回 null
function autoUpdateInitDbUrls(currentVal, newVersion) {
  if (!currentVal || !newVersion) return null;
  let arr;
  try { arr = JSON.parse(typeof currentVal === 'string' ? currentVal : JSON.stringify(currentVal)); } catch(e) { return null; }
  if (!Array.isArray(arr) || arr.length === 0) return null;
  let changed = false;
  const updated = arr.map(item => {
    if (!item || typeof item !== 'object') return item;
    const source = item.source;
    if (!source || typeof source !== 'string') return item;
    const idx = source.indexOf('/blob/');
    if (idx < 0) return item;
    const after = source.substring(idx + 6);
    const slashIdx = after.indexOf('/');
    if (slashIdx < 0) return item;
    const oldTag = after.substring(0, slashIdx);
    if (oldTag === newVersion) return item;
    changed = true;
    return {...item, source: replaceUrlTag(source, newVersion)};
  });
  return changed ? JSON.stringify(updated) : null;
}

// ===== Diff Utilities =====
function formatForDiff(v) {
  if (v === null || v === undefined || v === '') return '';
  const s = String(v);
  if (s.startsWith('{') || s.startsWith('[')) {
    try { return JSON.stringify(JSON.parse(s), null, 2); } catch(e) {}
  }
  return s;
}

function computeLineDiff(aLines, bLines) {
  const aLen = aLines.length, bLen = bLines.length;
  const dp = Array.from({length: aLen+1}, ()=>new Array(bLen+1).fill(0));
  for (let i=1; i<=aLen; i++) {
    for (let j=1; j<=bLen; j++) {
      dp[i][j] = aLines[i-1]===bLines[j-1] ? dp[i-1][j-1]+1 : Math.max(dp[i-1][j], dp[i][j-1]);
    }
  }
  const result = [];
  let i=aLen, j=bLen;
  while (i>0 || j>0) {
    if (i>0 && j>0 && aLines[i-1]===bLines[j-1]) {
      result.unshift({type:'ctx', aNum:i, bNum:j, text:aLines[i-1]});
      i--; j--;
    } else if (j>0 && (i===0 || dp[i][j-1]>=dp[i-1][j])) {
      result.unshift({type:'add', bNum:j, text:bLines[j-1]});
      j--;
    } else {
      result.unshift({type:'del', aNum:i, text:aLines[i-1]});
      i--;
    }
  }
  return result;
}

export {
  toast, showLoading, statusHTML, setPage, escapeHtml, fmtTime,
  filterDUsByPermission, VALUE_COLORS, crAssignValueColors,
  crArrSummary, summarizeValue, crFormatJson, crFormatPreviewValue,
  CR_INIT_DB_FIELDS, replaceUrlTag, autoUpdateInitDbUrls,
  formatForDiff, computeLineDiff,
};
