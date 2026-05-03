<script>
import $ from 'jquery';
import {GET} from '../modules/fetch.js';
import IssueDependencyPane from './IssueDependencyPane.vue';

export default {
  components: {IssueDependencyPane},
  props: {
    repoLink: {type: String, required: true},
    isDependenciesEnabled: {type: Boolean, default: false},
    locale: {type: Object, required: true},
  },
  data() {
    return {
      loading: true,
      errorText: null,
      columns: [],
      cycles: [],
      selectedMilestone: '',
      selectedState: '',
      hideBlocked: false,
      selectedIssueId: null,
      selectedMilestoneId: null,
      connectedIssues: new Set(),
      connectedMilestoneIds: new Set(),
      cleanupFns: [],
      cardHtmlCache: {},
      loadedCardIds: new Set(),
      pendingCardRequests: new Set(),
    };
  },
  computed: {
    issueMap() {
      const map = new Map();
      for (const col of this.columns) {
        if (col.is_milestone) continue;
        for (const issue of col.issues) {
          map.set(issue.id, issue);
        }
      }
      return map;
    },
    hasSelection() {
      return this.selectedIssueId !== null || this.selectedMilestoneId !== null;
    },
  },
  mounted() {
    this.bindExternalFilters();
    this.fetchData();
    this.cardObserver = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          const issueId = Number(entry.target.dataset.issueId);
          if (issueId && !this.loadedCardIds.has(issueId) && !this.pendingCardRequests.has(issueId)) {
            this.loadCard(issueId);
          }
        }
      }
    }, {rootMargin: '200px'});
  },
  unmounted() {
    if (this.cardObserver) {
      this.cardObserver.disconnect();
      this.cardObserver = null;
    }
    for (const fn of this.cleanupFns) {
      try { fn() } catch { /* already gone */ }
    }
    this.cleanupFns = [];
  },
  methods: {
    clearSelection() {
      this.selectedIssueId = null;
      this.selectedMilestoneId = null;
      this.connectedIssues = new Set();
      this.connectedMilestoneIds = new Set();
    },
    bfsCollect(startId, getNeighbors) {
      const visited = new Set([startId]);
      const queue = [startId];
      while (queue.length > 0) {
        const id = queue.shift();
        const issue = this.issueMap.get(id);
        if (!issue) continue;
        for (const neighborId of getNeighbors(issue)) {
          if (!visited.has(neighborId)) {
            visited.add(neighborId);
            queue.push(neighborId);
          }
        }
      }
      return visited;
    },
    computeConnected(issueId) {
      const downstream = this.bfsCollect(issueId, (issue) => issue.depends_on || []);
      const upstream = this.bfsCollect(issueId, (issue) => issue.blocks || []);
      for (const id of upstream) {
        downstream.add(id);
      }
      return downstream;
    },
    computeConnectedMilestones(connected) {
      const msIds = new Set();
      for (const issueId of connected) {
        const issue = this.issueMap.get(issueId);
        if (issue && issue.milestone && issue.milestone.id) {
          msIds.add(issue.milestone.id);
        }
      }
      return msIds;
    },
    issuesInMilestone(msId) {
      const ids = [];
      for (const [id, issue] of this.issueMap) {
        if (issue.milestone && issue.milestone.id === msId) {
          ids.push(id);
        }
      }
      return ids;
    },
    toggleSelect(issueId) {
      this.selectedMilestoneId = null;
      if (this.selectedIssueId === issueId) {
        this.clearSelection();
      } else {
        this.selectedIssueId = issueId;
        const connected = this.computeConnected(issueId);
        this.connectedIssues = connected;
        this.connectedMilestoneIds = this.computeConnectedMilestones(connected);
      }
    },
    toggleSelectMilestone(msId) {
      this.selectedIssueId = null;
      if (this.selectedMilestoneId === msId) {
        this.clearSelection();
      } else {
        this.selectedMilestoneId = msId;
        const msIssueIds = this.issuesInMilestone(msId);
        const combined = new Set();
        for (const id of msIssueIds) {
          for (const cid of this.computeConnected(id)) {
            combined.add(cid);
          }
        }
        this.connectedIssues = combined;
        const msIds = this.computeConnectedMilestones(combined);
        msIds.add(msId);
        this.connectedMilestoneIds = msIds;
      }
    },
    extractFilterMilestones(data) {
      for (const col of data.columns || []) {
        if (col.is_milestone && col.milestones) {
          this.updateMilestoneOptions(col.milestones);
          return;
        }
      }
    },
    updateMilestoneOptions(milestones) {
      const select = document.getElementById('dep-board-filter-milestone');
      if (!select) return;
      const current = select.value;
      select.innerHTML = `<option value="">${this.locale.stateAll}</option>`;
      for (const ms of milestones) {
        const opt = document.createElement('option');
        opt.value = String(ms.id);
        opt.textContent = ms.title;
        select.append(opt);
      }
      select.value = current;
    },
    bindExternalFilters() {
      this.bindMilestoneFilter();
      this.bindHideBlockedToggle();
      this.bindStateFilter();
    },
    bindMilestoneFilter() {
      const milestoneSelect = document.getElementById('dep-board-filter-milestone');
      if (!milestoneSelect) return;
      const handleChange = (value) => {
        this.selectedMilestone = value;
        this.fetchData();
      };
      $(milestoneSelect).dropdown({onChange: handleChange});
      this.cleanupFns.push(() => { $(milestoneSelect).dropdown('destroy') });
      const changeHandler = () => {
        handleChange(milestoneSelect.value);
      };
      milestoneSelect.addEventListener('change', changeHandler);
      this.cleanupFns.push(() => { milestoneSelect.removeEventListener('change', changeHandler) });
    },
    bindHideBlockedToggle() {
      const hideBlockedBtn = document.getElementById('dep-board-filter-hide-blocked');
      if (!hideBlockedBtn) return;
      const clickHandler = () => {
        this.hideBlocked = !this.hideBlocked;
        hideBlockedBtn.classList.toggle('active', this.hideBlocked);
        this.fetchData();
      };
      hideBlockedBtn.addEventListener('click', clickHandler);
      this.cleanupFns.push(() => { hideBlockedBtn.removeEventListener('click', clickHandler) });
    },
    bindStateFilter() {
      const stateContainer = document.getElementById('dep-board-filter-state');
      if (!stateContainer) return;
      const clickHandler = (e) => {
        const btn = e.target.closest('button[data-value]');
        if (!btn) return;
        for (const b of stateContainer.querySelectorAll('button')) b.classList.remove('active');
        btn.classList.add('active');
        this.selectedState = btn.dataset.value;
        this.fetchData();
      };
      stateContainer.addEventListener('click', clickHandler);
      this.cleanupFns.push(() => { stateContainer.removeEventListener('click', clickHandler) });
    },
    columnId(col) {
      return col.is_milestone ? 'milestones' : `depth_${col.depth}`;
    },
    columnTitle(col) {
      return col.is_milestone ? this.locale.milestones : this.locale.depth.replace('%d', col.depth);
    },
    async fetchData(opts = {}) {
      this.loading = true;
      this.errorText = null;
      if (!opts.keepSelection) {
        this.clearSelection();
      }
      this.cardHtmlCache = {};
      this.loadedCardIds = new Set();
      this.pendingCardRequests = new Set();

      const params = new URLSearchParams();
      if (this.selectedMilestone) params.set('milestone', this.selectedMilestone);
      if (this.selectedState) params.set('state', this.selectedState);
      if (this.hideBlocked) params.set('hide_blocked', '1');
      const qs = params.toString();
      const url = `${this.repoLink}/issues/dependency-board/data${qs ? `?${qs}` : ''}`;

      try {
        const res = await GET(url);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        this.columns = data.columns || [];
        this.cycles = data.cycles || [];
        this.extractFilterMilestones(data);
        this.$nextTick(() => this.observeCards());
      } catch (err) {
        this.errorText = err.message || 'Failed to load board';
      } finally {
        this.loading = false;
      }
    },
    observeCards() {
      if (!this.cardObserver) return;
      const els = this.$el.querySelectorAll('[data-issue-id]');
      for (const el of els) {
        this.cardObserver.observe(el);
      }
    },
    async loadCard(issueId) {
      this.pendingCardRequests.add(issueId);
      try {
        const res = await GET(`${this.repoLink}/issues/dependency-board/card/${issueId}`);
        if (!res.ok) return;
        const data = await res.json();
        this.cardHtmlCache[issueId] = data.card_html || '';
        this.loadedCardIds.add(issueId);
      } catch {
        // silently skip failed card loads
      } finally {
        this.pendingCardRequests.delete(issueId);
      }
    },
  },
};
</script>

