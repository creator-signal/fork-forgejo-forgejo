package cron

const (
	TaskDeleteInactiveAccounts           = "delete_inactive_accounts"
	TaskDeleteRepoArchives               = "delete_repo_archives"
	TaskGitGcRepos                       = "git_gc_repos"
	TaskResyncAllSshkeys                 = "resync_all_sshkeys"
	TaskResyncAllSshprincipals           = "resync_all_sshprincipals"
	TaskResyncAllHooks                   = "resync_all_hooks"
	TaskReinitMissingRepos               = "reinit_missing_repos"
	TaskDeleteMissingRepos               = "delete_missing_repos"
	TaskDeleteGeneratedRepositoryAvatars = "delete_generated_repository_avatars"
	TaskDeleteOldActions                 = "delete_old_actions"
	TaskUpdateChecker                    = "update_checker"
	TaskDeleteOldSystemNotices           = "delete_old_system_notices"
	TaskGcLfs                            = "gc_lfs"
	TaskRebuildIssueIndexer              = "rebuild_issue_indexer"
	TaskStopZombieTasks                  = "stop_zombie_tasks"
	TaskStopEndlessTasks                 = "stop_endless_tasks"
	TaskCancelAbandonedJobs              = "cancel_abandoned_jobs"
	TaskStartScheduleTasks               = "start_schedule_tasks"
	TaskCleanupActions                   = "cleanup_actions"
	TaskCleanupOfflineRunners            = "cleanup_offline_runners"
	TaskUpdateMirrors                    = "update_mirrors"
	TaskRepoHealthCheck                  = "repo_health_check"
	TaskCheckRepoStats                   = "check_repo_stats"
	TaskArchiveCleanup                   = "archive_cleanup"
	TaskSyncExternalUsers                = "sync_external_users"
	TaskDeletedBranchesCleanup           = "deleted_branches_cleanup"
	TaskUpdateMigrationPosterId          = "update_migration_poster_id"
	TaskCleanupHookTaskTable             = "cleanup_hook_task_table"
	TaskCleanupPackages                  = "cleanup_packages"
)

const (
	scheduleMidnight = "@midnight"
	scheduleAnnually = "@annually"
	scheduleEvery72h = "@every 72h"
)
