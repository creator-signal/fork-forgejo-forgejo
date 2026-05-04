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
window.config.pageData = window.config.pageData || {};
window.addEventListener('load', function(){
    const mentionValues = [];
    const mentionKeys = ['key', 'value', 'name', 'avatar', 'fullname'];
    document.querySelectorAll('.participants .participant').forEach((participant) => {
      let mention_participant = {};
      Object.entries(participant.dataset).forEach(([name, value]) => {
        if(mentionKeys.includes(name)){
          mention_participant[name] = value;
        }
      });
      mentionValues.push(mention_participant);
    });

    document.querySelectorAll('.assignees .assignee').forEach((assignee) => {
      let mention_assignee = {};
      Object.entries(assignee.dataset).forEach(([name, value]) => {
        if(mentionKeys.includes(name)){
          mention_assignee[name] = value;
        }
      });
      mentionValues.push(mention_assignee);
    });

    const teamorg = document.querySelector('.mentionableteams').dataset;
    document.querySelectorAll('.mentionableteams .team').forEach((team) => {
      mentionValues.push({
        'key': teamorg['org'] + '/' + team.dataset['name'],
        'value': teamorg['org'] + '/' + team.dataset['name'],
        'name': teamorg['org'] + '/' + team.dataset['name'],
        'avatar': teamorg['avatar']
      });
    });

    if (mentionValues){
      window.config['mentionValues'] = (mentionValues.length) ? mentionValues : null;
    }
});