<template>
  <div class="dep-board">
    <div class="dep-board-main">
      <div v-if="cycles.length > 0" class="ui warning message dep-board-cycle-warning">
        <strong>{{ locale.cycleWarning }}</strong>
        <div v-for="(cycle, i) in cycles" :key="i">
          {{ cycle.map(id => '#' + id).join(' → ') }}
        </div>
      </div>

      <div v-if="loading" class="dep-board-loading is-loading"/>
      <div v-else-if="errorText" class="dep-board-loading">
        <p class="text red">
          {{ errorText }}
        </p>
      </div>
      <div v-else-if="!columns.length" class="dep-board-loading">
        <p>{{ locale.noDependencies }}</p>
      </div>
      <div v-else class="board">
        <div
          v-for="col in columns"
          :key="columnId(col)"
          class="project-column"
          :class="{'dep-board-milestone-col': col.is_milestone}"
        >
          <div class="project-column-header">
            <div class="ui large label project-column-title tw-py-1">
              <div class="ui small circular grey label project-column-issue-count">
                {{ col.is_milestone ? col.milestones.length : col.issues.length }}
              </div>
              <span class="project-column-title-label">{{ columnTitle(col) }}</span>
            </div>
          </div>
          <div class="divider"/>

          <template v-if="col.is_milestone">
            <div class="ui cards">
              <div
                v-for="ms in col.milestones"
                :key="ms.id"
                class="dep-board-ms-card"
                :class="{
                  'dep-board-dimmed': hasSelection && selectedMilestoneId !== ms.id && !connectedMilestoneIds.has(ms.id),
                  'dep-board-selected': selectedMilestoneId === ms.id,
                  'dep-board-ms-card-closed': ms.state === 'closed',
                  'dep-board-ms-card-overdue': ms.is_overdue,
                }"
                @click.prevent.stop="toggleSelectMilestone(ms.id)"
              >
                <div class="dep-board-ms-card-header">
                  <span class="dep-board-ms-card-name">{{ ms.title }}</span>
                  <span class="dep-board-ms-card-state ui mini label" :class="ms.state === 'closed' ? 'green' : 'grey'">
                    {{ ms.state }}
                  </span>
                </div>
                <div class="dep-board-ms-card-progress">
                  <div class="dep-board-ms-card-bar-track">
                    <div class="dep-board-ms-card-bar" :style="{width: ms.completeness + '%'}"/>
                  </div>
                  <span class="dep-board-ms-card-progress-text">{{ ms.completeness }}%</span>
                </div>
                <div class="dep-board-ms-card-stats">
                  <span>{{ ms.open_issues }} open / {{ ms.closed_issues }} closed</span>
                </div>
                <div v-if="ms.due_on" class="dep-board-ms-card-deadline" :class="{'text red': ms.is_overdue}">
                  {{ ms.due_on }}
                </div>
              </div>
            </div>
          </template>
          <template v-else>
            <div class="ui cards">
              <!-- eslint-disable vue/no-v-html -->
              <div
                v-for="issue in col.issues"
                :key="issue.id"
                class="issue-card tw-break-anywhere"
                :class="{
                  'dep-board-dimmed': hasSelection && !connectedIssues.has(issue.id),
                  'dep-board-selected': selectedIssueId === issue.id,
                }"
                :data-issue-id="issue.id"
                @click.prevent.stop="toggleSelect(issue.id)"
                v-html="cardHtmlCache[issue.id] || ''"
              />
              <!-- eslint-enable vue/no-v-html -->
            </div>
          </template>
        </div>
      </div>
    </div>

    <IssueDependencyPane
      :repo-link="repoLink"
      :issue-id="selectedIssueId"
      :issue-map="issueMap"
      :locale="locale"
      @close="clearSelection"
      @data-changed="fetchData({keepSelection: true})"
    />
  </div>
