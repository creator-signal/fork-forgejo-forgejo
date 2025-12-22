import type {SearchQuery} from '@codemirror/search';
import type {EditorView, Panel, ViewUpdate} from '@codemirror/view';
import {svg} from '../svg.js';

class SearchPanel implements Panel {
  searchField: HTMLInputElement;
  replaceField: HTMLInputElement;
  caseField: HTMLInputElement;
  caseLabel: HTMLLabelElement;
  reField: HTMLInputElement;
  reLabel: HTMLLabelElement;
  wordField: HTMLInputElement;
  wordLabel: HTMLLabelElement;
  dom: HTMLElement;
  query: SearchQuery;
  search: CodeMirrorSearch;

  constructor(readonly codemirrorSearch: CodeMirrorSearch, readonly view: EditorView) {
    this.search = codemirrorSearch;

    const query = (this.query = this.search.getSearchQuery(view.state));
    this.commit = this.commit.bind(this);

    this.searchField = document.createElement('input');
    this.searchField.value = query.search;
    this.searchField.name = 'search';
    this.searchField.placeholder = 'Find';
    this.searchField.ariaLabel = 'Find';
    this.searchField.classList.add('cm-textfield');
    this.searchField.setAttribute('main-field', 'true');
    this.searchField.addEventListener('keyup', this.commit);
    this.searchField.addEventListener('change', this.commit);

    this.caseField = document.createElement('input');
    this.caseField.checked = query.caseSensitive;
    this.caseField.type = 'checkbox';
    this.caseField.name = 'case_sensitive';
    this.caseField.id = 'search_case_sensitive';
    this.caseField.addEventListener('change', this.commit);
    this.caseField.addEventListener('focus', () => this.updateLabels());
    this.caseField.addEventListener('blur', () => this.updateLabels());

    this.caseLabel = document.createElement('label');
    this.caseLabel.setAttribute('for', 'search_case_sensitive');
    this.caseLabel.textContent = 'aA';

    this.reField = document.createElement('input');
    this.reField.checked = query.regexp;
    this.reField.type = 'checkbox';
    this.reField.name = 'regexp';
    this.reField.id = 'search_regexp';
    this.reField.addEventListener('change', this.commit);
    this.reField.addEventListener('focus', () => this.updateLabels());
    this.reField.addEventListener('blur', () => this.updateLabels());

    this.reLabel = document.createElement('label');
    this.reLabel.setAttribute('for', 'search_regexp');
    this.reLabel.textContent = '[.+]';

    this.wordField = document.createElement('input');
    this.wordField.checked = query.wholeWord;
    this.wordField.type = 'checkbox';
    this.wordField.name = 'by_word';
    this.wordField.id = 'search_by_word';
    this.wordField.addEventListener('change', this.commit);
    this.wordField.addEventListener('focus', () => this.updateLabels());
    this.wordField.addEventListener('blur', () => this.updateLabels());

    this.wordLabel = document.createElement('label');
    this.wordLabel.setAttribute('for', 'search_by_word');
    this.wordLabel.textContent = 'W';

    const searchFieldContainer = document.createElement('span');
    searchFieldContainer.classList.add('search-input-group');
    searchFieldContainer.replaceChildren(this.searchField, this.caseLabel, this.reLabel, this.wordLabel);

    const hiddenInputs = document.createElement('div');
    hiddenInputs.classList.add('search-hidden-inputs');
    hiddenInputs.replaceChildren(this.caseField, this.reField, this.wordField);

    const prevSearch = document.createElement('button');
    prevSearch.classList.add('secondary', 'button');
    prevSearch.type = 'button';
    prevSearch.addEventListener('click', () => {
      this.search.findPrevious(view);
    });
    prevSearch.innerHTML = svg('octicon-arrow-up');

    const nextSearch = document.createElement('button');
    nextSearch.classList.add('secondary', 'button');
    nextSearch.type = 'button';
    nextSearch.addEventListener('click', () => {
      this.search.findNext(view);
    });
    nextSearch.innerHTML = svg('octicon-arrow-down');

    const searchSection = document.createElement('div');
    searchSection.classList.add('search-section');
    searchSection.replaceChildren(searchFieldContainer, hiddenInputs, prevSearch, nextSearch);

    this.replaceField = document.createElement('input');
    this.replaceField.value = query.replace;
    this.replaceField.name = 'replace';
    this.replaceField.placeholder = 'Replace';
    this.replaceField.ariaLabel = 'replace';
    this.replaceField.classList.add('cm-textfield');
    this.replaceField.addEventListener('keyup', this.commit);
    this.replaceField.addEventListener('change', this.commit);

    const replaceButton = document.createElement('button');
    replaceButton.classList.add('secondary', 'button');
    replaceButton.type = 'button';
    replaceButton.addEventListener('click', () => {
      this.search.replaceNext(view);
    });
    replaceButton.textContent = 'Replace';

    const replaceAllButton = document.createElement('button');
    replaceAllButton.classList.add('secondary', 'button');
    replaceAllButton.type = 'button';
    replaceAllButton.addEventListener('click', () => {
      this.search.replaceAll(view);
    });
    replaceAllButton.textContent = 'Replace all';

    const replaceSection = document.createElement('div');
    replaceSection.classList.add('replace-section');
    replaceSection.replaceChildren(this.replaceField, replaceButton, replaceAllButton);

    this.dom = document.createElement('div');
    this.dom.classList.add('fj-search');
    this.dom.addEventListener('keydown', (e: KeyboardEvent) => this.keydown(e));
    this.dom.replaceChildren(searchSection, replaceSection);
  }

