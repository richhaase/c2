package report

const reportStyleBlock = `</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    background: #0d1117;
    color: #c9d1d9;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    line-height: 1.6;
    padding: 24px;
    max-width: 960px;
    margin: 0 auto;
  }
  h1 { color: #f0f6fc; font-size: 28px; font-weight: 700; }
  h2 { color: #f0f6fc; font-size: 20px; font-weight: 600; margin-bottom: 16px; }
  .subtitle { color: #8b949e; font-size: 15px; margin-top: 4px; }
  .date { color: #8b949e; font-size: 13px; margin-top: 2px; }
  .muted { color: #8b949e; }
  .green { color: #3fb950; }
  .red { color: #f85149; }
  .blue { color: #58a6ff; }

  header { margin-bottom: 32px; }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    margin-bottom: 32px;
  }
  .stat-card {
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 8px;
    padding: 16px;
  }
  .stat-card .label {
    color: #8b949e;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 4px;
  }
  .stat-card .value {
    color: #f0f6fc;
    font-size: 24px;
    font-weight: 700;
  }
  .stat-card .unit {
    color: #8b949e;
    font-size: 13px;
    font-weight: 400;
  }

  .section {
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 8px;
    padding: 20px;
    margin-bottom: 24px;
  }

  .prose { font-size: 14px; }
  .prose p { margin-bottom: 10px; }
  .prose h3 { color: #f0f6fc; font-size: 15px; font-weight: 600; margin: 14px 0 6px; }
  .prose h4 { color: #c9d1d9; font-size: 13px; font-weight: 600; margin: 12px 0 4px; }
  .prose ul { margin: 0 0 10px 20px; }
  .prose li { margin-bottom: 4px; }

  .note-row { padding: 10px 0; border-bottom: 1px solid #21262d; }
  .note-row:last-child { border-bottom: none; }
  .note-meta { color: #8b949e; font-size: 11px; text-transform: uppercase; letter-spacing: 0.3px; margin-bottom: 3px; }
  .note-body { font-size: 13px; }

  .progress-container {
    position: relative;
    background: #21262d;
    border-radius: 6px;
    height: 32px;
    margin: 16px 0 8px;
    overflow: visible;
  }
  .progress-fill {
    height: 100%;
    border-radius: 6px;
    background: #f85149;
    position: relative;
    z-index: 1;
    min-width: 2px;
  }
  .progress-marker {
    position: absolute;
    top: -6px;
    height: 44px;
    width: 2px;
    background: #3fb950;
    z-index: 2;
  }
  .progress-marker-label {
    position: absolute;
    top: -22px;
    transform: translateX(-50%);
    font-size: 11px;
    color: #3fb950;
    white-space: nowrap;
    font-weight: 600;
  }
  .progress-label-row {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: #8b949e;
    margin-top: 4px;
  }

  .week-row {
    display: flex;
    align-items: center;
    margin-bottom: 8px;
    font-size: 13px;
  }
  .week-label {
    width: 70px;
    flex-shrink: 0;
    color: #8b949e;
    font-size: 12px;
    text-align: right;
    padding-right: 10px;
  }
  .week-bar-container {
    flex: 1;
    position: relative;
    height: 24px;
    background: #21262d;
    border-radius: 4px;
    overflow: visible;
  }
  .week-bar {
    height: 100%;
    border-radius: 4px;
    min-width: 2px;
  }
  .week-bar.on-pace { background: #238636; }
  .week-bar.behind { background: #8b2a2d; }
  .week-meta {
    width: 140px;
    flex-shrink: 0;
    text-align: right;
    font-size: 12px;
    color: #8b949e;
    padding-left: 8px;
  }
  .week-meta .meters { color: #c9d1d9; font-weight: 500; }
  .week-target-line {
    position: absolute;
    top: -2px;
    height: 28px;
    width: 0;
    border-left: 2px dashed #58a6ff;
    z-index: 2;
    opacity: 0.7;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }
  thead th {
    text-align: left;
    color: #8b949e;
    font-weight: 600;
    padding: 8px 10px;
    border-bottom: 1px solid #30363d;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }
  th.r, td.r { text-align: right; }
  tbody td {
    padding: 8px 10px;
    border-bottom: 1px solid #21262d;
    font-variant-numeric: tabular-nums;
  }
  tbody tr:last-child td { border-bottom: none; }
  tbody tr:hover { background: #1c2128; }

  .projection-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .projection-card {
    background: #21262d;
    border-radius: 6px;
    padding: 16px;
  }
  .projection-card h3 {
    font-size: 14px;
    font-weight: 600;
    margin-bottom: 8px;
  }
  .projection-card .big-num {
    font-size: 28px;
    font-weight: 700;
    margin-bottom: 4px;
  }
  .projection-card .detail {
    font-size: 12px;
    color: #8b949e;
    line-height: 1.8;
  }

  .target-legend {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: #8b949e;
    margin-bottom: 12px;
    justify-content: flex-end;
    padding-right: 140px;
  }
  .target-legend-line {
    width: 16px;
    border-top: 2px dashed #58a6ff;
  }

  @media (max-width: 640px) {
    .stats-grid { grid-template-columns: repeat(2, 1fr); }
    .projection-grid { grid-template-columns: 1fr; }
    body { padding: 16px; }
  }
</style>
</head>
<body>

<header>
  <h1>Rowing Progress</h1>
  <div class="subtitle">`
