// ForgejoFS: a VS Code file-system provider backed by the session-authed
// /_ide BFF. Reads are lazy (tree/blob); edits buffer into an in-memory overlay
// and are committed via ForgejoSCM -> POST /_ide/commit.
import * as vscode from 'vscode'
import {
  FileType,
  FileSystemProviderCapabilities,
  FileSystemProviderError,
  FileSystemProviderErrorCode,
  registerCustomProvider
} from '@codingame/monaco-vscode-files-service-override'

export interface ForgejoConfig {
  apiBase: string // repo link, e.g. /owner/repo
  ref: string
  baseCommit: string // commit of ref at load, sent as last_commit_id
}

export interface ForgejoChange {
  path: string
  status: 'A' | 'M' | 'D'
}

interface TreeEntry {
  name: string
  path: string
  type: 'file' | 'dir' | 'symlink' | 'submodule'
  size: number
}

interface BlobResponse {
  path: string
  size: number
  encoding?: string
  content?: string
  editable: boolean
}

const SCHEME = 'forgejo'
const enc = new TextEncoder()

function repoPath(uri: { path: string }): string {
  return uri.path.replace(/^\/+/, '')
}

export class ForgejoFS {
  readonly capabilities =
    FileSystemProviderCapabilities.FileReadWrite | FileSystemProviderCapabilities.PathCaseSensitive

  private readonly onDidChangeCapabilitiesEmitter = new vscode.EventEmitter<void>()
  readonly onDidChangeCapabilities = this.onDidChangeCapabilitiesEmitter.event
  private readonly onDidChangeFileEmitter = new vscode.EventEmitter<any>()
  readonly onDidChangeFile = this.onDidChangeFileEmitter.event
  // fired whenever the set of uncommitted changes changes (for the SCM view)
  private readonly onDidChangePendingEmitter = new vscode.EventEmitter<void>()
  readonly onDidChangePending = this.onDidChangePendingEmitter.event

  // dir path -> entries (cache for stat/readdir)
  private readonly dirCache = new Map<string, TreeEntry[]>()
  // paths known to exist on the server (to pick create vs update on commit)
  private readonly knownExisting = new Set<string>()
  // pending edits (path -> content) and deletions, applied by SCM on commit
  readonly pendingWrites = new Map<string, Uint8Array>()
  readonly pendingDeletes = new Set<string>()

  constructor(private readonly cfg: ForgejoConfig) {}

  private url(kind: 'tree' | 'blob', p: string, extra = ''): string {
    return `${this.cfg.apiBase}/_ide/${kind}?ref=${encodeURIComponent(this.cfg.ref)}&path=${encodeURIComponent(p)}${extra}`
  }

  private async listDir(p: string): Promise<TreeEntry[]> {
    const cached = this.dirCache.get(p)
    if (cached != null) return cached
    const res = await fetch(this.url('tree', p), { credentials: 'same-origin' })
    if (res.status === 404) throw FileSystemProviderError.create('Not found', FileSystemProviderErrorCode.FileNotFound)
    if (!res.ok) throw FileSystemProviderError.create(`tree ${res.status}`, FileSystemProviderErrorCode.Unavailable)
    const entries = (await res.json()) as TreeEntry[]
    this.dirCache.set(p, entries)
    for (const e of entries) {
      if (e.type === 'file' || e.type === 'symlink') this.knownExisting.add(e.path)
    }
    return entries
  }

  watch(): vscode.Disposable {
    return { dispose() {} }
  }

  async stat(resource: { path: string }): Promise<{ type: FileType; mtime: number; ctime: number; size: number }> {
    const p = repoPath(resource)
    if (p === '') return { type: FileType.Directory, mtime: 0, ctime: 0, size: 0 }
    if (this.pendingWrites.has(p)) {
      return { type: FileType.File, mtime: Date.now(), ctime: 0, size: this.pendingWrites.get(p)!.byteLength }
    }
    const parent = p.includes('/') ? p.slice(0, p.lastIndexOf('/')) : ''
    const name = p.includes('/') ? p.slice(p.lastIndexOf('/') + 1) : p
    const entry = (await this.listDir(parent)).find((e) => e.name === name)
    if (entry == null) {
      throw FileSystemProviderError.create('Not found', FileSystemProviderErrorCode.FileNotFound)
    }
    return { type: toFileType(entry.type), mtime: 0, ctime: 0, size: entry.size }
  }

  async readdir(resource: { path: string }): Promise<[string, FileType][]> {
    const entries = await this.listDir(repoPath(resource))
    return entries.map((e) => [e.name, toFileType(e.type)])
  }