  commit() {
    this.updateLabels();
    const query = new this.search.SearchQuery({
      search: this.searchField.value,
      caseSensitive: this.caseField.checked,
      regexp: this.reField.checked,
      wholeWord: this.wordField.checked,
      replace: this.replaceField.value,
    });
    if (!query.eq(this.query)) {
      this.query = query;
      this.view.dispatch({effects: this.search.setSearchQuery.of(query)});
    }
  }

  keydown(e: KeyboardEvent) {
    // if (runScopeHandlers(this.view, e, "search-panel")) {
    //      e.preventDefault();
    if (e.key === 'Enter' && e.target === this.searchField) {
      e.preventDefault();
      if (e.shiftKey) {
        this.search.findPrevious(this.view);
      } else {
        this.search.findNext(this.view);
      }
    } else if (e.key === 'Enter' && e.target === this.replaceField) {
      e.preventDefault();
      this.search.replaceNext(this.view);
    }
  }

  update(update: ViewUpdate) {
    for (const tr of update.transactions) for (const effect of tr.effects) {
      if (effect.is(this.search.setSearchQuery) && !effect.value.eq(this.query)) {
        this.setQuery(effect.value);
      }
    }
  }

  setQuery(query: SearchQuery) {
    this.query = query;
    this.searchField.value = query.search;
    this.replaceField.value = query.replace;
    this.caseField.checked = query.caseSensitive;
    this.reField.checked = query.regexp;
    this.wordField.checked = query.wholeWord;
    this.updateLabels();
  }

  updateLabels() {
    this.caseLabel.classList.toggle('active', this.caseField.checked);
    this.caseLabel.classList.toggle('focused', this.caseField === document.activeElement);
    this.reLabel.classList.toggle('active', this.reField.checked);
    this.reLabel.classList.toggle('focused', this.reField === document.activeElement);
    this.wordLabel.classList.toggle('active', this.wordField.checked);
    this.wordLabel.classList.toggle('focused', this.wordField === document.activeElement);
  }

  mount() {
    this.searchField.select();
  }

  get pos() {
    return 80;
  }

  get top() {
    return true;
  }
}

export function searchPanel(
  codemirrorSearch: CodeMirrorSearch,
): (view: EditorView) => Panel {
  return (view) => {
    return new SearchPanel(codemirrorSearch, view);
  };
}
