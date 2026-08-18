/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** PocketBase base URL used during local development (e.g. http://127.0.0.1:8090).
   *  When empty the SDK falls back to the same origin (embedded production build). */
  readonly VITE_PB_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
