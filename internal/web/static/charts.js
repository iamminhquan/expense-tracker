// The dashboard's doughnut and bar charts.
//
// Loaded by a <script src> at the end of the dashboard_month_section block,
// not from <head>: htmx re-executes script tags inside content it swaps in,
// which is what rebuilds both charts against the new #chart-data when the
// month picker replaces that section. A head script would run once and never
// again, and would not run at all after an hx-boost navigation.

// Chart.js takes plain colour values, not CSS variables, so the palette has
// to be read out of the document at build time -- and the charts rebuilt
// whenever it changes (app.js fires "themechange" on both a switch click and
// an OS flip while on "auto").
function chartColor(name) {
  return 'rgb(' + getComputedStyle(document.documentElement).getPropertyValue(name).trim() + ')';
}

// Category names are typed by the user, so they cannot go into innerHTML raw.
function escapeHTML(s) {
  var d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

// Shared singleton node -- a month switch rebuilds the charts, and one
// tooltip parked on <body> outlives every rebuild.
function pieTooltipEl() {
  var el = document.getElementById('pie-tooltip');
  if (!el) {
    el = document.createElement('div');
    el.id = 'pie-tooltip';
    el.style.cssText = 'position:fixed;z-index:50;pointer-events:none;opacity:0;' +
      'transition:opacity .12s ease;padding:7px 10px;border-radius:8px;' +
      'background:rgb(var(--c-ink));color:rgb(var(--c-app));' +
      'font-family:"Be Vietnam Pro",sans-serif;font-size:12px;line-height:1.35;' +
      'white-space:nowrap;box-shadow:0 4px 12px rgb(0 0 0 / 0.18)';
    document.body.appendChild(el);
  }
  return el;
}

// A tap has no "mouse left the canvas" counterpart, so on touch Chart.js
// keeps the slice active and the tooltip up until something else is tapped.
// Clearing the active elements as well as the node means the same slice can
// be tapped again right afterwards and still re-open.
function dismissPieTooltip() {
  pieTooltipEl().style.opacity = '0';
  var chart = window.__pieChart;
  if (!chart) return;
  chart.setActiveElements([]);
  if (chart.tooltip) chart.tooltip.setActiveElements([], { x: 0, y: 0 });
  chart.update('none');
}

function pieTooltip(ctx) {
  var el = pieTooltipEl();
  var tt = ctx.tooltip;
  if (!tt.opacity) { el.style.opacity = '0'; return; }

  var i = tt.dataPoints[0].dataIndex;
  var data = tt.dataPoints[0].dataset;
  var total = data.data.reduce(function (a, b) { return a + b; }, 0);
  var value = data.data[i];
  var pct = total ? Math.round((value / total) * 100) : 0;
  el.innerHTML =
    '<div style="display:flex;align-items:center;gap:6px">' +
      '<span style="width:8px;height:8px;border-radius:9999px;flex:none;background:' + data.backgroundColor[i] + '"></span>' +
      '<span>' + escapeHTML(tt.dataPoints[0].label) + '</span>' +
    '</div>' +
    '<div style="font-family:\'JetBrains Mono\',monospace;font-weight:600;margin-top:2px">' +
      Number(value).toLocaleString('en-US') + '₫ <span style="opacity:.6;font-weight:400">' + pct + '%</span>' +
    '</div>';

  // Centre above the hovered slice, then keep the whole box on screen -- the
  // canvas sits at the left edge of a narrow phone viewport, so an unclamped
  // tooltip would run off it.
  el.style.opacity = '1';
  var rect = ctx.chart.canvas.getBoundingClientRect();
  var box = el.getBoundingClientRect();
  var margin = 8;
  var left = rect.left + tt.caretX - box.width / 2;
  left = Math.min(Math.max(left, margin), window.innerWidth - box.width - margin);
  var top = rect.top + tt.caretY - box.height - 10;
  if (top < margin) top = rect.top + tt.caretY + 12;
  el.style.left = left + 'px';
  el.style.top = top + 'px';
}

window.__initCharts = function () {
  pieTooltipEl().style.opacity = '0';
  if (window.__pieChart) { window.__pieChart.destroy(); window.__pieChart = null; }
  if (window.__barChart) { window.__barChart.destroy(); window.__barChart = null; }
  var dataEl = document.getElementById('chart-data');
  if (!dataEl) return;
  var chartData = JSON.parse(dataEl.textContent);

  var pieCanvas = document.getElementById('pie-chart');
  if (pieCanvas && chartData.pie.labels.length > 0) {
    window.__pieChart = new Chart(pieCanvas, {
      type: 'doughnut',
      data: {
        labels: chartData.pie.labels,
        datasets: [{ data: chartData.pie.values, backgroundColor: chartData.pie.colors, borderWidth: 2, borderColor: chartColor('--c-surface') }]
      },
      options: {
        cutout: '62%',
        animation: false,
        plugins: {
          legend: { display: false },
          // Chart.js paints its built-in tooltip onto the canvas, so on
          // mobile (a 108px doughnut) the amount is clipped at the canvas
          // edge. An external tooltip is a plain DOM node instead, free to
          // overflow the canvas and be clamped to the viewport.
          tooltip: { enabled: false, external: pieTooltip, displayColors: false }
        }
      }
    });
  }

  var barCanvas = document.getElementById('bar-chart');
  if (barCanvas) {
    window.__barChart = new Chart(barCanvas, {
      type: 'bar',
      data: {
        labels: chartData.bars.labels,
        datasets: [
          { label: 'Spent', data: chartData.bars.expense, backgroundColor: chartColor('--c-expense'), borderRadius: 3, barPercentage: 0.62, categoryPercentage: 0.6 },
          { label: 'Earned', data: chartData.bars.income, backgroundColor: chartColor('--c-income'), borderRadius: 3, barPercentage: 0.62, categoryPercentage: 0.6 }
        ]
      },
      options: {
        animation: false,
        plugins: { legend: { display: false } },
        scales: {
          x: { grid: { display: false }, border: { display: false }, ticks: { color: chartColor('--c-ink-faintest'), font: { family: 'JetBrains Mono' } } },
          y: {
            beginAtZero: true, grid: { color: chartColor('--c-border-list') }, border: { display: false },
            ticks: {
              color: chartColor('--c-ink-zero'), maxTicksLimit: 4, font: { family: 'JetBrains Mono' },
              // "M" for million, matching the English UI. Chart.js gets no
              // say in the page's other numbers, so this callback is the one
              // place the axis abbreviation is decided.
              callback: function (value) { return value >= 1000000 ? (value / 1000000) + 'M' : value; }
            }
          }
        }
      }
    });
  }
};

window.__initCharts();
// This file re-runs on every dashboard render (a month switch swaps the
// whole section back in), so drop the previous listener before binding a new
// one -- otherwise each swap leaves another copy behind.
if (window.__onThemeChange) document.removeEventListener('themechange', window.__onThemeChange);
window.__onThemeChange = function () { window.__initCharts(); };
document.addEventListener('themechange', window.__onThemeChange);

if (window.__onDocPointerDown) {
  document.removeEventListener('pointerdown', window.__onDocPointerDown, true);
  window.removeEventListener('scroll', window.__onDocPointerDown, true);
  window.removeEventListener('resize', window.__onDocPointerDown);
}
// Capture phase: the tooltip has to go even when the tap lands on something
// that stops the event (a nav button, the month menu). A tap on the canvas
// itself is Chart.js's own business -- leave that one alone.
window.__onDocPointerDown = function (e) {
  if (e && e.target === document.getElementById('pie-chart')) return;
  dismissPieTooltip();
};
document.addEventListener('pointerdown', window.__onDocPointerDown, true);
// The node is position:fixed, so a scroll would otherwise leave it parked
// over the page while the slice it describes moves away.
window.addEventListener('scroll', window.__onDocPointerDown, true);
window.addEventListener('resize', window.__onDocPointerDown);
