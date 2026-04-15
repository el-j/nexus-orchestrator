import { ref } from 'vue';
import type { BrainStatus, ContextSection, ProjectKnowledge } from '../types/domain';
import {
  getBrainStatus,
  initProject,
  ingestKnowledge,
  listKnowledge,
  deleteKnowledge,
  searchKnowledge,
  getFileMap,
} from '../types/wails';

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
    listEntries,
    deleteEntry,
    search,
    fetchFileMap,
  };
}
