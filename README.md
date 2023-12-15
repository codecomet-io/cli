The CodeComet Go CLI/Agent

### How to use CLI

Simply prefix your `go test` command with `codecomet`.

For example, if your command is:

`go test $(go list ./... | grep -v wasm)`

Change this to:

`codecomet go test $(go list ./... | grep -v wasm)`

### Environment variables

In order to upload results to our servers, set the CODECOMET_API_KEY environment variable to your API Key.