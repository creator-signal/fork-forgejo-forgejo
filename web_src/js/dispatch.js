window.addEventListener('load', () => {
	const dropdown = $('#workflow_dispatch_dropdown');
	const menu = dropdown.find('> .menu');
	$(document.body).on('click', (ev) => {
		if (!dropdown[0].contains(ev.target) && menu.hasClass('visible')) {
			menu.transition({ animation: 'slide down out', duration: 200, queue: false });
		}
	});
	dropdown.on('click', (ev) => {
		const inMenu = $(ev.target).closest(menu).length !== 0;
		if (inMenu) return;
		ev.stopPropagation();
		if (menu.hasClass('visible')) {
			menu.transition({ animation: 'slide down out', duration: 200, queue: false });
		} else {
			menu.transition({ animation: 'slide down in', duration: 200, queue: true });
		}
	});
});
