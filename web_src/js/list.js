
function pollingOk() {
	return document.visibilityState === 'visible' && noActiveDropdowns();
}

// Intent: If the "Actor" or "Status" dropdowns are currently open and being navigated, or the workflow dispatch
// dropdown form is open, the htmx refresh would replace them with closed dropdowns.  Instead this prevents the list
// refresh from occurring while those dropdowns are open.
//
// Can't inline this into the `hx-trigger` above because using a left-brace ('[') breaks htmx's trigger parsing.
function noActiveDropdowns() {
	if (document.querySelector('[aria-expanded=true]') !== null)
		return false;
	const dropdownForm = document.querySelector('#branch-dropdown-form');
	if (dropdownForm !== null && dropdownForm.checkVisibility())
		return false;
	return true;
}

document.addEventListener("visibilitychange", () => {
  if (pollingOk()) {
    htmx.trigger("#repo-actions-list", "poll-now");
  }
});

setInterval(() => {
  if (pollingOk()) {
    htmx.trigger("#repo-actions-list", "poll-now");
  }
}, 30000);
