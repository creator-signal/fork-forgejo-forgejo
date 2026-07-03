// Boots the monaco-vscode-api workbench in the browser over ForgejoFS (real repo
// I/O via /_ide). Web workers must be wired via MonacoEnvironment before init.
// Upgrading monaco-vscode-api? Read ./UPGRADING.md first (all @codingame/* move together).
import { initialize as initializeMonacoService, LogLevel } from '@codingame/monaco-vscode-api'
import * as vscode from 'vscode'

import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import ExtensionHostWorker from '@codingame/monaco-vscode-api/workers/extensionHost.worker?worker'
import TextMateWorker from '@codingame/monaco-vscode-textmate-service-override/worker?worker'
import OutputLinkWorker from '@codingame/monaco-vscode-output-service-override/worker?worker'
import SearchWorker from '@codingame/monaco-vscode-search-service-override/worker?worker'

const workerFactories: Record<string, new () => Worker> = {
  editorWorkerService: EditorWorker,
  TextMateWorker,
  OutputLinkDetectionWorker: OutputLinkWorker,
  LocalFileSearchWorker: SearchWorker,
  extensionHostWorkerMain: ExtensionHostWorker
}
;(globalThis as any).MonacoEnvironment = {
  getWorker(_moduleId: unknown, label: string): Worker {
    const Factory = workerFactories[label]
    if (Factory == null) throw new Error(`Unregistered worker: ${label}`)
    return new Factory()
  }
}

import getExtensionServiceOverride from '@codingame/monaco-vscode-extensions-service-override'
import getLogServiceOverride from '@codingame/monaco-vscode-log-service-override'
import getModelServiceOverride from '@codingame/monaco-vscode-model-service-override'
import getNotificationsServiceOverride from '@codingame/monaco-vscode-notifications-service-override'
import getDialogsServiceOverride from '@codingame/monaco-vscode-dialogs-service-override'
import getConfigurationServiceOverride, {
  initUserConfiguration
} from '@codingame/monaco-vscode-configuration-service-override'
import getKeybindingsServiceOverride from '@codingame/monaco-vscode-keybindings-service-override'
import getTextmateServiceOverride from '@codingame/monaco-vscode-textmate-service-override'
import getThemeServiceOverride from '@codingame/monaco-vscode-theme-service-override'
import getLanguagesServiceOverride from '@codingame/monaco-vscode-languages-service-override'
import getPreferencesServiceOverride from '@codingame/monaco-vscode-preferences-service-override'
import getOutputServiceOverride from '@codingame/monaco-vscode-output-service-override'
import getSearchServiceOverride from '@codingame/monaco-vscode-search-service-override'
import getMarkersServiceOverride from '@codingame/monaco-vscode-markers-service-override'
import getAccessibilityServiceOverride from '@codingame/monaco-vscode-accessibility-service-override'
import getStorageServiceOverride from '@codingame/monaco-vscode-storage-service-override'
import getLifecycleServiceOverride from '@codingame/monaco-vscode-lifecycle-service-override'
import getEnvironmentServiceOverride from '@codingame/monaco-vscode-environment-service-override'
import getHostServiceOverride from '@codingame/monaco-vscode-host-service-override'
import getSecretStorageServiceOverride from '@codingame/monaco-vscode-secret-storage-service-override'
import getWorkspaceTrustOverride from '@codingame/monaco-vscode-workspace-trust-service-override'
import getExplorerServiceOverride from '@codingame/monaco-vscode-explorer-service-override'
import getEditorServiceOverride from '@codingame/monaco-vscode-editor-service-override'
import getOutlineServiceOverride from '@codingame/monaco-vscode-outline-service-override'
import getTimelineServiceOverride from '@codingame/monaco-vscode-timeline-service-override'
import getBulkEditServiceOverride from '@codingame/monaco-vscode-bulk-edit-service-override'
import getRemoteAgentServiceOverride from '@codingame/monaco-vscode-remote-agent-service-override'
import getScmServiceOverride from '@codingame/monaco-vscode-scm-service-override'
import getStatusBarServiceOverride from '@codingame/monaco-vscode-view-status-bar-service-override'
import getTitleBarServiceOverride from '@codingame/monaco-vscode-view-title-bar-service-override'
import getBannerServiceOverride from '@codingame/monaco-vscode-view-banner-service-override'
import getQuickAccessServiceOverride from '@codingame/monaco-vscode-quickaccess-service-override'
import getWorkingCopyServiceOverride from '@codingame/monaco-vscode-working-copy-service-override'
import getWorkbenchServiceOverride from '@codingame/monaco-vscode-workbench-service-override'
import '@codingame/monaco-vscode-theme-defaults-default-extension'
import '@codingame/monaco-vscode-json-default-extension'
import '@codingame/monaco-vscode-markdown-basics-default-extension'
import { registerForgejoFS, workspaceUri, FORGEJO_SCHEME } from './forgejoFs'
import { registerForgejoSCM } from './forgejoScm'
import { registerForgejoBrand } from './forgejoChrome'