</template>

<style scoped>
.dep-board {
  width: 100%;
  min-height: 400px;
  display: flex;
}
.dep-board-main {
  flex: 1;
  min-width: 0;
}
.dep-board-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  text-align: center;
  color: var(--color-text-light-2);
  padding: 2rem;
}
.dep-board-cycle-warning {
  margin-bottom: 1rem;
}
.dep-board-milestone-col {
  border-left: 3px solid var(--color-primary);
}
.dep-board-milestone-col .project-column-title {
  color: var(--color-primary);
  font-weight: var(--font-weight-semibold);
}

.dep-board-ms-card {
  background: var(--color-card);
  border: 1px solid var(--color-secondary);
  border-radius: var(--border-radius);
  padding: 0.75rem;
  margin-bottom: 0.5rem;
  cursor: pointer;
  transition: filter 0.2s ease, opacity 0.2s ease;
}
.dep-board-ms-card:hover {
  background-color: var(--color-hover);
}
.dep-board-ms-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}
.dep-board-ms-card-name {
  font-weight: var(--font-weight-semibold);
  font-size: 0.9rem;
}
.dep-board-ms-card-progress {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.4rem;
}
.dep-board-ms-card-bar-track {
  flex: 1;
  height: 6px;
  background: var(--color-secondary);
  border-radius: 3px;
}
.dep-board-ms-card-bar {
  height: 100%;
  background: var(--color-primary);
  border-radius: 3px;
  transition: width 0.3s ease;
}
.dep-board-ms-card-progress-text {
  font-size: 0.8rem;
  color: var(--color-text-light-2);
  white-space: nowrap;
}
.dep-board-ms-card-stats {
  font-size: 0.8rem;
  color: var(--color-text-light-2);
  margin-bottom: 0.3rem;
}
.dep-board-ms-card-deadline {
  font-size: 0.8rem;
  color: var(--color-text-light-2);
}
.dep-board-ms-card-closed {
  opacity: 0.7;
}
.dep-board-ms-card-overdue .dep-board-ms-card-deadline {
  color: var(--color-red);
  font-weight: var(--font-weight-semibold);
}

.dep-board-ms-card.dep-board-dimmed {
  opacity: var(--opacity-disabled);
  filter: saturate(0.1);
}
.dep-board-ms-card.dep-board-selected {
  background-color: var(--color-active);
}
</style>
