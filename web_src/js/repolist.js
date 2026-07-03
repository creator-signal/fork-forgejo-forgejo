const dataset = document.getElementById('dashboard-repo-list-data').dataset;

const data = {
  ...window.config.pageData.dashboardRepoList, // runtime values
};

for (const [key, value] of Object.entries(dataset)) {
  let parsedValue = value;

  if (value === 'true') { // boolean conversion
    parsedValue = true;
  } else if (value === 'false') {
    parsedValue = false;
  } else if (/^\d+$/.test(value)) { // numeric conversion
    parsedValue = Number(value);
  } else if ( // JSON arrays/objects
    (value.startsWith('[') && value.endsWith(']')) ||
    (value.startsWith('{') && value.endsWith('}'))
  ) {
    try {
      parsedValue = JSON.parse(value);
    } catch {
      console.error('could not parse dataset attribute as Json', key, value);
      continue;
    }
  }

  data[key] = parsedValue;
}

window.config.pageData.dashboardRepoList = data;
