<script>
import {diffTreeStore, diffFilterStore, saveFiltersToStorage, clearAllFilters, hasActiveFilters, fileMatchesFilters} from '../modules/stores.js';
import {SvgIcon} from '../svg.js';

export default {
  components: {SvgIcon},
  data: () => ({
    treeStore: diffTreeStore(),
    filterStore: diffFilterStore(),
    showExtDropdown: false,
  }),
  computed: {
    extensions() {
      const extSet = new Set();
      for (const file of this.treeStore.files) {
        const name = file.Name || '';
        const dotIdx = name.lastIndexOf('.');
        if (dotIdx !== -1) {
          extSet.add(name.substring(dotIdx));
        }
      }
      return Array.from(extSet).sort();
    },
    hasFilters() {
      return hasActiveFilters();
    },
    filteredCount() {
      let count = 0;
      for (const file of this.treeStore.files) {
        if (fileMatchesFilters(file)) count++;
      }
      return count;
    },
    totalCount() {
      return this.treeStore.files.length;
    },
  },
  mounted() {
    document.addEventListener('click', this.closeExtDropdown);
  },
  unmounted() {
    document.removeEventListener('click', this.closeExtDropdown);
  },
  methods: {
    setViewedFilter(value) {
      this.filterStore.viewedFilter = this.filterStore.viewedFilter === value ? null : value;
      saveFiltersToStorage();
      this.applyFilters();
    },
    toggleTypeFilter(type) {
      if (this.filterStore.typeFilters.has(type)) {
        this.filterStore.typeFilters.delete(type);
      } else {
        this.filterStore.typeFilters.add(type);
      }
      saveFiltersToStorage();
      this.applyFilters();
    },
    setExtensionFilter(ext) {
      this.filterStore.extensionFilter = this.filterStore.extensionFilter === ext ? '' : ext;
      this.showExtDropdown = false;
      saveFiltersToStorage();
      this.applyFilters();
    },
    clearFilters() {
      clearAllFilters();
      this.applyFilters();
    },
    toggleExtDropdown(e) {
      e.stopPropagation();
      this.showExtDropdown = !this.showExtDropdown;
    },
    closeExtDropdown() {
      this.showExtDropdown = false;
    },
    applyFilters() {
      // Show/hide diff file boxes based on filters
      for (const file of this.treeStore.files) {
        const el = document.getElementById(`diff-${file.NameHash}`);
        if (!el) continue;
        el.style.display = fileMatchesFilters(file) ? '' : 'none';
      }
    },
    isTypeActive(type) {
      return this.filterStore.typeFilters.has(type);
    },
  },
};
</script>
<template>
  <div v-if="treeStore.fileTreeIsVisible" class="diff-file-filter">
    <!-- Viewed/Unviewed filter -->
    <div class="filter-group">
      <button
        class="filter-chip"
        :class="{active: filterStore.viewedFilter === true}"
        @click="setViewedFilter(true)"
      >
        <SvgIcon name="octicon-check" :size="12"/>
        Viewed
      </button>
      <button
        class="filter-chip"
        :class="{active: filterStore.viewedFilter === false}"
        @click="setViewedFilter(false)"
      >
        <SvgIcon name="octicon-eye" :size="12"/>
        Unviewed
      </button>
    </div>

    <!-- Type filters -->
    <div class="filter-group">
      <button
        class="filter-chip filter-added"
        :class="{active: isTypeActive(1)}"
        @click="toggleTypeFilter(1)"
      >
        Added
      </button>
      <button
        class="filter-chip filter-modified"
        :class="{active: isTypeActive(2)}"
        @click="toggleTypeFilter(2)"
      >
        Modified
      </button>
      <button
        class="filter-chip filter-deleted"
        :class="{active: isTypeActive(3)}"
        @click="toggleTypeFilter(3)"
      >
        Deleted
      </button>
      <button
        class="filter-chip filter-renamed"
        :class="{active: isTypeActive(4)}"
        @click="toggleTypeFilter(4)"
      >
        Renamed
      </button>
    </div>

    <!-- Extension filter -->
    <div v-if="extensions.length > 1" class="filter-group">
      <div class="ext-dropdown-container">
        <button class="filter-chip" :class="{active: filterStore.extensionFilter}" @click="toggleExtDropdown">
          <SvgIcon name="octicon-file" :size="12"/>
          {{ filterStore.extensionFilter || 'Extension' }}
          <SvgIcon name="octicon-chevron-down" :size="12"/>
        </button>
        <div v-if="showExtDropdown" class="ext-dropdown" @click.stop>
          <button
            v-for="ext in extensions" :key="ext"
            class="ext-dropdown-item"
            :class="{active: filterStore.extensionFilter === ext}"
            @click="setExtensionFilter(ext)"
          >
            {{ ext }}
          </button>
        </div>
      </div>
    </div>

    <!-- Status bar -->
    <div v-if="hasFilters" class="filter-status">
      <span class="filter-count">{{ filteredCount }}/{{ totalCount }}</span>
      <button class="filter-clear" @click="clearFilters">
        <SvgIcon name="octicon-x" :size="12"/>
        Clear
      </button>
    </div>
  </div>
</template>
<style scoped>
.diff-file-filter {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--color-secondary);
  margin-bottom: 4px;
}

.filter-group {
  display: flex;
  flex-wrap: wrap;
  gap: 3px;
}

.filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  border: 1px solid var(--color-secondary);
  border-radius: 12px;
  background: transparent;
  color: var(--color-text-light-2);
  font-size: 11px;
  line-height: 18px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s ease;
}

.filter-chip:hover {
  background: var(--color-hover);
  color: var(--color-text);
}

.filter-chip.active {
  background: var(--color-active);
  color: var(--color-text);
  border-color: var(--color-primary);
}

.filter-chip.filter-added.active {
  border-color: var(--color-success-text);
  background: color-mix(in srgb, var(--color-success-text) 15%, transparent);
}

.filter-chip.filter-modified.active {
  border-color: var(--color-warning-text);
  background: color-mix(in srgb, var(--color-warning-text) 15%, transparent);
}

.filter-chip.filter-deleted.active {
  border-color: var(--color-error-text);
  background: color-mix(in srgb, var(--color-error-text) 15%, transparent);
}

.filter-chip.filter-renamed.active {
  border-color: var(--color-info-text);
  background: color-mix(in srgb, var(--color-info-text) 15%, transparent);
}

.ext-dropdown-container {
  position: relative;
}

.ext-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: 10;
  min-width: 100px;
  max-height: 200px;
  overflow-y: auto;
  background: var(--color-body);
  border: 1px solid var(--color-secondary);
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  margin-top: 2px;
}

.ext-dropdown-item {
  display: block;
  width: 100%;
  padding: 4px 10px;
  border: none;
  background: transparent;
  color: var(--color-text);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
}

.ext-dropdown-item:hover {
  background: var(--color-hover);
}

.ext-dropdown-item.active {
  background: var(--color-active);
  font-weight: 600;
}

.filter-status {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 2px;
}

.filter-count {
  font-size: 11px;
  color: var(--color-text-light-2);
}

.filter-clear {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 1px 6px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-light-2);
  font-size: 11px;
  cursor: pointer;
}

.filter-clear:hover {
  background: var(--color-hover);
  color: var(--color-text);
}
</style>
