const dataset = document.getElementById('dashboard-repo-list-data').dataset;

const data = {
	...window.config.pageData.dashboardRepoList, // runtime values
};

for (const [key, value] of Object.entries(dataset)) {
	let parsedValue = value;

	// boolean conversion
	if (value === 'true') {
		parsedValue = true;
	} else if (value === 'false') {
		parsedValue = false;
	}
	// numeric conversion
	else if (/^\d+$/.test(value)) {
		parsedValue = Number(value);
	}
	// JSON arrays/objects
	else if (
		(value.startsWith('[') && value.endsWith(']')) ||
		(value.startsWith('{') && value.endsWith('}'))
	) {
		try {
			parsedValue = JSON.parse(value);
		} catch {
			console.error("could not parse dataset attribute as Json", key, value);
            continue;
		}
	}

	data[key] = parsedValue;
}

window.config.pageData.dashboardRepoList = data;
