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
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>KICK Timeline</title>
  <style>
    :root {
      --ink: #1f2427;
      --muted: #6b7678;
      --line: #e2e6e6;
      --surface: #ffffff;
      --bg: #eef0f0;
      --panel: #f8f9f9;
      --blue: #1d71b8;
      --orange: #f9b233;
      --green: #2e9e5b;
      --red: #d64545;
      --amber: #e6a010;
      --teal: #2b9c9c;
      --gray: #8a9698;
      --lane-h: 26px;
    }
    * { box-sizing: border-box; }
    html, body { height: 100%; }
    body {
      margin: 0;
      font-family: "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
      color: var(--ink);
      background: var(--bg);
      font-size: 13px;
    }
    .app { display: flex; flex-direction: column; min-height: 100vh; }
    .topbar {
      display: flex; align-items: center; justify-content: space-between; gap: 16px;
      padding: 10px 16px; background: #22282a; color: #fff;
      position: sticky; top: 0; z-index: 30;
    }
    .brand { display: flex; align-items: baseline; gap: 10px; }
    .brand h1 { margin: 0; font-size: 16px; font-weight: 700; letter-spacing: 0.02em; }
    .brand .sub { color: #aeb8ba; font-size: 12px; }
    .tools { display: flex; align-items: center; gap: 8px; }
    .tools input[type=text] {
      background: #2f3639; border: 1px solid #3d4548; color: #fff;
      border-radius: 6px; padding: 6px 10px; font-size: 12px; width: 190px; font-family: inherit;
    }
    .tools input[type=text]::placeholder { color: #8a9698; }
    .tools label { display: flex; align-items: center; gap: 5px; color: #cdd4d5; font-size: 12px; cursor: pointer; }
    .tools button {
      background: var(--blue); border: 0; color: #fff; border-radius: 6px;
      padding: 6px 12px; font-size: 12px; font-weight: 600; cursor: pointer; font-family: inherit;
    }
    .tools button:hover { background: #195f9c; }
    .status { padding: 6px 16px; color: var(--muted); font-size: 12px; background: var(--panel); border-bottom: 1px solid var(--line); }
    .legend { display: flex; flex-wrap: wrap; gap: 12px; padding: 8px 16px; background: var(--surface); border-bottom: 1px solid var(--line); }
    .legend .item { display: flex; align-items: center; gap: 6px; font-size: 11px; color: var(--muted); }
    .legend .dot { width: 10px; height: 10px; border-radius: 50%; }
    .legend .bar { width: 16px; height: 10px; border-radius: 2px; opacity: 0.5; }

    .content { display: grid; grid-template-columns: 1.6fr 1fr; gap: 12px; padding: 12px 16px; align-items: start; }
    @media (max-width: 1100px) { .content { grid-template-columns: 1fr; } }

    .card { background: var(--surface); border: 1px solid var(--line); border-radius: 10px; overflow: hidden; }
    .card-head { padding: 8px 12px; border-bottom: 1px solid var(--line); background: var(--panel); font-weight: 700; font-size: 12px; display: flex; justify-content: space-between; align-items: center; }
    .card-head .count { color: var(--muted); font-weight: 500; }

    /* Timeline swimlanes */
    .axis { display: flex; justify-content: space-between; padding: 6px 12px 6px 210px; font-size: 10px; color: var(--muted); border-bottom: 1px solid var(--line); background: var(--panel); position: relative; }
    .axis span { transform: translateX(-50%); white-space: nowrap; }
    .axis span:first-child { transform: none; }
    .axis span:last-child { transform: translateX(-100%); }
    .lanes { max-height: calc(100vh - 220px); overflow: auto; }
    .ns-group { border-bottom: 1px solid var(--line); }
    .ns-head { position: sticky; top: 0; background: #eef3f7; color: #204b70; font-weight: 700; font-size: 11px; padding: 4px 12px; letter-spacing: 0.03em; z-index: 5; border-bottom: 1px solid #dbe6f0; }
    .lane { display: grid; grid-template-columns: 198px 1fr; align-items: center; height: var(--lane-h); border-bottom: 1px solid #f1f3f3; cursor: pointer; }
    .lane:hover { background: #fafcfd; }
    .lane.active { background: #fff6e2; }
    .lane .label { display: flex; align-items: center; gap: 6px; padding: 0 8px 0 12px; overflow: hidden; }
    .lane .label .pdot { width: 8px; height: 8px; border-radius: 50%; flex: 0 0 auto; }
    .lane .label .nm { font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    .lane .label .kind { color: var(--muted); font-size: 10px; }
    .track { position: relative; height: 100%; margin-right: 12px; }
    .track::before { content: ""; position: absolute; left: 0; right: 0; top: 50%; height: 1px; background: #eceeee; }
    .seg { position: absolute; top: 50%; transform: translateY(-50%); height: 10px; border-radius: 2px; opacity: 0.42; }
    .mk { position: absolute; top: 50%; width: 9px; height: 9px; border-radius: 50%; transform: translate(-50%, -50%); border: 1.5px solid #fff; box-shadow: 0 0 0 1px rgba(0,0,0,0.12); cursor: pointer; }
    .mk.sq { border-radius: 2px; }
    .mk:hover { transform: translate(-50%, -50%) scale(1.5); z-index: 4; }

    .empty { padding: 18px 12px; color: var(--muted); font-size: 12px; }

    /* Event log */
    .log-body { max-height: calc(100vh - 220px); overflow: auto; }
    .row { display: grid; grid-template-columns: 92px 1fr; gap: 8px; padding: 6px 12px; border-bottom: 1px solid #f1f3f3; border-left: 3px solid var(--gray); }
    .row:hover { background: #fafcfd; }
    .row .t { color: var(--muted); font-size: 11px; font-variant-numeric: tabular-nums; }
    .row .t .rel { display: block; color: #aab3b4; font-size: 10px; }
    .row .who { font-size: 11px; color: var(--muted); }
    .row .who b { color: var(--ink); }
    .row .msg { margin-top: 1px; font-size: 12px; }
    .row .badge { display: inline-block; font-size: 10px; font-weight: 700; padding: 0 6px; border-radius: 999px; color: #fff; margin-right: 6px; vertical-align: 1px; }

    .tooltip {
      position: fixed; z-index: 100; pointer-events: none; display: none;
      background: #22282a; color: #fff; border-radius: 6px; padding: 7px 9px;
      font-size: 11px; max-width: 320px; box-shadow: 0 8px 22px rgba(0,0,0,0.28); line-height: 1.4;
    }
    .tooltip b { color: #ffd98a; }
  </style>
</head>
<body>
  <div class="app">
    <div class="topbar">
      <div class="brand">
        <h1>KICK Timeline</h1>
        <span class="sub">what happened, when &mdash; across all namespaces</span>
      </div>
      <div class="tools">
        <input type="text" id="search" placeholder="filter workloads / namespace" />
        <label><input type="checkbox" id="auto" /> auto</label>
        <button id="refresh">Refresh</button>
      </div>
    </div>
    <div id="status" class="status">loading&hellip;</div>
    <div class="legend" id="legend"></div>
    <div class="content">
      <div class="card">
        <div class="card-head"><span>Timeline</span><span class="count" id="tlCount"></span></div>
        <div class="axis" id="axis"></div>
        <div class="lanes" id="lanes"></div>
      </div>
      <div class="card">
        <div class="card-head"><span>Event log</span><span class="count" id="logCount"></span></div>
        <div class="log-body" id="logBody"></div>
      </div>
    </div>
  </div>
  <div id="tooltip" class="tooltip"></div>
<script>
var COLORS = {
  blue: '#1d71b8', orange: '#f9b233', green: '#2e9e5b',
  red: '#d64545', amber: '#e6a010', teal: '#2b9c9c', gray: '#8a9698'
};
var state = { workloads: [], events: [], min: 0, max: 0, filter: '', selected: '' };

function esc(t) {
  return String(t).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function phaseColor(phase) {
  var p = (phase || '').toLowerCase();
  if (p.indexOf('succeed') >= 0 || p.indexOf('complete') >= 0 || p.indexOf('nolongerrequired') >= 0 || p.indexOf('notrequired') >= 0 || p.indexOf('ready') >= 0 || p === 'done') return COLORS.green;
  if (p.indexOf('fail') >= 0 || p.indexOf('error') >= 0) return COLORS.red;
  if (p.indexOf('wait') >= 0 || p.indexOf('block') >= 0 || p.indexOf('pending') >= 0) return COLORS.amber;
  if (p.indexOf('progress') >= 0 || p.indexOf('rollout') >= 0 || p.indexOf('kick') >= 0) return COLORS.blue;
  if (!p) return COLORS.gray;
  return COLORS.blue;
}

function eventStyle(ev) {
  var type = ev.type || '';
  var text = ((ev.reason || '') + ' ' + (ev.message || '')).toLowerCase();
  if (type === 'DependencyRelevantChange') return { color: COLORS.orange, square: true, label: 'dependency change' };
  if (type === 'WorkloadRestarted') return { color: COLORS.green, square: true, label: 'workload restarted' };
  if (type === 'KickRequestCreated') return { color: COLORS.blue, square: false, label: 'kick request created' };
  if (type === 'KickRequestPhase') return { color: phaseColor(ev.message), square: false, label: 'phase: ' + (ev.message || '') };
  if (type === 'KubernetesEvent') {
    var warn = text.indexOf('fail') >= 0 || text.indexOf('error') >= 0 || text.indexOf('backoff') >= 0 || text.indexOf('unhealthy') >= 0;
    return { color: warn ? COLORS.red : COLORS.gray, square: false, label: 'k8s event' };
  }
  if (type === 'KickRequestEvent') return { color: COLORS.teal, square: false, label: 'request event' };
  return { color: COLORS.gray, square: false, label: type };
}

function pct(ts) {
  if (state.max <= state.min) return 50;
  return ((ts - state.min) / (state.max - state.min)) * 100;
}

function fmtClock(d) {
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}
function fmtFull(d) {
  return d.toLocaleString([], { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' });
}
function relTime(ts) {
  var s = Math.round((Date.now() - ts) / 1000);
  if (s < 60) return s + 's ago';
  var m = Math.round(s / 60); if (m < 60) return m + 'm ago';
  var h = Math.round(m / 60); if (h < 48) return h + 'h ago';
  return Math.round(h / 24) + 'd ago';
}

var tooltip = document.getElementById('tooltip');
function showTip(html, x, y) {
  tooltip.innerHTML = html;
  tooltip.style.display = 'block';
  var tx = x + 14, ty = y + 14;
  if (tx + tooltip.offsetWidth > window.innerWidth) tx = x - tooltip.offsetWidth - 14;
  if (ty + tooltip.offsetHeight > window.innerHeight) ty = y - tooltip.offsetHeight - 14;
  tooltip.style.left = tx + 'px';
  tooltip.style.top = ty + 'px';
}
function hideTip() { tooltip.style.display = 'none'; }

function tipFor(ev) {
  var st = eventStyle(ev);
  return '<b>' + esc(st.label) + '</b><br>' +
    esc(fmtFull(new Date(ev.at))) + '<br>' +
    esc(ev.kind + '/' + ev.name) + ' &middot; ' + esc(ev.namespace) +
    (ev.reason ? '<br>reason: ' + esc(ev.reason) : '') +
    (ev.message ? '<br>' + esc(ev.message) : '');
}

function matchFilter(w) {
  if (!state.filter) return true;
  var f = state.filter.toLowerCase();
  return (w.namespace + ' ' + w.kind + ' ' + w.name + ' ' + (w.policy || '')).toLowerCase().indexOf(f) >= 0;
}

function renderLegend() {
  var items = [
    ['dot', COLORS.orange, 'dependency change'],
    ['dot', COLORS.green, 'workload restarted'],
    ['dot', COLORS.blue, 'kick request'],
    ['dot', COLORS.amber, 'waiting / blocked'],
    ['dot', COLORS.red, 'failed / warning'],
    ['dot', COLORS.gray, 'k8s event'],
    ['bar', COLORS.blue, 'state over time (band)']
  ];
  var el = document.getElementById('legend');
  el.innerHTML = '';
  items.forEach(function(it) {
    var d = document.createElement('div'); d.className = 'item';
    var m = document.createElement('span'); m.className = it[0]; m.style.background = it[1];
    d.appendChild(m);
    d.appendChild(document.createTextNode(it[2]));
    el.appendChild(d);
  });
}

function renderAxis() {
  var axis = document.getElementById('axis');
  axis.innerHTML = '';
  if (!state.events.length) return;
  var ticks = 5;
  for (var i = 0; i < ticks; i++) {
    var ts = state.min + ((state.max - state.min) * i) / (ticks - 1);
    var s = document.createElement('span');
    s.textContent = fmtClock(new Date(ts));
    axis.appendChild(s);
  }
}

function laneSegments(events) {
  // Colored state bands between successive KickRequestPhase events.
  var phases = events.filter(function(e) { return e.type === 'KickRequestPhase'; })
    .sort(function(a, b) { return new Date(a.at) - new Date(b.at); });
  var segs = [];
  for (var i = 0; i < phases.length; i++) {
    var start = new Date(phases[i].at).getTime();
    var end = i + 1 < phases.length ? new Date(phases[i + 1].at).getTime() : state.max;
    if (end <= start) end = start + (state.max - state.min) * 0.01;
    segs.push({ left: pct(start), width: Math.max(0.6, pct(end) - pct(start)), color: phaseColor(phases[i].message) });
  }
  return segs;
}

function laneKey(w) { return w.namespace + '/' + w.kind + '/' + w.name; }

function renderLanes() {
  var lanes = document.getElementById('lanes');
  lanes.innerHTML = '';
  var visible = state.workloads.filter(matchFilter);
  document.getElementById('tlCount').textContent = visible.length + ' workloads';
  if (!visible.length) {
    lanes.innerHTML = '<div class="empty">No managed workloads discovered. Apply a KickPolicy that selects a workload.</div>';
    return;
  }
  var currentNs = null, group = null;
  visible.forEach(function(w) {
    if (w.namespace !== currentNs) {
      currentNs = w.namespace;
      group = document.createElement('div'); group.className = 'ns-group';
      var head = document.createElement('div'); head.className = 'ns-head';
      head.textContent = w.namespace;
      group.appendChild(head);
      lanes.appendChild(group);
    }
    var lane = document.createElement('div');
    lane.className = 'lane' + (state.selected === laneKey(w) ? ' active' : '');
    lane.onclick = function() {
      state.selected = state.selected === laneKey(w) ? '' : laneKey(w);
      renderLanes(); renderLog();
    };

    var label = document.createElement('div'); label.className = 'label';
    var pdot = document.createElement('span'); pdot.className = 'pdot';
    pdot.style.background = phaseColor(w.phase);
    pdot.title = w.phase ? ('phase: ' + w.phase + (w.gateReason ? ' (' + w.gateReason + ')' : '')) : 'no active request';
    label.appendChild(pdot);
    var nm = document.createElement('span'); nm.className = 'nm';
    nm.innerHTML = esc(w.name) + ' <span class="kind">' + esc(w.kind) + '</span>';
    label.appendChild(nm);
    lane.appendChild(label);

    var track = document.createElement('div'); track.className = 'track';
    laneSegments(w.events).forEach(function(sg) {
      var seg = document.createElement('div'); seg.className = 'seg';
      seg.style.left = sg.left + '%'; seg.style.width = sg.width + '%'; seg.style.background = sg.color;
      track.appendChild(seg);
    });
    w.events.forEach(function(ev) {
      var st = eventStyle(ev);
      var mk = document.createElement('div'); mk.className = 'mk' + (st.square ? ' sq' : '');
      mk.style.left = pct(new Date(ev.at).getTime()) + '%';
      mk.style.background = st.color;
      mk.addEventListener('mousemove', function(e) { showTip(tipFor(ev), e.clientX, e.clientY); });
      mk.addEventListener('mouseleave', hideTip);
      track.appendChild(mk);
    });
    lane.appendChild(track);
    group.appendChild(lane);
  });
}

function shortType(type) {
  switch (type) {
    case 'DependencyRelevantChange': return 'dep';
    case 'WorkloadRestarted': return 'restart';
    case 'KickRequestCreated': return 'request';
    case 'KickRequestPhase': return 'phase';
    case 'KubernetesEvent': return 'k8s';
    case 'KickRequestEvent': return 'req-evt';
    default: return 'evt';
  }
}

function renderLog() {
  var body = document.getElementById('logBody');
  body.innerHTML = '';
  var evs = state.events.filter(function(e) {
    if (state.selected && laneKey(e) !== state.selected) return false;
    if (state.filter) {
      var f = state.filter.toLowerCase();
      if ((e.namespace + ' ' + e.kind + ' ' + e.name).toLowerCase().indexOf(f) < 0) return false;
    }
    return true;
  });
  document.getElementById('logCount').textContent = evs.length + ' events';
  if (!evs.length) {
    body.innerHTML = '<div class="empty">No events recorded yet.</div>';
    return;
  }
  evs.slice(0, 400).forEach(function(ev) {
    var st = eventStyle(ev);
    var d = new Date(ev.at);
    var row = document.createElement('div'); row.className = 'row';
    row.style.borderLeftColor = st.color;
    var t = document.createElement('div'); t.className = 't';
    t.innerHTML = esc(fmtClock(d)) + '<span class="rel">' + esc(relTime(d.getTime())) + '</span>';
    row.appendChild(t);
    var main = document.createElement('div');
    var badge = '<span class="badge" style="background:' + st.color + '">' + esc(shortType(ev.type)) + '</span>';
    main.innerHTML = '<div class="who">' + badge + '<b>' + esc(ev.kind + '/' + ev.name) + '</b> &middot; ' + esc(ev.namespace) + '</div>' +
      '<div class="msg">' + esc(ev.message || st.label) + (ev.reason ? ' <span style="color:var(--muted)">(' + esc(ev.reason) + ')</span>' : '') + '</div>';
    row.appendChild(main);
    body.appendChild(row);
  });
}

function computeBounds() {
  var times = state.events.map(function(e) { return new Date(e.at).getTime(); });
  if (!times.length) { state.min = Date.now() - 3600000; state.max = Date.now(); return; }
  state.min = Math.min.apply(null, times);
  state.max = Math.max.apply(null, times);
  if (state.max === state.min) { state.min -= 300000; state.max += 300000; }
  else { var pad = (state.max - state.min) * 0.04; state.min -= pad; state.max += pad; }
}

function renderAll() {
  computeBounds();
  renderLegend();
  renderAxis();
  renderLanes();
  renderLog();
}

async function load() {
  var status = document.getElementById('status');
  status.textContent = 'loading\u2026';
  try {
    var res = await fetch('/timeline/overview');
    if (!res.ok) { status.textContent = 'error: ' + (await res.text()); return; }
    var data = await res.json();
    state.workloads = data.workloads || [];
    state.events = data.events || [];
    var nsSet = {};
    state.workloads.forEach(function(w) { nsSet[w.namespace] = true; });
    status.textContent = state.workloads.length + ' workloads across ' + Object.keys(nsSet).length +
      ' namespaces \u00b7 ' + state.events.length + ' events \u00b7 updated ' + fmtClock(new Date());
    renderAll();
  } catch (e) {
    status.textContent = 'error: ' + e;
  }
}

document.getElementById('refresh').addEventListener('click', load);
document.getElementById('search').addEventListener('input', function(e) {
  state.filter = e.target.value.trim();
  renderLanes(); renderLog();
});
var timer = null;
document.getElementById('auto').addEventListener('change', function(e) {
  if (e.target.checked) { timer = setInterval(load, 8000); } else { clearInterval(timer); timer = null; }
});
window.addEventListener('scroll', hideTip, true);

load();
</script>
</body>
</html>`
