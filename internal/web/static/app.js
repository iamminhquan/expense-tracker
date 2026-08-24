// Behaviours shared by every page: CSRF, amount formatting, the gestures the
// mobile sheets rely on, the theme switch, and the popover dismissals.
//
// This file is loaded once from <head>. hx-boost swaps only <body>, so a
// boosted navigation never re-runs it -- which is exactly why everything
// here is a delegated listener on `document` rather than a listener bound to
// the elements of whichever page happened to be loaded first. Page-specific
// scripts cannot live here for the same reason: they belong inside the
// swapped content, not the head. See charts.js and categories.js.

document.addEventListener('htmx:configRequest', function (evt) {
  var meta = document.querySelector('meta[name="csrf-token"]');
  if (meta) {
    evt.detail.headers['X-CSRF-Token'] = meta.getAttribute('content');
  }
  if (evt.detail.parameters.amount) {
    evt.detail.parameters.amount = String(evt.detail.parameters.amount).replace(/\D/g, '');
  }
});

function formatAmountInput(el) {
  var digits = el.value.replace(/\D/g, '');
  // en-US, not vi-VN: this is what the user watches appear as they type an
  // amount, and it has to match the commas vnd() renders everywhere else on
  // the page.
  el.value = digits ? Number(digits).toLocaleString('en-US') : '';
}
document.addEventListener('input', function (evt) {
  if (evt.target.matches && evt.target.matches('input[name="amount"]')) {
    formatAmountInput(evt.target);
  }
});
function formatAmountInputsIn(root) {
  root.querySelectorAll('input[name="amount"]').forEach(formatAmountInput);
}
document.addEventListener('DOMContentLoaded', function () { formatAmountInputsIn(document); });
document.addEventListener('htmx:afterSwap', function () { formatAmountInputsIn(document); });
document.addEventListener('htmx:afterSettle', function () { formatAmountInputsIn(document); });

// Long-press (~500ms) on any element carrying data-longpress-target opens
// the <dialog> whose id it names -- used by mobile transaction rows to open
// the Edit/Delete action sheet without a persistent "⋯" button.
(function () {
  var timer = null, startX = 0, startY = 0, targetEl = null;
  function start(evt) {
    var el = evt.target.closest && evt.target.closest('[data-longpress-target]');
    if (!el) return;
    targetEl = el;
    var pt = evt.touches ? evt.touches[0] : evt;
    startX = pt.clientX; startY = pt.clientY;
    timer = setTimeout(function () {
      var dialog = document.getElementById(targetEl.getAttribute('data-longpress-target'));
      if (dialog) dialog.showModal();
      timer = null;
    }, 500);
  }
  function move(evt) {
    if (!timer) return;
    var pt = evt.touches ? evt.touches[0] : evt;
    if (Math.abs(pt.clientX - startX) > 10 || Math.abs(pt.clientY - startY) > 10) {
      clearTimeout(timer);
      timer = null;
    }
  }
  function cancel() {
    if (timer) { clearTimeout(timer); timer = null; }
  }
  document.addEventListener('touchstart', start, { passive: true });
  document.addEventListener('touchmove', move, { passive: true });
  document.addEventListener('touchend', cancel);
  document.addEventListener('touchcancel', cancel);
  document.addEventListener('mousedown', start);
  document.addEventListener('mousemove', move);
  document.addEventListener('mouseup', cancel);
})();

// Bottom sheets: drag the grab handle down to dismiss. The pill reads as
// draggable, so it has to behave that way -- the sheet follows the finger
// and then either snaps back or slides out. Pointer events rather than the
// touch/mouse pair above, because setPointerCapture keeps the drag alive
// when the finger slides off the handle; the handle carries touch-none so
// the gesture is not stolen by scrolling.
(function () {
  var sheet = null, startY = 0, moved = 0, startedAt = 0;

  document.addEventListener('pointerdown', function (evt) {
    var handle = evt.target.closest && evt.target.closest('[data-sheet-handle]');
    if (!handle) return;
    var dialog = handle.closest('dialog');
    if (!dialog) return;
    sheet = dialog;
    startY = evt.clientY;
    moved = 0;
    startedAt = Date.now();
    sheet.style.transition = 'none';
    sheet.style.transform = '';
    handle.setPointerCapture(evt.pointerId);
  });

  document.addEventListener('pointermove', function (evt) {
    if (!sheet) return;
    // downward only: dragging up must not lift the sheet off the bottom
    moved = Math.max(0, evt.clientY - startY);
    sheet.style.transform = 'translateY(' + moved + 'px)';
  });

  function release() {
    if (!sheet) return;
    var el = sheet, height = el.getBoundingClientRect().height;
    sheet = null;
    el.style.transition = 'transform 180ms ease-out';
    // past a quarter of the sheet, or a short flick meant as one
    if (moved > height * 0.25 || (moved > 40 && Date.now() - startedAt < 250)) {
      el.style.transform = 'translateY(100%)';
      setTimeout(function () {
        el.close();
        el.style.transition = '';
        el.style.transform = '';
      }, 180);
    } else {
      el.style.transform = '';
    }
  }
  document.addEventListener('pointerup', release);
  document.addEventListener('pointercancel', release);
})();

