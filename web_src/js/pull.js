const mergeData = document.getElementById("merge-data").dataset;

const defaultMergeTitle = mergeData.defaultMergeTitle;
const defaultSquashMergeTitle = mergeData.defaultSquashMergeTitle;
const defaultSquashMessageFieldText = mergeData.defaultSquashMessageFieldText;
const defaultMergeMessage = mergeData.defaultMergeMessage;
const defaultSquashMergeMessage = mergeData.defaultSquashMergeMessage;

const mergeForm = {};

for (const [key, value] of Object.entries(mergeData)) {
	// Pick only dataset keys starting with "mergeForm"
	if (key.startsWith('mergeForm')) {
		// Remove the prefix
		const stripped = key.slice('mergeForm'.length);

		// Lowercase first letter
		const finalKey =
			stripped.charAt(0).toLowerCase() + stripped.slice(1);

		mergeForm[finalKey] = value;
	}
}

const generalHideAutoMerge = mergeForm.canMergeNow && mergeForm.allOverridableChecksOk; // if this pr can be merged now, then hide the auto merge

mergeForm['mergeStyles'] = [
	{
		'name': 'merge',
		'mergeTitleFieldText': defaultMergeTitle,
		'mergeMessageFieldText': defaultMergeMessage,
		'hideAutoMerge': generalHideAutoMerge,
	},
	{
		'name': 'rebase',
		'hideMergeMessageTexts': true,
		'hideAutoMerge': generalHideAutoMerge,
	},
	{
		'name': 'rebase-merge',
		'mergeTitleFieldText': defaultMergeTitle,
		'mergeMessageFieldText': defaultMergeMessage,
		'hideAutoMerge': generalHideAutoMerge,
	},
	{
		'name': 'squash',
		'mergeTitleFieldText': defaultSquashMergeTitle,
		'mergeMessageFieldText': defaultSquashMessageFieldText,
		'hideAutoMerge': generalHideAutoMerge,
	},
	{
		'name': 'fast-forward-only',
		'hideMergeMessageTexts': true,
		'hideAutoMerge': generalHideAutoMerge,
	},
	{
		'name': 'manually-merged',
		'hideMergeMessageTexts': true,
		'hideAutoMerge': true,
	}
];

const mergeStyleMap = Object.fromEntries(
	mergeForm.mergeStyles.map(style => [style.name, style]),
);

for (const [key, value] of Object.entries(mergeData)) {
	// Handle mergeStyles-*
	if (key.startsWith('mergeStyles')) {
		// Example:
		// mergeStylesMergeAllowed
		// mergeStylesMergeTextDoMerge

		const stripped = key.slice('mergeStyles'.length);

		// Extract:
		// MergeAllowed => mergestylename + propertyname
		const match = stripped.match(
			/^(.+?)(Allowed|TextDoMerge|MergeTitleFieldText|HideMergeMessageTexts|MergeMessageFieldText|HideMergeMessageTexts|HideAutoMerge)$/
		);

		if (!match) continue;

		const [, styleNameRaw, propRaw] = match;

		const styleName =
			styleNameRaw.charAt(0).toLowerCase() +
			styleNameRaw.slice(1);

		const prop =
			propRaw.charAt(0).toLowerCase() +
			propRaw.slice(1);

		const style = mergeStyleMap[styleName];

		if (style) {
			// Optional boolean conversion
			if (value === 'true') {
				style[prop] = true;
			} else if (value === 'false') {
				style[prop] = false;
			} else {
				style[prop] = value;
			}
		}
	}
}
window.config.pageData.pullRequestMergeForm = mergeForm;
