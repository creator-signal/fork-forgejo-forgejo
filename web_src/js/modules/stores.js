import {reactive} from 'vue';

let diffTreeStoreReactive;
export function diffTreeStore() {
  if (!diffTreeStoreReactive) {
    diffTreeStoreReactive = reactive(window.config.pageData.diffFileInfo);
    window.config.pageData.diffFileInfo = diffTreeStoreReactive;
  }
  return diffTreeStoreReactive;
}

const FILTER_STORAGE_KEY = 'diff_file_filters';

function loadFiltersFromStorage() {
  try {
    const stored = localStorage.getItem(FILTER_STORAGE_KEY);
    if (stored) return JSON.parse(stored);
  } catch { /* ignore */ }
  return {};
}

let diffFilterStoreReactive;
export function diffFilterStore() {
  if (!diffFilterStoreReactive) {
    const saved = loadFiltersFromStorage();
    diffFilterStoreReactive = reactive({
      // null = show all, true = viewed only, false = unviewed only
      viewedFilter: saved.viewedFilter ?? null,
      // Set of active diff type filters (1=added, 2=modified, 3=deleted, 4=renamed, 5=copied)
      // empty set = show all
      typeFilters: new Set(saved.typeFilters ?? []),
      // empty string = show all, otherwise filter by extension (e.g. ".go")
      extensionFilter: saved.extensionFilter ?? '',
    });
  }
  return diffFilterStoreReactive;
}

export function saveFiltersToStorage() {
  const store = diffFilterStore();
  try {
    localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify({
      viewedFilter: store.viewedFilter,
      typeFilters: Array.from(store.typeFilters),
      extensionFilter: store.extensionFilter,
    }));
  } catch { /* ignore */ }
}

export function fileMatchesFilters(file) {
  const store = diffFilterStore();

  // Check viewed filter
  if (store.viewedFilter !== null) {
    if (store.viewedFilter && !file.IsViewed) return false;
    if (!store.viewedFilter && file.IsViewed) return false;
  }

  // Check type filter
  if (store.typeFilters.size > 0 && !store.typeFilters.has(file.Type)) return false;

  // Check extension filter
  if (store.extensionFilter) {
    const fileName = file.Name || '';
    const dotIdx = fileName.lastIndexOf('.');
    const ext = dotIdx !== -1 ? fileName.substring(dotIdx) : '';
    if (ext !== store.extensionFilter) return false;
  }

  return true;
}

export function clearAllFilters() {
  const store = diffFilterStore();
  store.viewedFilter = null;
  store.typeFilters.clear();
  store.extensionFilter = '';
  saveFiltersToStorage();
}

export function hasActiveFilters() {
  const store = diffFilterStore();
  return store.viewedFilter !== null || store.typeFilters.size > 0 || store.extensionFilter !== '';
}
