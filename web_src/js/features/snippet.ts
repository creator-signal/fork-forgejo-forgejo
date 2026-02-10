import {showTemporaryTooltip} from '../modules/tippy.js';
import {initRepoCloneLink} from './repo-common.js';
import {clippie} from 'clippie';

const {i18n} = window.config;

function addSnippetFileButtonClicked(): void {
  const template = document.getElementById('file-field-template') as HTMLTemplateElement;
  const clone = document.importNode(template.content, true);

  let currentID = 0;
  while (true) {
    if (document.querySelector(`input[name="file-name-${currentID}"]`) === null) {
      break;
    } else {
      currentID += 1;
    }
  }

  clone.querySelector('[data-template-name="field"]').setAttribute('id', `file-field-${currentID}`);
  clone.querySelector('[data-template-name="name-label"]').setAttribute('for', `file-name-${currentID}`);
  clone.querySelector('[data-template-name="name-input"]').setAttribute('name', `file-name-${currentID}`);
  clone.querySelector('[data-template-name="content-label"]').setAttribute('for', `file-content-${currentID}`);
  clone.querySelector('[data-template-name="content-input"]').setAttribute('name', `file-content-${currentID}`);

  const deleteButton = clone.querySelector('[data-template-name="delete-button"]') as HTMLButtonElement;
  deleteButton.setAttribute('data-file-id', currentID.toString());
  deleteButton.addEventListener('click', deleteFileButtonClicked);

  document.getElementById('file-field-container').append(clone);
}

function deleteFileButtonClicked(event: Event): void {
  const fileID = (event.target as HTMLButtonElement).getAttribute('data-file-id');
  const fileField = document.getElementById(`file-field-${fileID}`);
  fileField.remove();
}

function initAddSnippetFileButton(): void {
  const button = document.getElementById('add-snippet-file-button');

  if (button !== null) {
    button.addEventListener('click', addSnippetFileButtonClicked);
  }
}

function initSnippetCopyContent(): void {
  for (const elem of document.querySelectorAll('span[data-snippet-copy-content]')) {
    elem.addEventListener('click', async (event: Event) => {
      const target = event.currentTarget as HTMLSpanElement;

      if (target.classList.contains('is-loading')) {
        return;
      }

      target.classList.add('is-loading', 'loading-icon-2px');

      const fileID = target.getAttribute('data-snippet-copy-content');

      const lineEls = document.getElementById(`snippet-file-view-${fileID}`).querySelectorAll('.lines-code');
      const text = Array.from(lineEls, (el) => el.textContent).join('');

      const success = await clippie(text);

      if (success) {
        showTemporaryTooltip(target, i18n.copy_success);
      } else {
        showTemporaryTooltip(target, i18n.copy_error);
      }

      target.classList.remove('is-loading', 'loading-icon-2px');
    });
  }
}

function initSnippetFileDeleteButtons(): void {
  for (const elem of document.querySelectorAll('#edit-snippet-form > * button[data-template-name="delete-button"]')) {
    elem.addEventListener('click', deleteFileButtonClicked);
  }
}

export function initSnippets(): void {
  initAddSnippetFileButton();
  initSnippetCopyContent();
  initSnippetFileDeleteButtons();

  if (window.location.pathname.startsWith('/snippets')) {
    initRepoCloneLink();
  }
}
