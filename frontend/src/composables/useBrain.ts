import { ref } from 'vue';
import type { BrainStatus, ContextSection, ProjectKnowledge } from '../types/domain';
import {
  getBrainStatus,
  initProject,
  ingestKnowledge,
  ingestContent,
  listKnowledge,
  deleteKnowledge,
  searchKnowledge,
  getFileMap,
  getProjectContext,
  getFocusedContext,
} from '../types/wails';

/** Largest document the daemon accepts for ingestion (1 MiB). */
export const MAX_UPLOAD_BYTES = 1024 * 1024;

/** State and actions for the Context Brain: status, entries, ingestion, search and context. */
export function useBrain(projectPath: string) {
  const status = ref<BrainStatus | null>(null);
  const entries = ref<ProjectKnowledge[]>([]);
  const fileMap = ref<string[]>([]);
  const searchResults = ref<ContextSection[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchStatus() {
    loading.value = true;
    error.value = null;
    try {
      status.value = await getBrainStatus(projectPath);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function init(claudeMDPath = '') {
    loading.value = true;
    error.value = null;
    try {
      status.value = await initProject(projectPath, claudeMDPath);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function ingest(filePath: string): Promise<number> {
    error.value = null;
    try {
      const count = await ingestKnowledge(projectPath, filePath);
      await fetchStatus();
      return count;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      return 0;
    }
  }

  /**
   * Ingests a file the user picked in the browser. Its content is sent to the
   * daemon because a browser File has no path the daemon could read.
   */
  async function ingestUpload(file: {
    name: string;
    size: number;
    text(): Promise<string>;
  }): Promise<number> {
    error.value = null;
    if (file.size > MAX_UPLOAD_BYTES) {
      error.value = `${file.name} is larger than 1 MiB`;
      return 0;
    }
    try {
      const count = await ingestContent(projectPath, file.name, await file.text());
      await fetchStatus();
      return count;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      return 0;
    }
  }

  async function listEntries(kind = '') {
    loading.value = true;
    error.value = null;
    try {
      entries.value = await listKnowledge(projectPath, kind);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function deleteEntry(id: string) {
    error.value = null;
    try {
      await deleteKnowledge(id);
      entries.value = entries.value.filter((e) => e.id !== id);
      await fetchStatus();
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    }
  }

  async function search(query: string, limit = 10) {
    loading.value = true;
    error.value = null;
    try {
      searchResults.value = await searchKnowledge(projectPath, query, limit);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function fetchFileMap(focusArea = '') {
    loading.value = true;
    error.value = null;
    try {
      fileMap.value = await getFileMap(projectPath, focusArea);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function getContext(maxTokens = 800) {
    loading.value = true;
    error.value = null;
    try {
      return await getProjectContext(projectPath, maxTokens);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      return null;
    } finally {
      loading.value = false;
    }
  }

  async function getFocused(question: string, maxTokens = 400) {
    loading.value = true;
    error.value = null;
    try {
      return await getFocusedContext(projectPath, question, maxTokens);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      return null;
    } finally {
      loading.value = false;
    }
  }

  return {
    status,
    entries,
    fileMap,
    searchResults,
    loading,
    error,
    fetchStatus,
    init,
    ingest,
    ingestUpload,
    listEntries,
    deleteEntry,
    search,
    fetchFileMap,
    getContext,
    getFocusedContext: getFocused,
  };
}
