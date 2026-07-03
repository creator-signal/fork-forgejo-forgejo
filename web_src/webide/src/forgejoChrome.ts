// Thin Forgejo branding for the workbench, kept upgrade-friendly. The badge uses
// the stable public status-bar API; the only thing that API can't do — an exact
// brand background (it allows only error/warning colors, see vscode.d.ts
// StatusBarItem.backgroundColor) — is one scoped CSS rule that degrades to the
// default status-bar styling if VS Code ever renames the DOM.
import * as vscode from 'vscode'

const BRAND_ID = 'forgejoBrand'
const BRAND_COLOR = '#B13E02'

// A non-interactive "Forgejo" badge pinned to the far bottom-left, like GitLab's
// Web IDE. No command is set, so it is not clickable (no link).
export function registerForgejoBrand(): void {
  const badge = vscode.window.createStatusBarItem(BRAND_ID, vscode.StatusBarAlignment.Left, Number.MAX_SAFE_INTEGER)
  badge.text = 'Forgejo'
  badge.tooltip = 'Forgejo'
  badge.show()

  const style = document.createElement('style')
  style.textContent =
    `.monaco-workbench .part.statusbar .statusbar-item[id$="${BRAND_ID}"]{` +
    `background-color:${BRAND_COLOR};color:#fff;font-weight:600}`
  document.head.appendChild(style)
}
