import htmx from 'htmx.org';
import {showErrorToast} from './modules/toast.js';

// webpack's ProvidePlugin used to make `htmx` a free-standing global wherever referenced
// (e.g. features/repo-diff.js calls bare `htmx.process(...)`); replicate that via `window.htmx`.
window.htmx = htmx;

// https://github.com/bigskysoftware/idiomorph#htmx
// NOTE: this must be a dynamic import. idiomorph-ext.js reads the bare global `htmx` eagerly at
// module-evaluation time, but ESM static imports are hoisted and evaluated before this file's own
// `window.htmx = htmx` assignment above, regardless of source order — so a static import here would
// throw "htmx is not defined". A dynamic import defers evaluation until this line actually runs.
import('idiomorph/dist/idiomorph-ext.js');

// https://htmx.org/reference/#config
htmx.config.requestClass = 'is-loading';
htmx.config.scrollIntoViewOnBoost = false;

// https://htmx.org/events/#htmx:sendError
document.body.addEventListener('htmx:sendError', (event) => {
  // TODO: add translations
  showErrorToast(`Network error when calling ${event.detail.requestConfig.path}`);
});

// https://htmx.org/events/#htmx:responseError
document.body.addEventListener('htmx:responseError', (event) => {
  // hide any previous flash message to avoid confusions (in case the
  // error toast would have been shown over a success/info message)
  const flashMsgDiv = document.getElementById('flash-message');
  if (flashMsgDiv) {
    flashMsgDiv.innerHTML = '';
    flashMsgDiv.className = '';
  }
  // TODO: add translations
  showErrorToast(`Error ${event.detail.xhr.status} when calling ${event.detail.requestConfig.path}: ${event.detail.xhr.responseText}`);
});