const container = document.getElementById('repo-webide') ?? document.body

// Config from the shell: /webide/?owner=..&repo=..&ref=..&commit=..&path=..
const params = new URLSearchParams(location.search)
const owner = params.get('owner') ?? ''
const repo = params.get('repo') ?? ''
const ref = params.get('ref') || 'main'
const openPath = params.get('path') ?? ''
const apiBase = `${params.get('base') ?? ''}/${owner}/${repo}`
const cfg = { apiBase, ref, baseCommit: params.get('commit') ?? '' }

const forgejoFs = registerForgejoFS(cfg)

// Standard editor UX: no auto-save, so edits show the unsaved dot until saved,
// then appear in Source Control (matches desktop VS Code / GitLab Web IDE).
await initUserConfiguration(
  JSON.stringify({
    'files.autoSave': 'off',
    'workbench.startupEditor': 'none',
    'scm.alwaysShowRepositories': true
  })
)

await initializeMonacoService(
  {
    ...getExtensionServiceOverride(),
    ...getLogServiceOverride(),
    ...getModelServiceOverride(),
    ...getNotificationsServiceOverride(),
    ...getDialogsServiceOverride(),
    ...getConfigurationServiceOverride(),
    ...getKeybindingsServiceOverride(),
    ...getTextmateServiceOverride(),
    ...getThemeServiceOverride(),
    ...getLanguagesServiceOverride(),
    ...getPreferencesServiceOverride(),
    ...getOutputServiceOverride(),
    ...getSearchServiceOverride(),
    ...getMarkersServiceOverride(),
    ...getAccessibilityServiceOverride(),
    // Hide two controls that do nothing in this embed, via VS Code's own
    // preference keys (no code path to break on upgrade): the Accounts activity
    // (no auth providers) and the Remote Host status entry (no remote agent).
    ...getStorageServiceOverride({
      fallbackOverride: {
        'workbench.activity.showAccounts': false,
        'workbench.statusbar.hidden': '["status.host"]'
      }
    }),
    ...getLifecycleServiceOverride(),
    ...getEnvironmentServiceOverride(),
    ...getHostServiceOverride(),
    ...getSecretStorageServiceOverride(),
    ...getWorkspaceTrustOverride(),
    // No custom cross-group open fallback — the workbench opens editors itself.
    ...getEditorServiceOverride(async () => undefined),
    ...getBulkEditServiceOverride(),
    ...getExplorerServiceOverride(),
    ...getOutlineServiceOverride(),
    ...getTimelineServiceOverride(),
    ...getRemoteAgentServiceOverride(),
    ...getScmServiceOverride(),
    ...getStatusBarServiceOverride(),
    ...getTitleBarServiceOverride(),
    ...getBannerServiceOverride(),
    ...getQuickAccessServiceOverride(),
    ...getWorkingCopyServiceOverride(),
    ...getWorkbenchServiceOverride()
  },
  container,
  {
    workspaceProvider: {
      trusted: true,
      workspace: { folderUri: workspaceUri() },
      open: async () => false
    },
    developmentOptions: { logLevel: LogLevel.Info }
  }
)

registerForgejoSCM(forgejoFs, cfg)
registerForgejoBrand()

// Show the file explorer, and open the requested file if any.
await vscode.commands.executeCommand('workbench.view.explorer')
if (openPath !== '') {
  try {
    await vscode.window.showTextDocument(
      vscode.Uri.from({ scheme: FORGEJO_SCHEME, path: '/' + openPath })
    )
  } catch {
    /* ignore: file may be gone or binary */
  }
}
