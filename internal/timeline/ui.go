package timeline

import (
	"fmt"
	"net/http"
)

func serveUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, timelineHTML)
}

const timelineHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>KICK Timeline</title>
  <style>
    :root {
      --corewire-default: #373a3b;
      --corewire-light: #efeeee;
      --corewire-dark: #373a3b;
      --corewire-orange: #f9b233;
      --corewire-blue: #1d71b8;
      --corewire-gray: #424d4d;
      --surface: #ffffff;
      --surface-2: #f6f6f6;
      --line: #d8dddd;
      --line-strong: #8eb6d7;
      --muted: #606b6b;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "IBM Plex Sans", "Segoe UI", sans-serif;
      color: var(--corewire-default);
      background:
        radial-gradient(1200px 600px at 12% -15%, rgba(249, 178, 51, 0.18), transparent 52%),
        radial-gradient(1200px 500px at 90% 0%, rgba(29, 113, 184, 0.16), transparent 58%),
        var(--corewire-light);
      min-height: 100vh;
    }
    .shell { max-width: min(1700px, calc(100vw - 3rem)); margin: 22px auto 28px; }
    .frame {
      background: #fcfcfc;
      border: 1px solid var(--line);
      border-radius: 12px;
      box-shadow: 0 12px 28px rgba(0, 0, 0, 0.1);
      overflow: hidden;
    }
    .header {
      padding: 16px 18px;
      border-bottom: 1px solid var(--line);
      background: #f7f8f8;
    }
    h1 {
      margin: 0;
      font-size: 26px;
      letter-spacing: 0.02em;
      font-weight: 720;
    }
    .hint { margin: 6px 0 0 0; color: var(--muted); font-size: 13px; }
    .body { padding: 16px; display: grid; gap: 12px; }
    .controls,
    .filters { display: grid; gap: 8px; }
    .controls { grid-template-columns: 1.2fr auto 0.9fr 1.2fr auto; }
    .filters { grid-template-columns: 1fr 1fr 1fr auto; }
    .grid-main { display: grid; gap: 12px; grid-template-columns: 1.05fr 1.15fr 1.35fr; }
    .panel {
      border: 1px solid var(--line);
      border-radius: 10px;
      background: var(--surface);
      min-height: 220px;
      overflow: hidden;
    }
    .panel-head {
      border-bottom: 1px solid var(--line);
      padding: 10px 12px;
      background: var(--surface-2);
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
    }
    .panel-title { font-size: 14px; font-weight: 700; letter-spacing: 0.02em; }
    .panel-body { padding: 10px 12px; }
    .meta { color: var(--muted); font-size: 12px; margin-bottom: 7px; }
    .meta-tight { margin: 0; }
    input, select, button {
      min-height: 39px;
      border-radius: 999px;
      border: 1px solid var(--line);
      padding: 8px 13px;
      font-size: 14px;
      font-family: inherit;
    }
    input, select { background: white; color: var(--corewire-default); }
    input:focus, select:focus {
      outline: none;
      border-color: var(--line-strong);
      box-shadow: 0 0 0 3px rgba(29, 113, 184, 0.18);
    }
    button {
      cursor: pointer;
      border: 1px solid #9abfdd;
      background: #e7f0f8;
      color: var(--corewire-default);
      font-weight: 700;
      transition: background 150ms ease, border-color 150ms ease;
    }
    button:hover {
      background: #fff1d6;
      border-color: #edcb82;
    }
    .list-item {
      border: 1px solid #dde2e2;
      border-left: 3px solid var(--corewire-blue);
      border-radius: 8px;
      background: white;
      padding: 8px 10px;
      margin-bottom: 8px;
      cursor: pointer;
      transition: border-color 140ms ease, background 140ms ease;
    }
    .list-item:hover {
      background: #fff9ef;
      border-color: #efcf8f;
    }
    .list-item .title { font-size: 13px; font-weight: 700; }
    .list-item .sub { font-size: 12px; color: var(--muted); margin-top: 3px; }
    .badge {
      display: inline-block;
      border-radius: 999px;
      border: 1px solid #b4cee4;
      padding: 1px 7px;
      font-size: 11px;
      margin-left: 6px;
      color: var(--corewire-default);
      background: #edf5fb;
    }
    .dag {
      width: 100%;
      min-height: 280px;
      border: 1px dashed var(--line);
      border-radius: 9px;
      background: #fafafa;
      overflow: auto;
    }
    .entry {
      border: 1px solid #dde2e2;
      border-left: 3px solid var(--corewire-orange);
      border-radius: 8px;
      background: white;
      padding: 10px;
      margin-bottom: 9px;
    }
    .empty { color: var(--muted); font-size: 13px; }
    @media (max-width: 1280px) {
      .grid-main { grid-template-columns: 1fr; }
    }
    @media (max-width: 980px) {
      .controls,
      .filters { grid-template-columns: 1fr; }
      .shell { max-width: calc(100vw - 1rem); }
    }
  </style>
