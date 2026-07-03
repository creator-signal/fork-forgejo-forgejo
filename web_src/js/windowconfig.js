function windowErrorHandler(e) {
  window._globalHandlerErrors = window._globalHandlerErrors || [];
  window._globalHandlerErrors.push(e);
}

window.addEventListener('error', windowErrorHandler);
window.addEventListener('unhandledrejection', windowErrorHandler);
const dataset = document.documentElement.dataset;
window.config = {
  appUrl: dataset.appurl,
  appSubUrl: dataset.appsuburl,
  assetVersionEncoded: encodeURIComponent(dataset.assetversion), // will be used in URL construction directly
  assetUrlPrefix: dataset.asseturlprefix,
  runModeIsProd: (dataset.runmodelisprod === 'true'),
  customEmojis: new Set(JSON.parse(dataset.customemojis)),
  pageData: JSON.parse(dataset.pagedata),
  notificationSettings: JSON.parse(dataset.notificationsettings),
  enableTimeTracking: (dataset.enabletimetracking === 'true'),
  mermaidMaxSourceCharacters: parseInt(dataset.mermaidmaxsourcecharacters),
  i18n: ('i18n' in dataset) ? JSON.parse(dataset.i18n) : {},
};
window.config.pageData = window.config.pageData || {};


function mentionLoader(){
  const mentionValues = [];
  const mentionKeys = ['key', 'value', 'name', 'avatar', 'fullname'];
  for (const participant of document.querySelectorAll('.participants .participant')) {
    const mentionParticipant = {};

    for (const [name, value] of Object.entries(participant.dataset)) {
      if (mentionKeys.includes(name)) {
        mentionParticipant[name] = value;
      }
    }

    mentionValues.push(mentionParticipant);
  }

  for (const assignee of document.querySelectorAll('.assignees .assignee')) {
    const mentionAssignee = {};

    for (const [name, value] of Object.entries(assignee.dataset)) {
      if (mentionKeys.includes(name)) {
        mentionAssignee[name] = value;
      }
    }

    mentionValues.push(mentionAssignee);
  }

  const teamOrg = document.querySelector('.mentionableteams');
  for (const team of document.querySelectorAll('.mentionableteams .team')) {
    mentionValues.push({
      key: `${teamOrg.dataset.org}/${team.dataset.name}`,
      value: `${teamOrg.dataset.org}/${team.dataset.name}`,
      name: `${teamOrg.dataset.org}/${team.dataset.name}`,
      avatar: teamOrg.dataset.avatar,
    });
  }

  if (mentionValues) {
    window.config['mentionValues'] = (mentionValues.length) ? mentionValues : null;
  }
}

window.addEventListener('load', mentionLoader);
