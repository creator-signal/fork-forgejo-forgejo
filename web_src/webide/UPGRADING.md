# Upgrading the Web IDE (monaco-vscode-api)

Short, factual notes for a future upgrade. Written for both humans and LLMs — keep it current.

## Current pin (verified)

- `@codingame/monaco-vscode-*` = **34.1.3** (all 42 packages, lockstep).
- 34.1.3 is built from **VS Code 1.124.2** (string present in the bundled sources).
- `monaco-editor` and `vscode` are npm aliases to `@codingame/monaco-vscode-editor-api` / `-extension-api` at the same version — bump them too.
- Build: `vite` 8 (rolldown). Type-check: `typescript` 5.9.2.

When editing this file after an upgrade, re-verify the version/VS Code mapping from the source of truth:
- Releases: https://github.com/CodinGame/monaco-vscode-api/releases
- Wiki: https://github.com/CodinGame/monaco-vscode-api/wiki

## The one hard rule

**All `@codingame/monaco-vscode-*` packages must be the exact same version.** They share internal
singletons; a mismatch fails at runtime. Bump every one of them (see `package.json` `dependencies`)
plus the two npm aliases (`monaco-editor`, `vscode`) to the identical new version.

## Upgrade steps

1. Read the release notes for the target version (link above) — look for renamed/split service-override
   packages, changed override options, and proposed-API changes.
2. Set every `@codingame/monaco-vscode-*` (incl. the `monaco-editor` / `vscode` aliases) to the same new version, `npm install`.
3. `npm run typecheck` — **the build (esbuild) does NOT typecheck**, so this is what surfaces renamed
   imports/options. Must be clean (exit 0).
4. `npm run build`.
5. Smoke test (`docker compose up -d` from the repo root, then `/{owner}/{repo}/ide`):
   workbench loads → browse tree → open a file (unsaved dot on edit) → Source Control shows the change →
   Commit succeeds → Forgejo badge sits at the bottom-left corner → no Accounts icon / no Remote `><` button.

## Where it can break — ranked, with exact spots

| Risk | Location | Note |
|---|---|---|
| **Proposed API** (only one) | `src/forgejoScm.ts` — `enabledApiProposals: ['scmActionButton']` | Proposed APIs can change shape or graduate to stable between VS Code releases. If the Commit button/action disappears, re-check this proposal. |
| **Service-override options / worker subpaths** | `src/main.ts` — the ~42 `get*ServiceOverride()` imports, the `?worker` imports, `getStorageServiceOverride({ fallbackOverride })` | Packages/options occasionally get renamed or split. `npm run typecheck` catches most; worker path changes surface at runtime ("Unregistered worker"). |
| **Hidden-control preference keys** | `src/main.ts` — `workbench.activity.showAccounts`, `workbench.statusbar.hidden` (`["status.host"]`) | VS Code's own keys (stable). If Accounts / the Remote entry reappear, the key was renamed — update the value. Fails safe (control just reappears). |
| **Brand badge color (CSS)** | `src/forgejoChrome.ts` — rule on `.statusbar-item[id$="forgejoBrand"]` | The API can't set an arbitrary background (only error/warning, see `vscode.d.ts` `StatusBarItem.backgroundColor`), so the exact `#B13E02` is one CSS rule. If the badge loses its color, the status-bar DOM class changed — update the selector. Degrades gracefully (badge still shows). |
| **Stable extension API** | `createStatusBarItem`, `scm.createSourceControl`, `registerFileSystemProvider`, `commands`, `Uri`, `window.show*` | Covered by VS Code's API-stability guarantee. Rarely breaks. |

## Not affected by a monaco-vscode-api upgrade

The backend BFF has **zero** VS Code coupling and versions with Forgejo, not VS Code:
`routers/web/repo/webide.go`, the `/_ide/*` routes in `routers/web/web.go`, the `TypeWebIDE` unit,
and the `templates/repo/ide/` shell.