</head>
<body>
  <div class="shell">
    <div class="frame">
      <div class="header">
        <h1>KICK Timeline</h1>
        <p class="hint">Namespace explorer for dependency graph, KickPolicy matching, requests, and timeline flow.</p>
      </div>
      <div class="body">
        <div class="controls">
          <select id="namespace"></select>
          <button id="refreshNamespaces">Refresh Namespaces</button>
          <select id="kind">
            <option>Deployment</option>
            <option>StatefulSet</option>
            <option>DaemonSet</option>
          </select>
          <input id="name" placeholder="workload name" />
          <button id="traceBtn">Trace</button>
        </div>

        <div class="filters">
          <select id="policyFilter">
            <option value="">All policies</option>
          </select>
          <select id="kindFilter">
            <option value="All">All kinds</option>
            <option>Deployment</option>
            <option>StatefulSet</option>
            <option>DaemonSet</option>
          </select>
          <input id="nameFilter" placeholder="filter discovered workloads" />
          <button id="refreshDiscoveryBtn">Refresh Discovery</button>
        </div>

        <div class="panel">
          <div class="panel-head">
            <div class="panel-title">Namespace DAG</div>
            <div id="dagStatus" class="meta meta-tight"></div>
          </div>
          <div class="panel-body">
            <div id="dag" class="dag"></div>
            <div class="meta" style="margin-top:8px;">Edges: KickPolicy -> workload (manages), workload -> Secret or ConfigMap (dependsOn).</div>
          </div>
        </div>

        <div class="grid-main">
          <div class="panel">
            <div class="panel-head">
              <div class="panel-title">KickPolicies and KickRequests</div>
              <select id="resourceScope" style="min-height:30px; padding:4px 10px;">
                <option value="all">All namespaces</option>
                <option value="selected">Selected namespace</option>
              </select>
            </div>
            <div class="panel-body">
              <div id="resourcesStatus" class="meta"></div>
              <div class="meta"><strong>KickPolicies</strong></div>
              <div id="policyList"></div>
              <div class="meta" style="margin-top:10px;"><strong>KickRequests</strong></div>
              <div id="requestList"></div>
            </div>
          </div>

          <div class="panel">
            <div class="panel-head">
              <div class="panel-title">Discovered Workloads</div>
              <div id="discoveryStatus" class="meta meta-tight"></div>
            </div>
            <div class="panel-body">
              <div id="discovered"></div>
            </div>
          </div>

          <div class="panel">
            <div class="panel-head">
              <div class="panel-title">Timeline</div>
              <div id="status" class="meta meta-tight"></div>
            </div>
            <div class="panel-body">
              <div id="list"></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
<script>
let discoveredItems = [];

