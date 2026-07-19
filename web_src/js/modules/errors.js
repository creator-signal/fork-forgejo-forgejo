// Pure helpers with no top-level side effects, split out of bootstrap.js so they can safely be
// bundled into both the iife.js and index.js entries (Vite/Rolldown builds each as an independent
// module graph, so any file with unconditional top-level side effects — like bootstrap.js's
// initGlobalErrorHandler() call — must not be imported from both).
export function showGlobalErrorMessage(msg) {
  const pageContent = document.querySelector('.page-content');
  if (!pageContent) return;

  // compact the message to a data attribute to avoid too many duplicated messages
  const msgCompact = msg.replace(/\W/g, '').trim();
  let msgDiv = pageContent.querySelector(`.js-global-error[data-global-error-msg-compact="${msgCompact}"]`);
  if (!msgDiv) {
    const el = document.createElement('div');
    el.innerHTML = `<div class="ui container negative message center aligned js-global-error tw-mt-[15px] tw-whitespace-pre-line"></div>`;
    msgDiv = el.childNodes[0];
  }
  // merge duplicated messages into "the message (count)" format
  const msgCount = Number(msgDiv.getAttribute(`data-global-error-msg-count`)) + 1;
  msgDiv.setAttribute(`data-global-error-msg-compact`, msgCompact);
  msgDiv.setAttribute(`data-global-error-msg-count`, msgCount.toString());
  msgDiv.textContent = msg + (msgCount > 1 ? ` (${msgCount})` : '');
  pageContent.prepend(msgDiv);
}

// Ignore external and some known internal errors that we are unable to currently fix.
function shouldIgnoreError(err) {
  const assetBaseUrl = String(new URL(window.config?.assetUrlPrefix ?? '/assets', window.location.origin));

  if (!(err instanceof Error)) return false;
  // If the error stack trace does not include the base URL of our script assets, it likely came
  // from a browser extension or inline script. Ignore these errors.
  if (!err.stack?.includes(assetBaseUrl)) return true;
  return false;
}

/**
 * @param {ErrorEvent|PromiseRejectionEvent} event - Event
 * @param {string} event.message - Only present on ErrorEvent
 * @param {string} event.error - Only present on ErrorEvent
 * @param {string} event.type - Only present on ErrorEvent
 * @param {string} event.filename - Only present on ErrorEvent
 * @param {number} event.lineno - Only present on ErrorEvent
 * @param {number} event.colno - Only present on ErrorEvent
 * @param {string} event.reason - Only present on PromiseRejectionEvent
 * @param {number} event.promise - Only present on PromiseRejectionEvent
 */
export function processWindowErrorEvent({error, reason, message, type, filename, lineno, colno}) {
  const err = error ?? reason;
  const {runModeIsProd} = window.config ?? {};

  // `error` and `reason` are not guaranteed to be errors. If the value is falsy, it is likely a
  // non-critical event from the browser. We log them but don't show them to users. Examples:
  // - https://developer.mozilla.org/en-US/docs/Web/API/ResizeObserver#observation_errors
  // - https://github.com/mozilla-mobile/firefox-ios/issues/10817
  // - https://github.com/go-gitea/gitea/issues/20240
  if (!err) {
    if (message) console.error(new Error(message));
    if (runModeIsProd) return;
  }

  // In production do not display errors that should be ignored.
  if (runModeIsProd && shouldIgnoreError(err)) return;

  let msg = err?.message ?? message;
  if (lineno) msg += ` (${filename} @ ${lineno}:${colno})`;
  const dot = msg.endsWith('.') ? '' : '.';
  const renderedType = type === 'unhandledrejection' ? 'promise rejection' : type;
  showGlobalErrorMessage(`JavaScript ${renderedType}: ${msg}${dot} Open browser console to see more details.`);
}
