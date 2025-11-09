// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Details can be opened by clicking summary or by pressing Space or Enter while
// being focused on summary. But without JS options for closing it are limited.
// Event listeners in this file provide more convenient options for that:
// click iteration with anything on the page and pressing Escape.

export function initDropdowns() {
  document.addEventListener('click', (event) => {
    const dropdown = document.querySelector<HTMLDetailsElement>('details.dropdown[open]');
    if (dropdown === null)
      // No open dropdowns on page, nothing to do.
      return;

    const target = event.target as HTMLElement;
    if (dropdown.contains(target))
      // User clicked something in the open dropdown, don't interfere.
      return;

    // User clicked something that isn't the open dropdown, so close it.
    dropdown.removeAttribute('open');
  });

  // Close open dropdowns on Escape press
  document.addEventListener('keydown', (event) => {
    if (!['Escape', 'ArrowUp', 'ArrowDown'].includes(event.key))
      // This eventListener is only concerned about a few keys
      return;

    const dropdown = document.querySelector<HTMLDetailsElement>('details.dropdown[open]');
    if (dropdown === null)
      // No open dropdowns on page, nothing to do.
      return;

    if (event.key === 'Escape') {
      // User pressed Escape while having an open dropdown, probably wants it be closed.
      dropdown.removeAttribute('open');
      return;
    }

    // todo: double check if this is needed
    event.preventDefault();
    event.stopPropagation();

    const dropdownItems = dropdown.querySelectorAll<HTMLLIElement>('details.dropdown[open] > .content > ul > li');

    // Knowing document.activeElement, find the <li> that contains it
    let activeLi: HTMLLIElement, activeLiIndex: number;
    for (let i = 0; i < dropdownItems.length; i++) {
      const li = dropdownItems[i] as HTMLLIElement;
      if (!li.contains(document.activeElement)) continue;
      activeLi = li;
      activeLiIndex = i;
      break;
    }
    if (activeLi === undefined)
      // The focused element is not a list item or it's contents, but something else in the dropdown
      return;

    if (event.key === 'ArrowUp') {
      if (activeLiIndex === 0)
        // Last child is already selected
        return;
      console.log("Refucos up")
      dropdownItems[activeLiIndex - 1].querySelector<HTMLElement>(':is(a, button)').focus();
    }

    if (event.key === 'ArrowDown') {
      if (activeLiIndex === dropdownItems.length - 1)
        // First child is already selected
        return;
      console.log("Refucos down")
      dropdownItems[activeLiIndex + 1].querySelector<HTMLElement>(':is(a, button)').focus();
    }
  });
}