function esc(text) {
  return String(text)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function selectedNamespace() {
  return document.getElementById('namespace').value || '';
}

async function refreshNamespaces() {
  const select = document.getElementById('namespace');
  const previous = select.value;
  const res = await fetch('/timeline/namespaces');
  if (!res.ok) {
    return;
  }
  const data = await res.json();
  const items = data.items || [];
  select.innerHTML = '';
  for (const ns of items) {
    const option = document.createElement('option');
    option.value = ns;
    option.textContent = ns;
    select.appendChild(option);
  }
  if (items.includes(previous)) {
    select.value = previous;
  } else if (items.includes('default')) {
    select.value = 'default';
  } else if (items.length) {
    select.value = items[0];
  }
}

function renderDAG(data, namespace) {
  const container = document.getElementById('dag');
  container.innerHTML = '';

  const nodes = data.nodes || [];
  const edges = data.edges || [];
  if (!nodes.length) {
    container.innerHTML = '<div class="empty" style="padding:10px;">No DAG nodes found for namespace ' + esc(namespace) + '.</div>';
    return;
  }

  const policies = nodes.filter(function(node) { return node.kind === 'KickPolicy'; });
  const workloads = nodes.filter(function(node) {
    return node.kind === 'Deployment' || node.kind === 'StatefulSet' || node.kind === 'DaemonSet';
  });
  const sources = nodes.filter(function(node) { return node.kind === 'Secret' || node.kind === 'ConfigMap'; });

  const rowHeight = 68;
  const maxRows = Math.max(policies.length, workloads.length, sources.length, 1);
  const width = 980;
  const height = Math.max(260, 40 + rowHeight * maxRows);
  const xPolicy = 160;
  const xWorkload = 500;
  const xSource = 840;
  const boxW = 220;
  const boxH = 40;

  const svgNS = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(svgNS, 'svg');
  svg.setAttribute('viewBox', '0 0 ' + width + ' ' + height);
  svg.setAttribute('width', '100%');
  svg.setAttribute('height', String(height));

  const positionByID = {};
  function paintColumn(columnNodes, x, fill, stroke) {
    for (let i = 0; i < columnNodes.length; i++) {
      const node = columnNodes[i];
      const y = 40 + i * rowHeight;
      positionByID[node.id] = { x: x, y: y };

      const rect = document.createElementNS(svgNS, 'rect');
      rect.setAttribute('x', String(x - boxW / 2));
      rect.setAttribute('y', String(y));
      rect.setAttribute('width', String(boxW));
      rect.setAttribute('height', String(boxH));
      rect.setAttribute('rx', '8');
      rect.setAttribute('fill', fill);
      rect.setAttribute('stroke', stroke);
      rect.setAttribute('stroke-width', '1');
      svg.appendChild(rect);

      const text = document.createElementNS(svgNS, 'text');
      text.setAttribute('x', String(x));
      text.setAttribute('y', String(y + 23));
      text.setAttribute('text-anchor', 'middle');
      text.setAttribute('fill', '#373a3b');
      text.setAttribute('font-size', '12');
      text.textContent = node.label;
      svg.appendChild(text);

      if (node.kind === 'Deployment' || node.kind === 'StatefulSet' || node.kind === 'DaemonSet') {
        rect.style.cursor = 'pointer';
        text.style.cursor = 'pointer';
        const parts = node.label.split('/');
        if (parts.length === 2) {
          const kind = parts[0];
          const name = parts[1];
          const click = function() { traceWorkload({ namespace: namespace, kind: kind, name: name }); };
          rect.addEventListener('click', click);
          text.addEventListener('click', click);
        }
      }
    }
  }

  paintColumn(policies, xPolicy, '#f0f6fc', '#9bbddd');
  paintColumn(workloads, xWorkload, '#fff7ea', '#d8c6a4');
  paintColumn(sources, xSource, '#f7f7f7', '#cdd0d0');

  for (const edge of edges) {
    const from = positionByID[edge.from];
    const to = positionByID[edge.to];
    if (!from || !to) {
      continue;
    }
    const line = document.createElementNS(svgNS, 'line');
    line.setAttribute('x1', String(from.x + boxW / 2));
    line.setAttribute('y1', String(from.y + boxH / 2));
    line.setAttribute('x2', String(to.x - boxW / 2));
    line.setAttribute('y2', String(to.y + boxH / 2));
    line.setAttribute('stroke', edge.type === 'manages' ? '#1d71b8' : '#f9b233');
    line.setAttribute('stroke-width', '2');
    line.setAttribute('opacity', '0.8');
    svg.insertBefore(line, svg.firstChild);
  }

  container.appendChild(svg);
}

async function refreshDAG(namespace) {
  const dagStatus = document.getElementById('dagStatus');
  if (!namespace) {
    dagStatus.textContent = 'namespace is required';
    renderDAG({ nodes: [], edges: [] }, '');
    return;
  }
  dagStatus.textContent = 'building DAG';
  const res = await fetch('/timeline/dag?namespace=' + encodeURIComponent(namespace));
  if (!res.ok) {
    dagStatus.textContent = await res.text();
    renderDAG({ nodes: [], edges: [] }, namespace);
    return;
  }
  const data = await res.json();
  dagStatus.textContent = (data.nodes || []).length + ' nodes, ' + (data.edges || []).length + ' edges';
  renderDAG(data, namespace);
}

function syncPolicyFilter(items) {
  const names = new Set();
  for (const item of items) {
    names.add(item.policy);
  }
  const policySelect = document.getElementById('policyFilter');
  const current = policySelect.value;
  policySelect.innerHTML = '<option value="">All policies</option>';
  for (const name of Array.from(names).sort()) {
    const option = document.createElement('option');
    option.value = name;
    option.textContent = name;
    policySelect.appendChild(option);
  }
  policySelect.value = Array.from(names).includes(current) ? current : '';
}

function renderDiscovered(items) {
  const list = document.getElementById('discovered');
  list.innerHTML = '';
  if (!items.length) {
    list.innerHTML = '<div class="empty">No discovered workloads for this namespace and filter.</div>';
    return;
  }
  for (const item of items) {
    const node = document.createElement('div');
    node.className = 'list-item';
    node.innerHTML = '<div class="title">' + esc(item.kind + '/' + item.name) + '</div>' +
      '<div class="sub">policy: ' + esc(item.policy) + ' <span class="badge">' + esc(item.namespace) + '</span></div>';
    node.onclick = function() { traceWorkload(item); };
    list.appendChild(node);
  }
}

function renderPolicies(items) {
  const list = document.getElementById('policyList');
  list.innerHTML = '';
  if (!items.length) {
    list.innerHTML = '<div class="empty">No KickPolicies found.</div>';
    return;
  }
  for (const item of items) {
    const node = document.createElement('div');
    node.className = 'list-item';
    node.innerHTML = '<div class="title">' + esc(item.name) + '</div><div class="sub">KickPolicy <span class="badge">' + esc(item.namespace) + '</span></div>';
    node.onclick = async function() {
      document.getElementById('namespace').value = item.namespace;
      await refreshDiscovery();
      document.getElementById('policyFilter').value = item.name;
      await refreshDiscovery();
    };
    list.appendChild(node);
  }
}

function renderRequests(items) {
  const list = document.getElementById('requestList');
  list.innerHTML = '';
  if (!items.length) {
    list.innerHTML = '<div class="empty">No KickRequests found.</div>';
    return;
  }
  for (const item of items) {
    const node = document.createElement('div');
    node.className = 'list-item';
    node.innerHTML = '<div class="title">' + esc(item.targetKind + '/' + item.targetName) + '</div>' +
      '<div class="sub">request: ' + esc(item.name) + ' <span class="badge">' + esc(item.namespace) + '</span> phase: ' + esc(item.phase || 'Unknown') + '</div>';
    node.onclick = function() {
      traceWorkload({
        namespace: item.namespace,
        kind: item.targetKind,
        name: item.targetName,
      });
    };
    list.appendChild(node);
  }
}

function traceWorkload(item) {
  document.getElementById('namespace').value = item.namespace;
  document.getElementById('kind').value = item.kind;
  document.getElementById('name').value = item.name;
  loadTimeline();
}

async function refreshResources() {
  const status = document.getElementById('resourcesStatus');
  const scope = document.getElementById('resourceScope').value;
  const namespace = selectedNamespace();
  const query = scope === 'selected' && namespace ? ('?namespace=' + encodeURIComponent(namespace)) : '';

  status.textContent = 'loading resources';
  const res = await fetch('/timeline/resources' + query);
  if (!res.ok) {
    status.textContent = await res.text();
    renderPolicies([]);
    renderRequests([]);
    return;
  }
  const data = await res.json();
  const policies = data.policies || [];
  const requests = data.requests || [];
  status.textContent = policies.length + ' policies, ' + requests.length + ' requests';
  renderPolicies(policies);
  renderRequests(requests);
}

async function refreshDiscovery() {
  const ns = selectedNamespace();
  const policy = document.getElementById('policyFilter').value;
  const kind = document.getElementById('kindFilter').value;
  const name = document.getElementById('nameFilter').value.trim();
  const status = document.getElementById('discoveryStatus');
  if (!ns) {
    status.textContent = 'namespace is required';
    renderDiscovered([]);
    renderDAG({ nodes: [], edges: [] }, '');
    return;
  }

  status.textContent = 'discovering';
  const url = '/timeline/discovery?namespace=' + encodeURIComponent(ns) +
    '&policy=' + encodeURIComponent(policy) +
    '&kind=' + encodeURIComponent(kind) +
    '&name=' + encodeURIComponent(name);
  const res = await fetch(url);
  if (!res.ok) {
    status.textContent = await res.text();
    renderDiscovered([]);
    return;
  }
  const data = await res.json();
  discoveredItems = data.items || [];
  syncPolicyFilter(discoveredItems);
  status.textContent = discoveredItems.length + ' discovered';
  renderDiscovered(discoveredItems);
  await refreshDAG(ns);

  if (discoveredItems.length && !document.getElementById('name').value.trim()) {
    traceWorkload(discoveredItems[0]);
  }
}

async function loadTimeline() {
  const ns = selectedNamespace();
  const kind = document.getElementById('kind').value;
  const name = document.getElementById('name').value.trim();
  const status = document.getElementById('status');
  const list = document.getElementById('list');
  list.innerHTML = '';
  if (!ns || !name) {
    status.textContent = 'namespace and workload name are required';
    return;
  }
  status.textContent = 'loading timeline';
  const url = '/timeline?namespace=' + encodeURIComponent(ns) + '&kind=' + encodeURIComponent(kind) + '&name=' + encodeURIComponent(name);
  const res = await fetch(url);
  if (!res.ok) {
    status.textContent = await res.text();
    return;
  }
  const data = await res.json();
  const items = data.items || [];
  status.textContent = items.length + ' events';
  if (!items.length) {
    list.innerHTML = '<div class="empty">No timeline events found.</div>';
    return;
  }
  for (const item of items) {
    const node = document.createElement('div');
    node.className = 'entry';
    node.innerHTML = '<div class="meta">' + new Date(item.at).toLocaleString() + ' - ' + esc(item.type) + ' - ' + esc(item.object) + '</div>' +
      '<div><strong>' + esc(item.reason || 'info') + '</strong> - ' + esc(item.message) + '</div>';
    list.appendChild(node);
  }
}

async function refreshAll() {
  await refreshDiscovery();
  await refreshResources();
}

document.getElementById('refreshNamespaces').addEventListener('click', async function() {
  await refreshNamespaces();
  await refreshAll();
});
document.getElementById('traceBtn').addEventListener('click', loadTimeline);
document.getElementById('refreshDiscoveryBtn').addEventListener('click', refreshDiscovery);
document.getElementById('namespace').addEventListener('change', refreshAll);
document.getElementById('resourceScope').addEventListener('change', refreshResources);
document.getElementById('policyFilter').addEventListener('change', refreshDiscovery);
document.getElementById('kindFilter').addEventListener('change', refreshDiscovery);
document.getElementById('nameFilter').addEventListener('input', refreshDiscovery);

(async function init() {
  await refreshNamespaces();
  await refreshAll();
})();
</script>
</body>
</html>`
