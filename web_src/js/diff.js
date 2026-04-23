// Default to true if unset
const diffTreeVisible = localStorage?.getItem('diff_file_tree_visible') !== 'false';
const diffTreeBtn = document.querySelector('.diff-toggle-file-tree-button');
const diffTreeIcon = `.octicon-sidebar-${diffTreeVisible ? 'expand' : 'collapse'}`;
diffTreeBtn.querySelector(diffTreeIcon).classList.remove('tw-hidden');
diffTreeBtn.setAttribute('data-tooltip-content', diffTreeBtn.getAttribute(diffTreeVisible ? 'data-hide-text' : 'data-show-text'));

const el = document.getElementById('diff-data');
const diff = el ? JSON.parse(el.dataset.diff) : null;

if (diff) {
  let diffFileInfo = window.config.pageData.diffFileInfo || {
    files: [],
    fileTreeIsVisible: false,
    fileListIsVisible: false,
    isLoadingNewData: false,
    selectedItem: '',
  };

  diffFileInfo = Object.assign(diffFileInfo, diff);
  diffFileInfo.files.push(...diff.files);

  window.config.pageData.diffFileInfo = diffFileInfo;
}

if (diffTreeVisible) document.getElementById('diff-file-tree').classList.remove('tw-hidden');
