package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	traceconsumerv1 "github.com/codecomet-io/api-proto/gen/proto/v1"
	"github.com/codecomet-io/api-proto/gen/proto/v1/traceconsumerv1connect"
	"github.com/codecomet-io/cli/pkg/testobs"
)

var CoverageFile string

func init() {
	rootCmd.AddCommand(junitCmd)
	junitCmd.Flags().StringVarP(&CoverageFile, "coverage", "c", "", "coverage file")
}

var junitCmd = &cobra.Command{
	Use:   "junit",
	Short: "Consume JUnit file. The only argument is the xml file.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("No junit output file was provided")
			os.Exit(1)
		}
		junitfile, err := os.Open(args[0])
		if err != nil {
			fmt.Printf("Unable to open file: %v\n", err)
			os.Exit(1)
		}
		defer junitfile.Close()

		remoteServer := viper.GetString("REMOTE_SERVER")
		if remoteServer == "" {
			remoteServer = "https://app.codecomet.io/api"
		}
		client := traceconsumerv1connect.NewTraceServiceClient(
			http.DefaultClient,
			remoteServer,
			connect.WithSendGzip(),
		)
		civars := testobs.AutodetectCI()
		populateSuiteNameAndRunID(civars)

		outBytes, err := io.ReadAll(junitfile)
		if err != nil {
			fmt.Printf("Unable to read file bytes: %v\n", err)
			return
		}

		isr := &traceconsumerv1.IngestSuiteRequest{
			Run: &traceconsumerv1.TestSuiteRun{
				SuiteName:    SuiteName,
				BuildTag:     BuildTag,
				CiSystem:     string(civars.System),
				Repository:   civars.RepositoryOwner + "/" + civars.Repository,
				Branch:       civars.Branch,
				CommitHash:   civars.CommitHash,
				Output:       outBytes,
				OutputFormat: "junit-xml",
				// skip pass/fail info. This will be obtained from the xml.
			},
		}

		var cbts []byte
		cf, err := os.Open(CoverageFile)
		if err != nil {
			fmt.Printf("Unable to open coverage file, will skip coverage for now: %v\n", err)
		} else {
			cbts, err = io.ReadAll(cf)
			if err != nil {
				fmt.Printf("Unable to read coverage bytes: %v\n", err)
			}
		}
		cf.Close()
		isr.Run.CoverageInfo = cbts
		req := connect.NewRequest(isr)
		req.Header().Add("Api-Key", viper.GetString("API_KEY"))
		resp, err := client.IngestTestSuiteRun(context.Background(), req)
		if err != nil {
			fmt.Printf("Call to CodeComet failed: %v\n", err)
		}
		fmt.Printf("CodeComet returned %v\n", resp)
	},
}
