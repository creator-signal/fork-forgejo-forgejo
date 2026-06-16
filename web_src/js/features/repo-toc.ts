// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

export function initRepoToc() {
  const toc = document.getElementById('toc');
  if (!toc) return;
  const root = document.createElement('ul');
  const stack: {level: number; ul: HTMLUListElement}[] = [
    {level: 0, ul: root},
  ];
  for (const anchor of document.querySelectorAll<HTMLAnchorElement>(
    '.anchor',
  )) {
    const heading = anchor.parentElement;
    let level = Number.parseInt(heading.tagName.slice(1));
    if (level === 1) level = 2;
    const li = document.createElement('li');
    const ref = document.createElement('a');
    ref.addEventListener('click', (ev) => {
      document
        .querySelector<HTMLAnchorElement>(
          `#user-content-${(ev.target as HTMLElement).dataset.ref}>a`,
        )
        ?.click();
    });
    ref.href = anchor.hash;
    ref.dataset.ref = anchor.hash.slice(1);
    ref.textContent = heading.textContent;
    li.append(ref);
    while (stack[stack.length - 1].level >= level) {
      stack.pop();
    }
    const parent = stack[stack.length - 1].ul;
    parent.append(li);
    const child = document.createElement('ul');
    li.append(child);
    stack.push({level, ul: child});
  }
  toc.append(root);
}
