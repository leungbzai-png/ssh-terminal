export {};

declare global {
  interface Window {
    runtime: {
      EventsOn: (event: string, cb: (...args: any[]) => void) => void;
      EventsOff: (event: string, ...handlers: any[]) => void;
      EventsEmit: (event: string, ...data: any[]) => void;
      WindowSetDarkTheme: () => void;
      WindowSetLightTheme: () => void;
      WindowSetSystemDefaultTheme: () => void;
    };
    go: {
      main: {
        App: {
          AppInfo: () => Promise<Record<string, string>>;

          GetSettings: () => Promise<AppSettings>;
          SaveSettings: (s: AppSettings) => Promise<void>;

          ListHosts: () => Promise<HostRecord[]>;
          UpsertHost: (h: HostRecord) => Promise<HostRecord>;
          DeleteHost: (id: string) => Promise<void>;

          OpenSession: (sessionID: string, hostID: string, cols: number, rows: number) => Promise<void>;
          SshOpenQuick: (sessionID: string, params: QuickConnectParams, cols: number, rows: number) => Promise<void>;
          WriteSession: (sessionID: string, dataB64: string) => Promise<void>;
          ResizeSession: (sessionID: string, cols: number, rows: number) => Promise<void>;
          CloseSession: (sessionID: string) => Promise<void>;
          AnswerHostKey: (fingerprint: string, accept: boolean) => Promise<void>;
          ActiveSessionCount: () => Promise<number>;
          ConfirmQuit: () => Promise<void>;

          SftpList: (sessionID: string, dir: string) => Promise<FileEntry[]>;
          SftpCwd: (sessionID: string) => Promise<string>;
          SftpDownload: (sessionID: string, remote: string, local: string) => Promise<void>;
          SftpUpload: (sessionID: string, local: string, remote: string) => Promise<void>;
          SftpUploadPaths: (sessionID: string, localPaths: string[], remoteDir: string) => Promise<void>;
          SftpUploadTracked: (sessionID: string, localPaths: string[], remoteDir: string) => Promise<void>;
          SftpDownloadTracked: (sessionID: string, remote: string, local: string) => Promise<void>;
          SftpDownloadPathsTracked: (sessionID: string, remotePaths: string[], localDir: string) => Promise<void>;
          SftpExists: (sessionID: string, remotePath: string) => Promise<boolean>;
          SftpPreviewText: (sessionID: string, remote: string) => Promise<TextPreview>;
          SftpDelete: (sessionID: string, remote: string) => Promise<void>;
          SftpDeleteRecursive: (sessionID: string, remote: string) => Promise<void>;
          SftpMkdir: (sessionID: string, remote: string) => Promise<void>;
          SftpRename: (sessionID: string, oldPath: string, newPath: string) => Promise<void>;

          // Local filesystem browse (v1.1.0, SFTP two-pane local pane). Reuses FileEntry.
          LocalList: (dir: string) => Promise<FileEntry[]>;
          LocalHome: () => Promise<string>;
          LocalRoots: () => Promise<string[]>;
          LocalParent: (dir: string) => Promise<[string, boolean]>;
          LocalExists: (path: string) => Promise<boolean>;

          ListKeys: () => Promise<ManagedKey[]>;
          GenerateKey: (name: string, comment: string, keyType: string, rsaBits: number, passphrase: string) => Promise<ManagedKey>;
          ImportPrivateKey: (name: string, comment: string, keyPath: string, passphrase: string) => Promise<ManagedKey>;
          DeleteKey: (id: string) => Promise<void>;
          GetPublicKey: (id: string) => Promise<string>;
          DeployPublicKeyToHost: (hostId: string, keyId: string) => Promise<void>;

          ExportHosts: () => Promise<string>;
          PreviewHostsImport: () => Promise<HostsImportPreview>;
          ImportHosts: (entries: SafeHost[], overwrite: boolean) => Promise<HostsImportResult>;

          GetOpenTabs: () => Promise<OpenTabRef[]>;
          SaveOpenTabs: (tabs: OpenTabRef[]) => Promise<void>;

          ListBookmarks: (hostId: string) => Promise<Bookmark[]>;
          AddBookmark: (hostId: string, name: string, remotePath: string) => Promise<Bookmark>;
          DeleteBookmark: (id: string) => Promise<void>;

          PickFileToUpload: () => Promise<string>;
          PickFilesToUpload: () => Promise<string[]>;
          PickSaveLocation: (suggested: string) => Promise<string>;
          PickPrivateKey: () => Promise<string>;

          DefaultSshConfigPath: () => Promise<string>;
          PickSshConfig: () => Promise<string>;
          PreviewSshConfig: (path: string) => Promise<SshConfigPreviewEntry[]>;
          ImportSshConfig: (entries: SshConfigEntry[]) => Promise<SshConfigImportResult>;
        };
      };
    };
  }
}

