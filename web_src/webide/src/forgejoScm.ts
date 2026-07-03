// ForgejoSCM: a Source Control panel over ForgejoFS's pending edits, with a
// visible Commit action button. The button collates saved changes into one
// POST /_ide/commit. The action button needs the scmActionButton API proposal,
// so this is registered as a system extension.
import { ExtensionHostKind, registerExtension } from '@codingame/monaco-vscode-api/extensions'
import { ForgejoFS, ForgejoConfig, FORGEJO_SCHEME, workspaceUri } from './forgejoFs'

export function registerForgejoSCM(fs: ForgejoFS, cfg: ForgejoConfig): void {
  const { getApi } = registerExtension(
    {
      name: 'forgejo-scm',
      publisher: 'forgejo',
      engines: { vscode: '*' },
      version: '1.0.0',
      enabledApiProposals: ['scmActionButton']
    },
    ExtensionHostKind.LocalProcess,
    { system: true }
  )

  void getApi().then((vscode) => {
    const scm = vscode.scm.createSourceControl('forgejo', 'Forgejo', workspaceUri())
    scm.inputBox.placeholder = `Message (commit to ${cfg.ref})`
    scm.acceptInputCommand = { command: 'forgejo.commit', title: 'Commit' }
    const group = scm.createResourceGroup('changes', 'Changes')

    const label = (s: 'A' | 'M' | 'D') => (s === 'A' ? 'Added' : s === 'D' ? 'Deleted' : 'Modified')
    const refresh = () => {
      const changes = fs.changes()
      group.resourceStates = changes.map((c) => ({
        resourceUri: vscode.Uri.from({ scheme: FORGEJO_SCHEME, path: '/' + c.path }),
        decorations: { tooltip: label(c.status), strikeThrough: c.status === 'D', letter: c.status }
      }))
      scm.count = changes.length
      scm.actionButton = {
        command: {
          command: 'forgejo.commit',
          title: changes.length > 0 ? `$(check) Commit (${changes.length})` : '$(check) Commit'
        },
        enabled: changes.length > 0
      }
    }

    vscode.commands.registerCommand('forgejo.commit', async () => {
      const message = scm.inputBox.value.trim()
      if (message === '') {
        void vscode.window.showWarningMessage('Enter a commit message')
        return
      }
      if (!fs.hasPending()) {
        void vscode.window.showInformationMessage('No changes to commit')
        return
      }
      try {
        await fs.commit(message)
        scm.inputBox.value = ''
        void vscode.window.showInformationMessage(`Committed to ${cfg.ref}`)
      } catch (e) {
        void vscode.window.showErrorMessage(`Commit failed: ${(e as Error).message}`)
      }
      refresh()
    })

    fs.onDidChangePending(refresh)
    refresh()
  })
}
