import {createCodemirror, updateCodemirrorFilename} from './codemirror.ts';
import {showTemporaryTooltip} from '../modules/tippy.js';
import {initRepoCloneLink} from './repo-common.js';
import {clippie} from 'clippie';

const {i18n} = window.config;

function checkDuplicates(): void {
  const inputList = document.querySelectorAll('#edit-snippet-form > * input[data-template-name="name-input"]') as NodeListOf<HTMLInputElement>;
  const submitButton = document.getElementById('submit-snippet-button') as HTMLButtonElement | null;

  if (submitButton === null) {
    return;
  }

  const duplicateSet = new Set<string>();
  const nameSet = new Set<string>();

  for (const input of inputList) {
    if (input.value === '') {
      continue;
    }

    if (nameSet.has(input.value)) {
      duplicateSet.add(input.value);
    }

    nameSet.add(input.value);
  }

  for (const input of inputList) {
    const errorBox = document.getElementById(input.getAttribute('data-duplicate-error-id'));

    errorBox.classList.toggle('tw-hidden', !duplicateSet.has(input.value));
  }

  submitButton.disabled = duplicateSet.size !== 0;
}

function updateDeleteButtonVisible(): void {
  const buttons = document.querySelectorAll('#edit-snippet-form > * button[data-template-name="delete-button"]');

  if (buttons.length === 1) {
    buttons[0].classList.add('tw-hidden');
    return;
  }

  for (const currentButton of buttons) {
    currentButton.classList.remove('tw-hidden');
  }
}

function updateAddSnippetButtonEnabled(): void {
  const addButton = document.getElementById('add-snippet-file-button') as HTMLButtonElement | null;
  const maxFilesInput = document.getElementById('max-snippet-files') as HTMLInputElement | null;

  if (addButton === null || maxFilesInput === null) {
    return;
  }

  const fileCount = document.querySelectorAll('#edit-snippet-form > * #file-field-container > div').length;
  const maxFiles = parseInt(maxFilesInput.value);

  addButton.disabled = fileCount >= maxFiles;
}

async function setupFileEditor(element: Element, id: number): Promise<void> {
  const nameInput = element.querySelector('[data-template-name="name-input"]') as HTMLInputElement;
  const deleteButton = element.querySelector('[data-template-name="delete-button"]') as HTMLButtonElement;
  const textarea = element.querySelector('textarea') as HTMLTextAreaElement;

  element.setAttribute('id', `file-field-${id}`);
  element.querySelector('[data-template-name="name-label"]').setAttribute('for', `file-name-${id}}`);
  nameInput.setAttribute('name', `file-name-${id}`);
  nameInput.setAttribute('data-duplicate-error-id', `duplicate-error-${id}`);
  element.querySelector('[data-template-name="content-label"]').setAttribute('for', `file-content-${id}`);
  textarea.setAttribute('name', `file-content-${id}`);
  element.querySelector('[data-template-name="duplicate-error"]').setAttribute('id', `duplicate-error-${id}`);

  deleteButton.setAttribute('data-file-id', id.toString());
  deleteButton.addEventListener('click', () => {
    element.remove();
    updateDeleteButtonVisible();
    updateAddSnippetButtonEnabled();
  });

  const editor = await createCodemirror(textarea, nameInput.value, {});

  nameInput.addEventListener('change', (event) => {
    const target = event.target as HTMLInputElement;
    updateCodemirrorFilename(editor, target.value);
    checkDuplicates();
  });
}

async function addSnippetFileButtonClicked(): Promise<void> {
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

  await setupFileEditor(clone.querySelector('[data-template-name="field"]'), currentID);

  document.getElementById('file-field-container').append(clone);

  updateDeleteButtonVisible();
  updateAddSnippetButtonEnabled();
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

async function initSnippetFileEditors(): Promise<void> {
  const promiseList: Promise<void>[] = [];
  let currentID = 0;

  for (const elem of document.querySelectorAll('#edit-snippet-form > * #file-field-container > div')) {
    promiseList.push(setupFileEditor(elem, currentID));
    currentID += 1;
  }

  await Promise.all(promiseList);

  updateAddSnippetButtonEnabled();
  updateDeleteButtonVisible();
  checkDuplicates();
}

export function initSnippets(): void {
  if (!window.location.pathname.startsWith('/snippets')) {
    return;
  }

  initAddSnippetFileButton();
  initSnippetCopyContent();
  initSnippetFileEditors();
  initRepoCloneLink();
}
