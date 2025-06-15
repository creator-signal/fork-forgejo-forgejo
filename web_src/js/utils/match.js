import emojis from '../../../assets/emoji.json';

const maxMatches = 6;

function sortAndReduce(map) {
  const sortedMap = new Map(Array.from(map.entries()).sort((a, b) => a[1] - b[1]));
  return Array.from(sortedMap.keys()).slice(0, maxMatches);
}

export function matchEmoji(queryText) {
  const query = queryText.toLowerCase().replaceAll('_', ' ');
  if (!query) return emojis.slice(0, maxMatches).map((e) => e.aliases[0]);

  // results is a map of weights, lower is better
  const results = new Map();
  for (const {aliases} of emojis) {
    const mainAlias = aliases[0];
    for (const [aliasIndex, alias] of aliases.entries()) {
      const index = alias.replaceAll('_', ' ').indexOf(query);
      if (index === -1) continue;
      const existing = results.get(mainAlias);
      const rankedIndex = index + aliasIndex;
      results.set(mainAlias, existing ? existing - rankedIndex : rankedIndex);
    }
  }

  return sortAndReduce(results);
}

export function matchMention(queryText) {
  const query = queryText.toLowerCase();

  // results is a map of weights, lower is better
  const results = new Map();
  for (const obj of window.config.mentionValues ?? []) {
    const index = obj.key.toLowerCase().indexOf(query);
    if (index === -1) continue;
    const existing = results.get(obj);
    results.set(obj, existing ? existing - index : index);
  }

  return sortAndReduce(results);
}

export function matchIssue(queryText, currentIssue = null) {
  const issues = (window.config.issueValues ?? []).filter(
    (issue) => issue.number !== currentIssue,
  );
  const query = queryText.toLowerCase().trim();

  if (!query) {
    return Array.from(issues)
      .sort((a, b) => b.number - a.number)
      .slice(0, maxMatches);
  }

  const isNumber = /^\d+$/.test(query);
  const results = new Set();

  if (isNumber) {
    // Find issues/prs with number starting with the query (prefix), sorted by number ascending
    const prefixMatches = issues.filter((issue) =>
      String(issue.number).startsWith(query),
    ).sort((a, b) => a.number - b.number);

    for (const issue of prefixMatches) {
      results.add(issue);
    }
  }

  if (!isNumber || results.length < maxMatches) {
    // Fallback: find by title match, sorted by number descending
    const titleMatches = issues
      .filter((issue) =>
        issue.title.toLowerCase().includes(query),
      )
      .sort((a, b) => b.number - a.number);

    // Add only those not already in the result set
    for (const match of titleMatches) {
      results.add(match);
      if (results.size >= maxMatches) break;
    }
  }

  return Array.from(results).slice(0, maxMatches);
}
