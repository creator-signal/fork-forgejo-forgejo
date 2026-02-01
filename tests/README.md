# Forgejo Tests

- `cmd`: command line tools to help around testing
- `e2e`: end-to-end tests with playwright
- **`forgery`: helper library for integration tests**
- `fuzz`: fuzz tests
- `gitea-*`: legacy git data used by some integration tests
- `integration`: legacy integration tests (sequential)
- **`integration_exp`: new integration tests (aims at being parallelisable)**
- `testdata`: legacy testdata used by integration tests (should be moved to `integration_exp/testdata/TestName/...`)

## Current state

Due to the issues with the `integration` package (listed in forgejo/discussions#170), a new `integration_exp` package was created.

The first goal is to convert some tests from `integration` to `integration_exp`.

Once enough tests were converted (ensuring that the `forgery` helpers are good enough), we should rename `integration` to `integration_legacy` and `integration_exp` to `integration`.

### Converting tests to `integration_exp`

Rough steps for the conversion:

- move the file to `integration_exp` (in a dedicated commit to ensure that git understands the rename)
- remove all occurences of `tests.PrepareTestEnv`
- replace the `onApplicationRun` closure with:

```go
t.Parallel()
fgi := forgery.SharedInstance(t)
sess := fgi.Session()
```

- replace all usages of `unittest.*` and `tests.*` with `forgery.*` equivalents

### Which tests to convert

The following tests are currently blocking forgejo/forgejo!10834 and should be converted (conflicts with https://codeberg.org/forgejo/forgejo/pulls/10397):

- [x] `api_issue_config_test.go`
  - TestAPIRepoGetIssueConfig
  - TestAPIRepoIssueConfigPaths
  - TestAPIRepoValidateIssueConfig
- [ ] `empty_repo_test.go`
  - TestEmptyRepoAddFile
  - TestEmptyRepoUploadFile
  - TestEmptyRepoAddFileByAPI
- [ ] `api_repo_git_hook_test.go`
  - TestAPIListGitHooks
  - TestAPIGetGitHook

Ideally all tests should be converted at some point.

## Issues with the current `integration` package

- they depend on a `gitea` binary (which must be updated manually: slow and error prone)
- tests are slow (edit: the `-race` flag is responsible for a 10x slowness...)
- git blob among source file [is not nice](https://codeberg.org/forgejo/forgejo/issues/3820#issuecomment-1838749)
- some tests are wrongly setup ([git hooks not executable](https://codeberg.org/forgejo/forgejo/pulls/10834))
- scanning the CI logs to identify the failing test is cumbersome
