document.body.addEventListener("htmx:afterSettle", (event) => {
	const form = event.target;

	if (!form.matches("form[data-refocus]"))
		return;

	form.querySelector(form.dataset.refocus)?.focus();
});