export interface AppSettings {
  theme: "light" | "dark" | "system";
  fontFamily: string;
  fontSize: number;
  cursorStyle: "block" | "bar" | "underline";
  cursorBlink: boolean;
  scrollBack: number;
  confirmCloseWithActiveSessions: boolean;
  showCommandBar: boolean;
  connectTimeoutSec: number;
  keepAliveEnabled: boolean;
  keepAliveIntervalSec: number;
}

export interface QuickConnectParams {
  address: string;
  port: number;
  user: string;
  authType: "password" | "key";
  password?: string;
  keyPath?: string;
  passphrase?: string;
}

export interface SshConfigEntry {
  alias: string;
  hostName: string;
  user: string;
  port: number;
  identityFile: string;
  warnings: string[];
}

export interface SshConfigPreviewEntry extends SshConfigEntry {
  identityExists: boolean;
  duplicate: boolean;
}

export interface SshConfigImportResult {
  imported: number;
  skipped: number;
  names: string[];
}

// --- Advanced SSH (v0.8.0): all fields are NON-SECRET ---

export interface ProxyJumpConfig {
  mode: "savedHost" | "manual";
  jumpHostId?: string; // when mode === savedHost
  address?: string; // when mode === manual (key-only, no password)
  port?: number;
  user?: string;
  keyPath?: string;
}

export interface PortForward {
  name?: string;
  localHost?: string; // default 127.0.0.1
  localPort?: number;
  remoteHost?: string;
  remotePort?: number;
  enabled: boolean;
}

export interface DynamicForward {
  name?: string;
  localHost?: string; // default 127.0.0.1
  localPort?: number;
  enabled: boolean;
}

export interface AutoReconnectConfig {
  enabled: boolean;
  maxAttempts: number; // 0..10
  delaySeconds: number; // 1..60
}

export interface AdvancedSSH {
  proxyJump?: ProxyJumpConfig;
  localForwards?: PortForward[];
  remoteForwards?: PortForward[];
  dynamicForwards?: DynamicForward[];
  autoReconnect?: AutoReconnectConfig;
}

// Tunnel status event payload (ssh:tunnel:<sessionID>). Non-secret.
export interface TunnelStatus {
  kind: "local" | "remote" | "dynamic";
  name: string;
  listen: string;
  ok: boolean;
  err?: string;
}

// Safe host export/import: non-secret host metadata only (no password,
// passphrase, or private-key material).
export interface SafeHost {
  name: string;
  address: string;
  port: number;
  user: string;
  authType: "password" | "key" | "managedKey";
  keyPath?: string;
  managedKeyId?: string;
  group?: string;
  note?: string;
  advanced?: AdvancedSSH;
}

export interface HostImportPreviewEntry extends SafeHost {
  duplicate: boolean;
  keyExists: boolean;
}

export interface HostsImportPreview {
  path: string;
  hosts: HostImportPreviewEntry[];
}

export interface HostsImportResult {
  imported: number;
  skipped: number;
  overwritten: number;
}

// Non-secret restorable tab intent (host reference only).
export interface OpenTabRef {
  hostId: string;
  hostName: string;
}

// Non-secret remote-path bookmark for a host.
export interface Bookmark {
  id: string;
  hostId: string;
  name: string;
  path: string;
  createdAt: number;
}

export interface HostRecord {
  id: string;
  name: string;
  address: string;
  port: number;
  user: string;
  authType: "password" | "key" | "managedKey";
  password?: string;
  keyPath?: string;
  passphrase?: string;
  managedKeyId?: string;
  group?: string;
  note?: string;
  advanced?: AdvancedSSH;
  updatedAt?: number;
}

export interface TextPreview {
  content: string;
  size: number;
  tooLarge: boolean;
  binary: boolean;
}

export interface FileEntry {
  name: string;
  path: string;
  size: number;
  mode: string;
  modTime: number;
  isDir: boolean;
  isLink: boolean;
}

export interface ManagedKey {
  id: string;
  name: string;
  type: "ed25519" | "rsa";
  comment: string;
  fingerprint: string;
  publicKey: string;
  hasPassword: boolean;
  createdAt: number;
}