  async readFile(resource: { path: string }): Promise<Uint8Array> {
    const p = repoPath(resource)
    if (this.pendingWrites.has(p)) return this.pendingWrites.get(p)!
    const res = await fetch(this.url('blob', p), { credentials: 'same-origin' })
    if (res.status === 404) throw FileSystemProviderError.create('Not found', FileSystemProviderErrorCode.FileNotFound)
    if (!res.ok) throw FileSystemProviderError.create(`blob ${res.status}`, FileSystemProviderErrorCode.Unavailable)
    this.knownExisting.add(p)
    const ct = res.headers.get('content-type') ?? ''
    if (ct.includes('application/json')) {
      const blob = (await res.json()) as BlobResponse
      if (blob.editable && blob.encoding === 'base64' && blob.content != null) {
        return base64ToBytes(blob.content)
      }
    }
    // Oversized/binary: fetch raw bytes.
    const raw = await fetch(this.url('blob', p, '&download=1'), { credentials: 'same-origin' })
    return new Uint8Array(await raw.arrayBuffer())
  }

  async writeFile(resource: { path: string }, content: Uint8Array): Promise<void> {
    const p = repoPath(resource)
    this.pendingWrites.set(p, content)
    this.pendingDeletes.delete(p)
    this.onDidChangeFileEmitter.fire([{ type: 1 /* Changed */, resource }])
    this.onDidChangePendingEmitter.fire()
  }

  async delete(resource: { path: string }): Promise<void> {
    const p = repoPath(resource)
    this.pendingWrites.delete(p)
    if (this.knownExisting.has(p)) this.pendingDeletes.add(p)
    this.dirCache.clear()
    this.onDidChangeFileEmitter.fire([{ type: 2 /* Deleted */, resource }])
    this.onDidChangePendingEmitter.fire()
  }

  async rename(from: { path: string }, to: { path: string }): Promise<void> {
    const content = await this.readFile(from)
    await this.writeFile(to, content)
    await this.delete(from)
  }

  async mkdir(): Promise<void> {
    // Git has no empty directories; created implicitly when a file is committed.
  }

  hasPending(): boolean {
    return this.pendingWrites.size > 0 || this.pendingDeletes.size > 0
  }

  changes(): ForgejoChange[] {
    const out: ForgejoChange[] = []
    for (const p of this.pendingWrites.keys()) out.push({ path: p, status: this.knownExisting.has(p) ? 'M' : 'A' })
    for (const p of this.pendingDeletes) out.push({ path: p, status: 'D' })
    return out.sort((a, b) => a.path.localeCompare(b.path))
  }

  // Commit all pending changes in one commit via the BFF, then clear them.
  async commit(message: string): Promise<void> {
    const files: Array<Record<string, string>> = []
    for (const [p, content] of this.pendingWrites) {
      files.push({ operation: this.knownExisting.has(p) ? 'update' : 'create', path: p, content: bytesToBase64(content) })
    }
    for (const p of this.pendingDeletes) files.push({ operation: 'delete', path: p })
    if (files.length === 0) throw new Error('No changes to commit')

    const res = await fetch(`${this.cfg.apiBase}/_ide/commit`, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ branch: this.cfg.ref, message, last_commit_id: this.cfg.baseCommit, files })
    })
    if (!res.ok) {
      let msg = `commit failed (${res.status})`
      try {
        const j = (await res.json()) as { error?: string }
        if (j.error != null) msg = j.error
      } catch {
        /* ignore */
      }
      throw new Error(msg)
    }

    for (const p of this.pendingWrites.keys()) this.knownExisting.add(p)
    for (const p of this.pendingDeletes) this.knownExisting.delete(p)
    this.pendingWrites.clear()
    this.pendingDeletes.clear()
    this.dirCache.clear()
    this.onDidChangePendingEmitter.fire()
  }
}

function bytesToBase64(bytes: Uint8Array): string {
  let bin = ''
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i])
  return btoa(bin)
}

function toFileType(t: TreeEntry['type']): FileType {
  switch (t) {
    case 'dir':
      return FileType.Directory
    case 'symlink':
      return FileType.SymbolicLink
    default:
      return FileType.File
  }
}

function base64ToBytes(b64: string): Uint8Array {
  const bin = atob(b64)
  const out = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i)
  return out
}

export function registerForgejoFS(cfg: ForgejoConfig): ForgejoFS {
  const fs = new ForgejoFS(cfg)
  registerCustomProvider(SCHEME, fs as any)
  return fs
}

export const FORGEJO_SCHEME = SCHEME
export const workspaceUri = (): vscode.Uri => vscode.Uri.from({ scheme: SCHEME, path: '/' })
export { enc }