// htmx requests over 300ms: dim the submitting button and block further
// clicks until the request settles. No spinner, label stays put.
(function () {
  var timers = new WeakMap();
  document.addEventListener('htmx:beforeRequest', function (evt) {
    var elt = evt.detail.elt;
    if (!elt || elt.tagName !== 'BUTTON') return;
    timers.set(elt, setTimeout(function () {
      elt.classList.add('opacity-60', 'pointer-events-none');
    }, 300));
  });
  document.addEventListener('htmx:afterRequest', function (evt) {
    var elt = evt.detail.elt;
    if (!elt || elt.tagName !== 'BUTTON') return;
    var t = timers.get(elt);
    if (t) { clearTimeout(t); timers.delete(elt); }
    elt.classList.remove('opacity-60', 'pointer-events-none');
  });
})();

// Theme switch (account menu). The hx-put on each button persists the
// choice; this recolours the page immediately rather than waiting for that
// round trip, since the response carries no markup to swap. The class on
// <html> is what app.css's palette blocks key off, and it survives hx-boost
// navigations because boosting only replaces <body>.
(function () {
  var ACTIVE = ['bg-accent/10', 'text-accent', 'font-semibold'];
  var IDLE = ['text-ink-faint'];

  function applyTheme(theme) {
    document.documentElement.className = theme;
    document.querySelectorAll('[data-theme-switch] [data-theme]').forEach(function (b) {
      var on = b.getAttribute('data-theme') === theme;
      b.setAttribute('aria-pressed', on ? 'true' : 'false');
      ACTIVE.forEach(function (c) { b.classList.toggle(c, on); });
      IDLE.forEach(function (c) { b.classList.toggle(c, !on); });
    });
    // Chart.js reads its colours once at construction, so anything already
    // drawn has to be told to rebuild against the new palette.
    document.dispatchEvent(new CustomEvent('themechange'));
  }

  document.addEventListener('click', function (evt) {
    var btn = evt.target.closest && evt.target.closest('[data-theme-switch] [data-theme]');
    if (btn) applyTheme(btn.getAttribute('data-theme'));
  });

  // On "auto", the OS flipping between light and dark changes nothing in the
  // DOM -- only the media query -- so charts need the same nudge.
  var media = window.matchMedia('(prefers-color-scheme: dark)');
  media.addEventListener('change', function () {
    var cls = document.documentElement.classList;
    if (!cls.contains('light') && !cls.contains('dark')) {
      document.dispatchEvent(new CustomEvent('themechange'));
    }
  });
})();

// Close the user menu (<details data-user-menu>) or the header balance
// popover (<details data-balance-popover>) when clicking outside it.
document.addEventListener('click', function (evt) {
  document.querySelectorAll('details[data-user-menu][open], details[data-balance-popover][open]').forEach(function (el) {
    if (!el.contains(evt.target)) el.removeAttribute('open');
  });
});

// Category color-picker popovers (data-color-picker-toggle / -panel pairs in
// category_row.html): opening one closes any other that's open, and clicking
// outside a panel closes it too, instead of every open picker staying open at
// once.
document.addEventListener('click', function (evt) {
  var toggle = evt.target.closest && evt.target.closest('[data-color-picker-toggle]');
  var panel = evt.target.closest && evt.target.closest('[data-color-picker-panel]');
  document.querySelectorAll('[data-color-picker-panel]').forEach(function (p) {
    if (p !== panel && p !== (toggle && toggle.nextElementSibling)) p.classList.add('hidden');
  });
  if (toggle) toggle.nextElementSibling.classList.toggle('hidden');
});
