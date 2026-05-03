<script>
import $ from 'jquery';
import {SvgIcon} from '../svg.js';
import {GET, POST} from '../modules/fetch.js';
import htmx from 'htmx.org';
import {setPageReloadFn} from '../modules/page-reload.js';
import {initRepoIssueTitleEdit, initRepoIssueCommentDelete, initRepoIssueDependencyDelete, initRepoIssueCodeCommentCancel, initRepoIssueComments, initRepoIssueWipToggle, initRepoIssueReferenceIssue, initRepoPullRequestUpdate, initSingleCommentEditor} from '../features/repo-issue.js';
import {initRepoIssueCommentEdit} from '../features/repo-legacy.js';
import {initCompReactionSelector} from '../features/comp/ReactionSelector.js';
import {initMarkupContent, initCommentContent} from '../markup/content.js';
import {initRepoDiffView} from '../features/repo-diff.js';

export default {
  components: {SvgIcon},
  props: {
    repoLink: {type: String, required: true},
    issueId: {type: Number, default: null},
    issueMap: {type: Map, required: true},
    locale: {type: Object, required: true},
  },
  emits: ['close', 'data-changed'],
  data() {
    return {
      paneHTML: null,
      paneLoading: false,
      cleanupFns: [],
      activeTab: 'conversation',
      tabCounts: {comments: 0, commits: 0, files: 0},
    };
  },
  computed: {
    currentIssue() {
      if (this.issueId === null) return null;
      return this.issueMap.get(this.issueId);
    },
    isPR() {
      return this.currentIssue?.pull_request !== null;
    },
    externalURL() {
      if (!this.currentIssue) return '';
      const type = this.isPR ? 'pulls' : 'issues';
      return `${this.repoLink}/${type}/${this.currentIssue.number}`;
    },
  },
  watch: {
    issueId(val) {
      if (val !== null) {
        this.loadPane(val);
      } else {
        this.resetPane();
      }
    },
  },
  mounted() {
    this._delegatedClickHandler = (e) => {
      if (this.activeTab !== 'files') return;
      const diffLink = e.target.closest('.diff-detail-actions a[href]');
      if (diffLink) {
        e.preventDefault();
        e.stopPropagation();
        this.reloadDiffWithParams(diffLink.getAttribute('href'));
      }
    };
    document.addEventListener('click', this._delegatedClickHandler, true);
    this.cleanupFns.push(() => {
      document.removeEventListener('click', this._delegatedClickHandler, true);
    });
  },
  unmounted() {
    for (const fn of this.cleanupFns) {
      try { fn() } catch { /* already gone */ }
    }
    this.cleanupFns = [];
    this.resetPane();
  },
  methods: {
    paneURL(issue) {
      const type = issue.pull_request ? 'pulls' : 'issues';
      return `${this.repoLink}/${type}/${issue.number}/pane`;
    },
    async loadPane(issueId) {
      const issue = this.issueMap.get(issueId);
      if (!issue) return;
      this.paneLoading = true;
      this.paneHTML = null;
      this.activeTab = 'conversation';
      if (issue.pull_request) {
        this.fetchTabCounts(issue);
      }
      try {
        const res = await GET(this.paneURL(issue));
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        this.paneHTML = await res.text();
        this.paneLoading = false;
        this.$nextTick(() => this.initPane());
      } catch {
        this.paneHTML = null;
        this.paneLoading = false;
      }
    },
    async refreshPane() {
      if (this.issueId === null) return;
      const issue = this.issueMap.get(this.issueId);
      if (!issue) return;
      try {
        const res = await GET(this.paneURL(issue));
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        this.paneHTML = await res.text();
        this.activeTab = 'conversation';
        this.$nextTick(() => this.initPane());
      } catch {}
    },
    async fetchTabCounts(issue) {
      try {
        const res = await GET(`${this.repoLink}/pulls/${issue.number}/pane/tab-data`);
        if (!res.ok) return;
        const data = await res.json();
        this.tabCounts = {comments: data.comments || 0, commits: data.commits || 0, files: data.files || 0};
      } catch {}
    },
    async switchTab(tabName) {
      if (tabName === this.activeTab) return;
      this.activeTab = tabName;
      if (tabName === 'conversation') {
        await this.refreshPane();
        return;
      }
      const issue = this.currentIssue;
      if (!issue) return;
      let url;
      if (tabName === 'files') {
        url = `${this.repoLink}/pulls/${issue.number}/pane/files`;
      } else if (tabName === 'commits') {
        url = `${this.repoLink}/pulls/${issue.number}/pane/commits`;
      } else {
        return;
      }
      try {
        const res = await GET(url);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const html = await res.text();
        this.paneHTML = html;
        this.$nextTick(() => {
          const drawer = this.$refs.drawer;
          if (!drawer) return;
          $(drawer).find('.ui.dropdown').dropdown();
          htmx.process(drawer);
          if (tabName === 'files') {
            initRepoDiffView();
          }
        });
      } catch (err) {
        console.error('Failed to load tab content:', err);
      }
    },
    async reloadDiffWithParams(href) {
      const issue = this.currentIssue;
      if (!issue) return;
      const url = `${this.repoLink}/pulls/${issue.number}/pane/files${href.startsWith('?') ? href : ''}`;
      try {
        const res = await GET(url);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        this.paneHTML = await res.text();
        this.$nextTick(() => {
          const drawer = this.$refs.drawer;
          if (!drawer) return;
          $(drawer).find('.ui.dropdown').dropdown();
          htmx.process(drawer);
          initRepoDiffView();
        });
      } catch (err) {
        console.error('Failed to reload diff:', err);
      }
    },
    interceptForm(drawer, formId) {
      const form = drawer.querySelector(formId);
      if (!form) return;
      const handler = async (e) => {
        e.preventDefault();
        try {
          const body = new URLSearchParams(new FormData(form));
          const resp = await POST(form.getAttribute('action'), {data: body});
          if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
          await this.refreshPane();
          this.$emit('data-changed');
        } catch (err) {
          console.error(`Failed to submit ${formId}:`, err);
        }
      };
      form.addEventListener('submit', handler);
      this.cleanupFns.push(() => { form.removeEventListener('submit', handler) });
    },
    initPane() {
      const drawer = this.$refs.drawer;
      if (!drawer) return;
      $(drawer).find('.ui.dropdown').dropdown();
      htmx.process(drawer);
      this._restoreReload = setPageReloadFn(() => this.refreshPane());
      initRepoIssueTitleEdit();
      initRepoIssueCommentEdit();
      const $drawerCommentForm = $(drawer).find('.comment.form');
      if ($drawerCommentForm.length && $drawerCommentForm.find('.combo-markdown-editor').length) {
        initSingleCommentEditor($drawerCommentForm);
      }
      initCompReactionSelector($(drawer));
      initMarkupContent();
      initCommentContent();
      initRepoIssueComments();
      initRepoIssueCommentDelete();
      initRepoIssueDependencyDelete();
      initRepoIssueCodeCommentCancel();
      initRepoIssueWipToggle();
      initRepoIssueReferenceIssue();
      initRepoPullRequestUpdate();
      this.interceptForm(drawer, '#removeDependencyForm');
      this.interceptForm(drawer, '#addDependencyForm');
    },
    resetPane() {
      if (this._restoreReload) {
        this._restoreReload();
        this._restoreReload = null;
      }
      this.paneHTML = null;
      this.paneLoading = false;
      this.activeTab = 'conversation';
      this.tabCounts = {comments: 0, commits: 0, files: 0};
    },
  },
};
</script>

