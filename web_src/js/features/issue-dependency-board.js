import {createApp} from 'vue';

export async function initIssueDependencyBoard() {
  const el = document.getElementById('issue-dependency-board');
  if (!el) return;

  const {default: Board} = await import(
    /* webpackChunkName: "issue-dependency-board" */ '../components/IssueDependencyBoard.vue',
  );
  try {
    const app = createApp(Board, {
      repoLink: el.getAttribute('data-repo-link'),
      isDependenciesEnabled: el.getAttribute('data-is-dependencies-enabled') === 'true',
      locale: {
        title: el.getAttribute('data-locale-title'),
        noDependencies: el.getAttribute('data-locale-no-dependencies'),
        loading: el.getAttribute('data-locale-loading'),
        depth: el.getAttribute('data-locale-depth'),
        dependents: el.getAttribute('data-locale-dependents'),
        cycleWarning: el.getAttribute('data-locale-cycle-warning'),
        milestone: el.getAttribute('data-locale-milestone'),
        stateOpen: el.getAttribute('data-locale-state-open'),
        stateClosed: el.getAttribute('data-locale-state-closed'),
        stateAll: el.getAttribute('data-locale-state-all'),
        filterMilestone: el.getAttribute('data-locale-filter-milestone'),
        closePane: el.getAttribute('data-locale-close-pane'),
        loadingPane: el.getAttribute('data-locale-loading-pane'),
        tabConversation: el.getAttribute('data-locale-tab-conversation'),
        tabCommits: el.getAttribute('data-locale-tab-commits'),
        tabFiles: el.getAttribute('data-locale-tab-files'),
      },
    });
    app.mount(el);
  } catch (err) {
    console.error('IssueDependencyBoard failed to load', err);
    el.textContent = el.getAttribute('data-locale-component-failed-to-load');
  }
}
