/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** 'stable' on main, 'beta' on develop. Baked in at SSG build time. */
  readonly VITE_DOCS_CHANNEL: 'stable' | 'beta' | undefined;
  /** Base path of the stable docs site (baked in on develop builds). */
  readonly VITE_STABLE_URL: string | undefined;
  /** Base path of the beta docs site (baked in on main builds). */
  readonly VITE_BETA_URL: string | undefined;
  /** Vite base path, set per channel: /nexus-orchestrator/ or /nexus-orchestrator/beta/ */
  readonly VITE_BASE_PATH: string | undefined;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
