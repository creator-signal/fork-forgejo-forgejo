const data = JSON.parse(document.getElementById('branch-dropdown-data').dataset.branchDropdown);

window.config.pageData.branchDropdownDataList = window.config.pageData.branchDropdownDataList || [];
window.config.pageData.branchDropdownDataList.push(data);


