// DO NOT IMPORT window.config HERE!
// to make sure the error handler always works, we should never import `window.config`, because
// some user's custom template breaks it.
import {showGlobalErrorMessage, processWindowErrorEvent} from './modules/errors.js';

// This sets up the URL prefix used in webpack's chunk loading.
// This file must be imported before any lazy-loading is being attempted.
__webpack_public_path__ = `${window.config?.assetUrlPrefix ?? '/assets'}/`;

function initGlobalErrorHandler() {
  if (window._globalHandlerErrors?._inited) {
    showGlobalErrorMessage(`The global error handler has been initialized, do not initialize it again`);
    return;
  }
  if (!window.config) {
    showGlobalErrorMessage(`Gitea JavaScript code couldn't run correctly, please check your custom templates`);
  }
  // we added an event handler for window error at the very beginning of <script> of page head the
  // handler calls `_globalHandlerErrors.push` (array method) to record all errors occur before
  // this init then in this init, we can collect all error events and show them.
  for (const e of window._globalHandlerErrors || []) {
    processWindowErrorEvent(e);
  }
  // then, change _globalHandlerErrors to an object with push method, to process further error
  // events directly
  window._globalHandlerErrors = {_inited: true, push: (e) => processWindowErrorEvent(e)};
}

initGlobalErrorHandler();
