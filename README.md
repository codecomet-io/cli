## The CodeComet CLI

The CodeComet CLI collects metrics about your tests and uploads them to the CodeComet web app.


### CI systems supported

We support Github Actions and CircleCI currently. We will add support for other CI systems. If your CI system is not on this list, you can still use CodeComet by setting a few environment variables:

- `CODECOMET_BRANCH` - set to the branch name
- `CODECOMET_REPOSITORY` - set to the repository name (without the owner). For example, for this repo, the name is `cli`.
- `CODECOMET_REPOSITORY_OWNER` - set to the owner name. For example, for this repo, the owner is `codecomet-io`
- `CODECOMET_COMMIT_HASH` - set to the current commit hash
- `CODECOMET_SEQ_BUILD_ID` - set to a sequential build ID. This should typically be an increasing number, starting with 1 (your very first build). Every subsequent run should increase this number. It's important to set this to different values every time you run tests so that CodeComet can separate your test suites and keep track of their runs accordingly.

### How to use CLI

To use, simply prefix your "go test" or "pytest" or other test commands with `codecomet`

You can pass flags to the codecomet executable, such as the suite name (it will default to your folder name) or the suite run ID (which will default to the run ID in your CI system).

`-s, --suite  Provide a name for this test suite. Use the same test suite name for test runs you want to group together.`
`-r, --runid  Provide a run ID. Defaults to your CI system's run ID, if any. `


For example:

```
codecomet -s MyBackendTests -- go test -json -coverprofile=cover.out ./...
```

The above will run all tests in `./...` and group them logically as a "suite name" named MyBackendTests. Since the suite Run ID is not provided, it is assumed that you are using one of our supported CI systems, or you have set up a run ID yourself with the `CODECOMET_SEQ_BUILD_ID` environment variable.

```
codecomet -r LOC2 -- go test $(go list ./... | grep -v wasm)
```

The above will run all tests in the provided packages except for a package named "wasm". The run ID used is `LOC2`. No suite name is provided here, so one will be automatically created from the base directory. You may want to provide a suite name to make it clear which groups of tests should be grouped together as a "suite". In the above example, the base directory that this command is being run in is the same for all tests, so they will all share a suite name and suite run ID.


### Environment variables

In order to upload results to our servers, set the CODECOMET_API_KEY environment variable to your API Key. You can generate your API Key from the settings menu in the CodeComet web app.