<template>
  <div v-if="paneLoading || paneHTML" ref="drawer" class="dep-board-drawer">
    <div class="dep-board-drawer-header">
      <a v-if="currentIssue" class="ui small basic button" :href="externalURL" target="_blank" rel="noopener">
        <svg-icon name="octicon-link-external" :size="16"/>
      </a>
      <button class="ui small basic button" @click="$emit('close')">
        {{ locale.closePane }}
      </button>
    </div>
    <div v-if="isPR && !paneLoading" class="dep-board-pr-tabs">
      <div class="ui top attached pull tabular menu">
        <a class="item dep-board-pr-tab" :class="{active: activeTab === 'conversation'}" href="#" @click.prevent="switchTab('conversation')">
          <svg-icon name="octicon-comment-discussion"/>
          {{ locale.tabConversation }}
          <span class="ui small label">{{ tabCounts.comments }}</span>
        </a>
        <a class="item dep-board-pr-tab" :class="{active: activeTab === 'commits'}" href="#" @click.prevent="switchTab('commits')">
          <svg-icon name="octicon-git-commit"/>
          {{ locale.tabCommits }}
          <span class="ui small label">{{ tabCounts.commits }}</span>
        </a>
        <a class="item dep-board-pr-tab" :class="{active: activeTab === 'files'}" href="#" @click.prevent="switchTab('files')">
          <svg-icon name="octicon-diff"/>
          {{ locale.tabFiles }}
          <span class="ui small label">{{ tabCounts.files }}</span>
        </a>
      </div>
      <div class="ui tabs divider"/>
    </div>
    <div v-if="paneLoading" class="dep-board-loading is-loading"/>
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div v-else class="dep-board-drawer-body" v-html="paneHTML"/>
  </div>
</template>

<style scoped>
.dep-board-drawer {
  position: fixed;
  top: 0;
  right: 0;
  height: 100vh;
  width: 70%;
  min-width: 600px;
  max-width: 950px;
  z-index: 100;
  background: var(--color-body);
  border-left: 2px solid var(--color-secondary);
  box-shadow: -4px 0 12px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.dep-board-drawer-header {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 0.5rem;
  border-bottom: 1px solid var(--color-secondary);
  flex-shrink: 0;
}
.dep-board-pr-tabs {
  flex-shrink: 0;
}
.dep-board-drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem;
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
</style>
