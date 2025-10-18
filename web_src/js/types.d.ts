declare const __webpack_public_path__: string;

interface Window {
  config: WindowConfig,
}

type Integer = number;

interface WindowConfig {
  appUrl: string;
  appSubUrl: string,
  assetVersionEncoded: string,
  assetUrlPrefix: string,
  runModeIsProd: boolean,
  customEmojis: Set<string>,
  csrfToken: string,
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  pageData: PageData | {},
  notificationSettings: Record<string, Integer>,
  enableTimeTracking: boolean,
  mentionValues?: MentionData[],
  mermaidMaxSourceCharacters: Integer,
  i18n: {
    copy_success: string,
    copy_error: string,
    error_occurred: string,
    network_error: string,
    remove_label_str: string,
    modal_confirm: string,
    modal_cancel: string,
    more_items: string,
    incorrect_root_url: string,
  }
}

interface PageData {
  DATETIMESTRINGS: {
    FUTURE: string,
    NOW: string,
    'relativetime.1day': string,
    'relativetime.1month': string,
    'relativetime.1week': string,
    'relativetime.1year': string,
    'relativetime.2days': string,
    'relativetime.2months': string,
    'relativetime.2weeks': string,
    'relativetime.2years': string,
  },
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  PLURALSTRINGS_FALLBACK: PluralStringsDefinitions | {},
  PLURALSTRINGS_LANG: PluralStringsDefinitions,
  PLURAL_RULE_FALLBACK: Integer,
  PLURAL_RULE_LANG: Integer,
  pullRequestMergeForm?: unknown,
  branchDropdownDataList?: unknown,
  diffFileInfo?: unknown,
  dashboardRepoList?: unknown,
  prReview?: unknown,
  citationFileContent?: unknown,
  repoLink?: unknown,
  repoActivityTopAuthors?: unknown,
  adminUserListSearchForm?: unknown,
}

interface PluralStringsDefinitions {
  'relativetime.days': [string, string],
  'relativetime.hours': [string, string],
  'relativetime.mins': [string, string],
  'relativetime.months': [string, string],
  'relativetime.weeks': [string, string],
  'relativetime.years': [string, string],
}

interface MentionData {
  key: string,
  value: string,
  name: string,
  fullname: string,
  avatar: string,
}

declare module '*.vue' {
  import Vue from 'vue';
  export default Vue;
}
