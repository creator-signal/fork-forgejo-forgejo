import $ from 'jquery';
import {htmlEscape} from 'escape-goat';
import {emojiHTML} from './emoji.js';

const {appSubUrl} = window.config;

export function initRepoIssueSidebarList() {
  const repolink = $('#repolink').val();
  const repoId = $('#repoId').val();
  const crossRepoSearch = $('#crossRepoSearch').val() === 'true';
  const tp = $('#type').val();

  $('#new-dependency-drop-list')
    .dropdown({
      apiSettings: {
        beforeSend(settings) {
          if (!settings.urlData.query.trim()) {
            settings.url = `${appSubUrl}/${repolink}/issues/search?q={query}&type=${tp}&sort=updated`;
          } else if (crossRepoSearch) {
            settings.url = `${appSubUrl}/issues/search?q={query}&priority_repo_id=${repoId}&type=${tp}&sort=relevance`;
          } else {
            settings.url = `${appSubUrl}/${repolink}/issues/search?q={query}&type=${tp}&sort=relevance`;
          }
          return settings;
        },
        onResponse(response: Record<string, {
          id: string,
          number: number,
          title: string,
          repository: {
            full_name: string
          }
        }>) {
          const filteredResponse = {success: true, results: []};
          const currIssueId = $('#new-dependency-drop-list').data('issue-id');
          // Parse the response from the api to work with our dropdown
          for (const [_, issue] of Object.entries(response)) {
            // Don't list current issue in the dependency list.
            if (issue.id === currIssueId) {
              continue;
            }
            filteredResponse.results.push({
              name: `#${issue.number} ${issueTitleHTML(htmlEscape(issue.title))
              }<div class="text small tw-break-anywhere">${htmlEscape(issue.repository.full_name)}</div>`,
              value: issue.id,
            });
          }
          return filteredResponse;
        },
        cache: false,
      },

      fullTextSearch: true,
    });

  $('.menu button.label-exclude-item-btn').each(function () {
    $(this).on('click', function () {
      const label = this.closest('.item').querySelector('a.label-filter-item');

      if (!label) {
        return;
      }

      excludeLabel(label);
    });
  });

  // Increase surface area to include a label in the filters
  for (const labelFilterItem of document.querySelectorAll<HTMLAnchorElement>('.menu a.label-filter-item')) {
    const menuItem = labelFilterItem.closest('.item');
    menuItem.addEventListener('click', (event: MouseEvent) => {
      if (labelFilterItem === event.target || (event.target as HTMLElement).closest('.label-exclude-item-btn')) {
        return;
      }

      labelFilterItem.click();
    });
  }

  $('.menu .ui.dropdown.label-filter').on('keydown', (e: KeyboardEvent) => {
    const selectedItem = document.querySelector('.menu .ui.dropdown.label-filter .menu .item.selected');

    if (!selectedItem) {
      return;
    }

    if (e.key === 'Enter') {
      const labelElement = selectedItem.querySelector<HTMLAnchorElement>('a.label-filter-item');
      const excludeButtonIsSelected = selectedItem.querySelector('.label-exclude-item-btn.selected');

      if (excludeButtonIsSelected) {
        excludeLabel(labelElement);
      } else {
        labelElement.click();
      }
    }

    const isOnInput = (e.target as HTMLElement).matches('input');

    if (isOnInput) {
      const input = e.target as HTMLInputElement;

      if (e.key === 'ArrowRight' && isCaretAtEnd(input)) {
        selectedItem.querySelector('.label-exclude-item-btn')?.classList.add('selected');
      }

      if (e.key === 'ArrowLeft' && isCaretAtEnd(input)) {
        selectedItem.querySelector('.label-exclude-item-btn')?.classList.remove('selected');
      }
    }
  });

  $('.ui.dropdown.label-filter, .ui.dropdown.select-label').dropdown('setting', {'hideDividers': 'empty'}).dropdown('refreshItems');
}

/**
 * Render the issue's title.
 * It converts emojis and code blocks syntax into their respective HTML equivalent.
 */
export function issueTitleHTML(title: string) {
  return title.replaceAll(/:[-+\w]+:/g, (emoji) => emojiHTML(emoji.substring(1, emoji.length - 1)))
    .replaceAll(/`[^`]+`/g, (code) => `<code class="inline-code-block">${code.substring(1, code.length - 1)}</code>`);
}

/**
 * Adds or excludes a label from label filters given an element identified by a data-label-id attribute
 */
function excludeLabel(item: HTMLElement) {
  const id = item.getAttribute('data-label-id');
  const excludedId = `-${id}`;

  const params = new URLSearchParams(window.location.search);
  const labelIds = new Set((params.get('labels') ?? '').split(',').filter((id) => id.length > 0));

  if (labelIds.has(id)) {
    labelIds.delete(id);
    labelIds.add(excludedId);
  } else if (labelIds.has(excludedId)) {
    labelIds.delete(excludedId);
  } else {
    labelIds.add(excludedId);
  }

  params.set('labels', Array.from(labelIds).join(','));

  window.location.search = params.toString();
}

/**
 * Returns true if the caret is at the end of the input even if it has content
 */
function isCaretAtEnd(inputElement: HTMLInputElement) {
  const value = inputElement.value;
  return (
    inputElement.selectionStart === inputElement.selectionEnd &&
    inputElement.selectionEnd === value.length
  );
}
