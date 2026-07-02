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
          SftpDelete: (sessionID: string, remote: string) => Promise<void>;
          SftpDeleteRecursive: (sessionID: string, remote: string) => Promise<void>;
          SftpMkdir: (sessionID: string, remote: string) => Promise<void>;
          SftpRename: (sessionID: string, oldPath: string, newPath: string) => Promise<void>;

          ListKeys: () => Promise<ManagedKey[]>;
          GenerateKey: (name: string, comment: string, keyType: string, rsaBits: number, passphrase: string) => Promise<ManagedKey>;
          DeleteKey: (id: string) => Promise<void>;
          GetPublicKey: (id: string) => Promise<string>;
          DeployPublicKeyToHost: (hostId: string, keyId: string) => Promise<void>;

          PickFileToUpload: () => Promise<string>;
          PickFilesToUpload: () => Promise<string[]>;
          PickSaveLocation: (suggested: string) => Promise<string>;
          PickPrivateKey: () => Promise<string>;
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
  updatedAt?: number;
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
