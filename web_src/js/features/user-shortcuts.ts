let goto_state = false;

function keyboardSelector(up: boolean) {
  const subpath = window.location.pathname.split('/')[3];
  let query = '';
  switch (subpath) {
    case 'issues':
    case 'pulls':
      query = '#issue-list';
      break;
    case 'projects':
      query = '.milestone-list';
      break;
    case 'src':
      query = '#repo-files-table tbody';
      break;
    case 'wiki':
      query = '.wiki-pages-list tbody';
      break;
    default:
      if (window.location.pathname.split('/').length === 3) {
        query = '#repo-files-table tbody';
      } else return;
  }
  const rows = Array.from(
    document.querySelector(query)?.children ?? [],
  ) as HTMLElement[];
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

function goto(query: string) {
  if (!goto_state) return true;
  document.querySelector<HTMLAnchorElement>(query)?.click();
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
        goto('#repo-actions-tab');
        break;
      case 'c':
        goto('#repo-code-tab');
        break;
      case 'g':
        if (goto_state) return;
        goto_state = true;
        setTimeout(() => {
          goto_state = false;
        }, 750);
        return;
      case 'i':
        goto('#repo-issues-tab');
        break;
      case 'j':
        (e.target as HTMLInputElement).blur();
        keyboardSelector(false);
        break;
      case 'k':
        (e.target as HTMLInputElement).blur();
        keyboardSelector(true);
        break;
      case 'o':
        goto('#repo-projects-tab');
        break;
      case 'p':
        goto('#repo-pull-requests-tab');
        break;
      case 'w':
        goto('#repo-wiki-tab');
        break;
    }
    goto_state = false;
  });
}
