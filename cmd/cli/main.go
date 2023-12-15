package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/viper"

	traceconsumerv1 "github.com/codecomet-io/api-proto/gen/proto/v1"
	"github.com/codecomet-io/api-proto/gen/proto/v1/traceconsumerv1connect"
	"github.com/codecomet-io/codecomet-cli/pkg/testobs"
)

func shiftSlice(s []string, pos int, val string) []string {
	s = append(s, "")
	copy(s[pos+1:], s[pos:])
	s[pos] = val
	return s
}

// The codecomet CLI

func main() {
	if len(os.Args) < 3 {
		panic("need a go test command to prefix")
	}
	viper.AutomaticEnv()
	viper.SetEnvPrefix("CODECOMET")

	remoteServer := viper.GetString("REMOTE_SERVER")
	if remoteServer == "" {
		remoteServer = "https://codecomet.io/api"
	}
	client := traceconsumerv1connect.NewTraceServiceClient(
		http.DefaultClient,
		remoteServer,
		connect.WithSendGzip(),
	)

	args := os.Args[1:]
	if args[0] != "go" || args[1] != "test" {
		panic("first two arguments need to be `go test`")
	}
	var coverProfileFile string
	// now search through the arguments and avoid passing our needed arguments more than once
	var jsonFound, coverProfileFound bool
	for _, arg := range args {
		if arg == "-json" {
			jsonFound = true
			continue
		}
		if strings.HasPrefix(arg, "-coverprofile=") {
			coverProfileFound = true
			coverProfileFile = strings.TrimPrefix(arg, "-coverprofile=")
			continue
		}
	}
	if !jsonFound {
		args = shiftSlice(args, 2, "-json")
	}
	if !coverProfileFound {
		cf, err := os.CreateTemp("", "cover")
		if err != nil {
			panic(err)
		}
		coverProfileFile = cf.Name()
		cf.Close()
		args = shiftSlice(args, 2, "-coverprofile="+coverProfileFile)
	}
	cmd := exec.Command(args[0], args[1:]...)

	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		// XXX: remove these panics and fail more gracefully.
		panic(err)
	}

	cisystem := testobs.AutodetectCI()
	isr := &traceconsumerv1.IngestSuiteRequest{
		Suite: &traceconsumerv1.TestSuiteRun{
			CiSystem: string(cisystem),
			Output:   []byte(out.String()),
		},
	}

	cf, err := os.Open(coverProfileFile)
	if err != nil {
		// XXX: fail more gracefully. this is a user-facing tool.
		panic(err)
	}
	cbts, err := io.ReadAll(cf)
	if err != nil {
		panic(err)
	}
	isr.Suite.CoverageInfo = cbts

	req := connect.NewRequest(isr)
	req.Header().Add("Api-Key", viper.GetString("API_KEY"))
	resp, err := client.IngestSuiteRun(context.Background(), req)
	if err != nil {
		panic(err)
	}
	fmt.Println("response is", resp)

}
