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
type PageType = (typeof Page)[keyof typeof Page];

let goto_state = false;
let where: PageType = Page.Homepage;
const list = document.querySelector(`\
#issue-list,\
#notification_table,\
#repo-files-table tbody,\
.milestone-list,\
.wiki-pages-list tbody\
`);

function keyboardSelector(up: boolean, tab: boolean) {
  const rows = Array.from(list?.children ?? []) as HTMLElement[];
  if (rows.length === 0) return;
  const cur = rows.findIndex((row) =>
    row.classList.contains('keyboard-selected'),
  );
  if (cur === -1) {
    const idx = up ? rows.length - 1 : 0;
    rows[idx].classList.add('keyboard-selected');
    rows[idx].querySelector('a')?.focus();
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
  if (tab) {
    if (rows[cur] === el) {
      el.classList.remove('keyboard-selected');
      return;
    }
  } else el.querySelector('a')?.focus();
  if (rows[cur] === el) return;
  el.classList.add('keyboard-selected');
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

function clearErrors(dialog: HTMLDialogElement) {
  for (const el of dialog.querySelectorAll<HTMLElement>('[data-error]')) {
    el.style.display = 'none';
    el.toggleAttribute('aria-hidden', true);
  }
}

function showError(dialog: HTMLDialogElement, type: string) {
  clearErrors(dialog);
  const div = dialog.querySelector<HTMLElement>(`[data-error="${type}"]`);
  if (!div) return;
  div.style.display = '';
  div.toggleAttribute('aria-hidden', false);
}

function validate(input: HTMLInputElement, dialog: HTMLDialogElement) {
  const num = Number(input.value);
  if (Number.isNaN(num) || num < 1) {
    showError(dialog, 'invalid');
    return;
  }
  if (!document.querySelector(`#L${num}`)) {
    showError(dialog, 'notfound');
    return;
  }
  window.location.hash = `#L${num}`;
  input.value = '';
  dialog.close();
}

function initLineJump() {
  const dialog = document.querySelector<HTMLDialogElement>('#line-jump');
  if (!dialog) return;
  clearErrors(dialog);
  const input = dialog.querySelector('input');
  if (!input) return;
  input.max = document.querySelectorAll('.code-view tr').length.toString();
  dialog.addEventListener('close', () => {
    input.value = '';
    clearErrors(dialog);
  });
  input.addEventListener('keydown', (ev) => {
    if (ev.key !== 'Enter') return;
    validate(input, dialog);
  });
  const btn = dialog.querySelector('[type="button"]');
  if (!btn) return;
  btn.addEventListener('click', () => {
    validate(input, dialog);
  });
}

function goto(page: PageType): boolean {
  if (!goto_state) return false;
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
      if (document.querySelector('.repo-header')) {
        document.querySelector<HTMLAnchorElement>('#repo-issues-tab')?.click();
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
      if (document.querySelector('.repo-header')) {
        document
          .querySelector<HTMLAnchorElement>('#repo-pull-requests-tab')
          ?.click();
      } else window.location.pathname = '/pulls';
      break;
    }
    case Page.Releases:
      document.querySelector<HTMLAnchorElement>('#repo-releases-tab')?.click();
      break;
    case Page.Wiki:
      document.querySelector<HTMLAnchorElement>('#repo-wiki-tab')?.click();
  }
  return true;
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
      if (goto(Page.Actions)) return;
      switch (where) {
        case Page.Issues:
        case Page.Pulls:
          document
            .querySelector<HTMLAnchorElement>('#assignee-dropdown')
            ?.click();
          e.preventDefault();
          break;
      }
      break;
    case 'b':
      document.querySelector<HTMLAnchorElement>('#blame-btn')?.click();
      break;
    case 'c':
      if (goto(Page.Code)) return;
      document
        .querySelector<HTMLAnchorElement>(
          '.issue-list-new, .release-list-buttons .primary',
        )
        ?.click();
      break;
    case 'g':
      if (goto_state) return;
      goto_state = true;
      setTimeout(() => {
        goto_state = false;
      }, 750);
      return;
    case 'h': {
      if (goto(Page.Homepage)) return;
      const btn = document.querySelector<HTMLAnchorElement>('#history-btn');
      if (btn) btn.click();
      else if (document.querySelector('#commits-table')) {
        window.location.pathname = window.location.pathname.replace(
          'commits',
          'src',
        );
      }
      break;
    }
    case 'i':
      goto(Page.Issues);
      break;
    case 'j':
      (e.target as HTMLInputElement).blur();
      keyboardSelector(false, false);
      break;
    case 'k':
      (e.target as HTMLInputElement).blur();
      keyboardSelector(true, false);
      break;
    case 'l':
      switch (where) {
        case Page.Code: {
          const dialog =
            document.querySelector<HTMLDialogElement>('#line-jump');
          if (dialog) {
            dialog.showModal();
            dialog.querySelector('input')?.focus();
            e.preventDefault();
          }
          break;
        }
        case Page.Issues:
        case Page.Pulls:
          document.querySelector<HTMLAnchorElement>('#label-dropdown')?.click();
          e.preventDefault();
          break;
      }
      break;
    case 'm':
      document.querySelector<HTMLAnchorElement>('#milestone-dropdown')?.click();
      e.preventDefault();
      break;
    case 'n':
      goto(Page.Notifications);
      break;
    case 'o':
      goto(Page.Projects);
      break;
    case 'p':
      if (goto(Page.Pulls)) return;
      switch (where) {
        case Page.Issues:
        case Page.Pulls:
          document
            .querySelector<HTMLAnchorElement>('#project-dropdown')
            ?.click();
          e.preventDefault();
          break;
      }
      break;
    case 'q':
    case 'Escape':
      document.querySelector<HTMLDialogElement>('dialog[open]')?.close();
      break;
    case 'r':
      if (goto(Page.Releases)) return;
      document.querySelector<HTMLAnchorElement>('#raw-btn')?.click();
      break;
    case 's':
      document.querySelector<HTMLAnchorElement>('#sort-dropdown')?.focus();
      break;
    case 't':
      document.querySelector<HTMLAnchorElement>('#type-dropdown')?.focus();
      break;
    case 'u':
      document.querySelector<HTMLAnchorElement>('#author-dropdown')?.click();
      e.preventDefault();
      break;
    case 'w':
      if (goto(Page.Wiki)) return;
      document
        .querySelector<HTMLAnchorElement>('.branch-dropdown-button')
        ?.click();
      e.preventDefault();
      break;
    case 'x':
      document
        .querySelector<HTMLInputElement>(
          '.keyboard-selected input[type="checkbox"]',
        )
        ?.click();
      break;
    case 'y': {
      const perma = document.querySelector<HTMLAnchorElement>('#permalink-btn');
      if (!perma) return;
      navigator.clipboard.writeText(perma.href);
      break;
    }
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
    case 'Tab': {
      setTimeout(() => {
        const ks = document.querySelector('.keyboard-selected');
        if (!ks) return;
        if (ks.contains(document.activeElement)) return;
        keyboardSelector(false, true);
      }, 50);
    }
  }
  goto_state = false;
}

export function initUserShortcuts() {
  initPopupTabs();
  initEnableCheckbox();
  initLineJump();
  if (document.querySelector<HTMLInputElement>('#shortcuts input')?.checked) {
    document.addEventListener('keydown', onKeydown);
  }
  document.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === '?') {
      document.querySelector<HTMLDialogElement>('#shortcuts')?.showModal();
    }
    if (!document.querySelector<HTMLInputElement>('#shortcuts input')?.checked) return;
    if (e.key === 'Tab' && e.shiftKey) {
      setTimeout(() => {
        const ks = document.querySelector('.keyboard-selected');
        if (!ks) return;
        if (ks.contains(document.activeElement)) return;
        keyboardSelector(true, true);
      }, 50);
    }
  });
  if (document.querySelector('#repo-code-tab.active')) {
    where = Page.Code;
  } else if (document.querySelector('#repo-issues-tab.active')) {
    where = Page.Issues;
  } else if (document.querySelector('#repo-pull-requests-tab.active')) {
    where = Page.Pulls;
  }
}
