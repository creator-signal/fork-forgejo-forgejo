import {POST} from '../modules/fetch.js';

const Page = Object.freeze({
  Actions: 0,
  Code: 1,
  Homepage: 2,
  Issues: 3,
  Notifications: 4,
  Projects: 5,
  Pulls: 6,
  Releases: 7,
  Wiki: 8,
});

let goto_state = false;
const list = document.querySelector(`\
#issue-list,\
#notification_table,\
#repo-files-table tbody,\
.milestone-list,\
.wiki-pages-list tbody\
`);

function keyboardSelector(up: boolean) {
  const rows = Array.from(list?.children ?? []) as HTMLElement[];
  if (rows.length === 0) return;
  const cur = rows.findIndex((row) =>
    row.classList.contains('keyboard-selected'),
  );
  if (cur === -1) {
    rows[0].classList.add('keyboard-selected');
    rows[0].querySelector('a')?.focus();
    return;
  }
  let el = rows[cur];
  if (up) {
    if (el.previousElementSibling) {
      el = el.previousElementSibling as HTMLElement;
    }
  } else if (el.nextElementSibling) {
    el = el.nextElementSibling as HTMLElement;
  }
  el.classList.add('keyboard-selected');
  el.querySelector('a')?.focus();
  if (rows[cur] === el) return;
  rows[cur].classList.remove('keyboard-selected');
}

function initPopupTabs() {
  const dialog = document.querySelector<HTMLDialogElement>('#shortcuts');
  const btns = dialog.querySelectorAll<HTMLButtonElement>('[data-tab]');
  const lists = dialog.querySelectorAll('ul');
  const selectList = (name: string) => {
    for (const btn of btns) {
      btn.classList.toggle('active', btn.dataset.tab === name);
    }
    for (const ls of lists) {
      const active = ls.dataset.tab !== name;
      ls.style.visibility = active ? 'hidden' : 'visible';
      ls.toggleAttribute('aria-hidden', active);
    }
  };
  for (const btn of btns) {
    btn.addEventListener('click', () => {
      selectList(btn.dataset.tab);
    });
  }
  dialog.addEventListener('close', () => {
    selectList('global');
  });
}

function initEnableCheckbox() {
  const checkbox = document.querySelector<HTMLInputElement>('#shortcuts input');
  checkbox?.addEventListener('change', () => {
    const enable = checkbox.checked;
    if (enable) document.addEventListener('keydown', onKeydown);
    else document.removeEventListener('keydown', onKeydown);
    const data = new URLSearchParams();
    if (enable) {
      data.set('enable_shortcuts', 'on');
    }
    POST(`${window.config.appSubUrl}/user/settings/appearance/shortcuts`, {
      headers: {Accept: 'application/json'},
      data,
    }).catch(console.error);
  });
}

function goto(page: (typeof Page)[keyof typeof Page]) {
  if (!goto_state) return true;
  switch (page) {
    case Page.Actions:
      document.querySelector<HTMLAnchorElement>('#repo-actions-tab')?.click();
      break;
    case Page.Code:
      document.querySelector<HTMLAnchorElement>('#repo-code-tab')?.click();
      break;
    case Page.Homepage:
      window.location.pathname = '/';
      break;
    case Page.Issues: {
      const el = document.querySelector<HTMLAnchorElement>('#repo-issues-tab');
      if (el) {
        el.click();
      } else window.location.pathname = '/issues';
      break;
    }
    case Page.Notifications:
      window.location.pathname = '/notifications';
      break;
    case Page.Projects:
      document.querySelector<HTMLAnchorElement>('#repo-projects-tab')?.click();
      break;
    case Page.Pulls: {
      const el = document.querySelector<HTMLAnchorElement>(
        '#repo-pull-requests-tab',
      );
      if (el) {
        el.click();
      } else window.location.pathname = '/pulls';
      break;
    }
    case Page.Releases:
      document.querySelector<HTMLAnchorElement>('#repo-releases-tab')?.click();
      break;
    case Page.Wiki:
      document.querySelector<HTMLAnchorElement>('#repo-wiki-tab')?.click();
  }
  return false;
}

function onKeydown(e: KeyboardEvent) {
  if (
    (e.altKey ||
      e.ctrlKey ||
      e.metaKey ||
      e.shiftKey ||
      !(e.target instanceof Element) ||
      e.target.closest('input, textarea, [contenteditable]')) &&
    (e.target as HTMLInputElement).type !== 'checkbox'
  ) {
    goto_state = false;
    return;
  }
  switch (e.key) {
    case 'a':
      goto(Page.Actions);
      break;
    case 'b':
      document.querySelector<HTMLAnchorElement>('#blame-btn')?.click();
      break;
    case 'c':
      if (goto(Page.Code)) {
        document
          .querySelector<HTMLAnchorElement>(
            '.issue-list-new, .release-list-buttons .primary',
          )
          ?.click();
      }
      break;
    case 'g':
      if (goto_state) return;
      goto_state = true;
      setTimeout(() => {
        goto_state = false;
      }, 750);
      return;
    case 'h':
      if (goto(Page.Homepage)) {
        document.querySelector<HTMLAnchorElement>('#history-btn')?.click();
      }
      break;
    case 'i':
      goto(Page.Issues);
      break;
    case 'j':
      (e.target as HTMLInputElement).blur();
      keyboardSelector(false);
      break;
    case 'k':
      (e.target as HTMLInputElement).blur();
      keyboardSelector(true);
      break;
    case 'n':
      goto(Page.Notifications);
      break;
    case 'o':
      goto(Page.Projects);
      break;
    case 'p':
      goto(Page.Pulls);
      break;
    case 'r':
      if (goto(Page.Releases)) {
        document.querySelector<HTMLAnchorElement>('#raw-btn')?.click();
      }
      break;
    case 'w':
      if (goto(Page.Wiki)) {
        document
          .querySelector<HTMLAnchorElement>('.branch-dropdown-button')
          ?.click();
        e.preventDefault();
      }
      break;
    case 'x':
      document
        .querySelector<HTMLInputElement>(
          '.keyboard-selected input[type="checkbox"]',
        )
        ?.click();
      break;
    case 'y':
      document.querySelector<HTMLAnchorElement>('#permalink-btn')?.click();
      break;
    case '/':
      document.querySelector<HTMLInputElement>('input[type="search"]')?.focus();
      e.preventDefault();
      break;
    case 'ArrowLeft': {
      const btns = Array.from(
        document.querySelectorAll<HTMLButtonElement>('#shortcuts .item'),
      );
      const idx = btns.findIndex((btn) => btn.classList.contains('active'));
      if (idx > 0) {
        btns[idx - 1].click();
      }
      break;
    }
    case 'ArrowRight': {
      const btns = Array.from(
        document.querySelectorAll<HTMLButtonElement>('#shortcuts .item'),
      );
      const idx = btns.findIndex((btn) => btn.classList.contains('active'));
      if (idx < btns.length - 1) {
        btns[idx + 1].click();
      }
      break;
    }
    case 'Escape':
      document.querySelector<HTMLDialogElement>('#shortcuts')?.close();
  }
  goto_state = false;
}

export function initUserShortcuts() {
  initPopupTabs();
  initEnableCheckbox();
  if (document.querySelector<HTMLInputElement>('#shortcuts input')?.checked) {
    document.addEventListener('keydown', onKeydown);
  }
  document.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === '?' && e.shiftKey) {
      document.querySelector<HTMLDialogElement>('#shortcuts')?.showModal();
    }
  });
}
