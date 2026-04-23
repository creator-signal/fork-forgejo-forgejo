window.addEventListener('error', function(e) {window._globalHandlerErrors=window._globalHandlerErrors||[]; window._globalHandlerErrors.push(e);});
window.addEventListener('unhandledrejection', function(e) {window._globalHandlerErrors=window._globalHandlerErrors||[]; window._globalHandlerErrors.push(e);});
const dataset = document.documentElement.dataset;
window.config = {
	appUrl: dataset.appurl,
	appSubUrl: dataset.appsuburl,
	assetVersionEncoded: encodeURIComponent(dataset.assetversion), // will be used in URL construction directly
	assetUrlPrefix: dataset.asseturlprefix,
	runModeIsProd: (dataset.runmodelisprod == "true"),
	customEmojis: new Set(JSON.parse(dataset.customemojis)),
	pageData: JSON.parse(dataset.pagedata),
	notificationSettings: JSON.parse(dataset.notificationsettings), 
	enableTimeTracking: (dataset.enabletimetracking == "true"),
	mermaidMaxSourceCharacters: parseInt(dataset.mermaidmaxsourcecharacters),
	i18n: JSON.parse(dataset.i18n),
};
if(dataset.mentionValues){
    window.config['mentionValues'] = JSON.parse(dataset.mentionValues)
}

window.config.pageData = window.config.pageData || {};
