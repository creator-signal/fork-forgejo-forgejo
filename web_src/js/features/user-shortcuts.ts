const Page = Object.freeze({
  Actions: 0,
  Code: 1,
  Dashboard: 2,
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
    rows[0].scrollIntoView({block: 'nearest'});
    return;
  }
  const el = rows[cur];
  if (up) {
    if (el.previousElementSibling) {
      el.previousElementSibling.classList.add('keyboard-selected');
      el.previousElementSibling.scrollIntoView({block: 'nearest'});
    } else return;
  } else {
    if (el.nextElementSibling) {
      el.nextElementSibling.classList.add('keyboard-selected');
      el.nextElementSibling.scrollIntoView({block: 'nearest'});
    } else return;
  }
  rows[cur].classList.remove('keyboard-selected');
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
    case Page.Dashboard:
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

export function initUserShortcuts() {
  document.addEventListener('keydown', (e) => {
    if (
      e.altKey ||
      e.ctrlKey ||
      e.metaKey ||
      e.shiftKey ||
      !(e.target instanceof Element) ||
      e.target.closest(
        'input, textarea, [contenteditable], .CodeMirror, .cm-editor',
      )
    ) {
      goto_state = false;
      return;
    }
    switch (e.key) {
      case 'Enter':
        if (e.target.tagName !== 'BUTTON') {
          document
            .querySelector<HTMLAnchorElement>('.keyboard-selected a')
            ?.click();
        }
        break;
      case 'a':
        goto(Page.Actions);
        break;
      case 'b':
        document.querySelector<HTMLAnchorElement>('#blame-btn')?.click();
        break;
      case 'c':
        goto(Page.Code);
        break;
      case 'd':
        goto(Page.Dashboard);
        break;
      case 'g':
        if (goto_state) return;
        goto_state = true;
        setTimeout(() => {
          goto_state = false;
        }, 750);
        return;
      case 'h':
        document.querySelector<HTMLAnchorElement>('#history-btn')?.click();
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
        goto(Page.Wiki);
        break;
      case 'y':
        document.querySelector<HTMLAnchorElement>('#permalink-btn')?.click();
    }
    goto_state = false;
  });
}
