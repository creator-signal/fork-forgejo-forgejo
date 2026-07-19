// This file is the entry point for the code which must block page rendering / run synchronously
// before any deferred `type="module"` script executes. It is built as a classic IIFE bundle.
// bootstrap module must be the first one to be imported, it handles global errors
import './bootstrap.js';

// many users expect to use jQuery in their custom scripts (https://docs.gitea.com/administration/customizing-gitea#example-plantuml)
// so load globals (including jQuery) as early as possible
import './jquery.js';
import '../fomantic/build/semantic.js';
