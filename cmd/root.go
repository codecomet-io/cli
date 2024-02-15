package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"
	traceconsumerv1 "github.com/codecomet-io/api-proto/gen/proto/v1"
	"github.com/codecomet-io/api-proto/gen/proto/v1/traceconsumerv1connect"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/codecomet-io/cli/pkg/testobs"
)

func shiftSlice(s []string, pos int, val string) []string {
	s = append(s, "")
	copy(s[pos+1:], s[pos:])
	s[pos] = val
	return s
}

type GoTestLine struct {
	Time    time.Time
	Action  string
	Package string
	Test    string
	Output  string
	Elapsed float64
}

var SuiteName string
var SuiteRunID string

func populateSuiteNameAndRunID(civars testobs.CISystemVars) {
	if SuiteName == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Printf("wd: unable to set suitename: %v\n", err)
		}
		homedir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("homedir: unable to set suitename: %v\n", err)
		}
		SuiteName = strings.TrimPrefix(wd, homedir)
	}
	if SuiteRunID == "" {
		SuiteRunID = civars.SeqBuildID
	}

	fmt.Println("SuiteName:", SuiteName, "RunID", SuiteRunID)
}

var rootCmd = &cobra.Command{
	Use:   "codecomet",
	Short: "The CodeComet CLI collects metrics about your tests and uploads them to the CodeComet web app.",
	Long: `To use, simply prefix your "go test" or "pytest" or other test commands with codecomet --

Don't forget the -- separator! You can pass flags to the codecomet executable,
such as the suite name (it will default to your folder name) or the suite run ID
(which will default to the run ID in your CI system).

For example:

codecomet -s MyBackendTests -- go test -json -coverprofile=cover.out ./...

	`,
	Run: func(cmd *cobra.Command, args []string) {
		// args are the arguments after the -- above
		remoteServer := viper.GetString("REMOTE_SERVER")
		if remoteServer == "" {
			remoteServer = "https://app.codecomet.io/api"
		}
		client := traceconsumerv1connect.NewTraceServiceClient(
			http.DefaultClient,
			remoteServer,
			connect.WithSendGzip(),
		)

		var coverProfileFile, testLogOutputFile string

		if args[0] == "go" {
			args, coverProfileFile = rewriteArgsForGotest(args)
		} else if args[0] == "pytest" {
			args, testLogOutputFile, coverProfileFile = rewriteArgsForPytest(args)
		} else {
			log.Fatal("You can only use go or pytest at this time.")
		}

		civars := testobs.AutodetectCI()
		populateSuiteNameAndRunID(civars)

		fmt.Println("CodeComet: Running command: ", args)

		testcmd := exec.Command(args[0], args[1:]...)
		var out strings.Builder

		stdout, err := testcmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		stderr, err := testcmd.StderrPipe()
		if err != nil {
			panic(err)
		}

		if err := testcmd.Start(); err != nil {
			panic(err)
		}
		// Write output to the console and to the string builder.
		w := io.MultiWriter(&out, os.Stdout)

		go io.Copy(w, stdout)
		go io.Copy(os.Stderr, stderr)

		var runErr error
		if runErr = testcmd.Wait(); runErr != nil {
			fmt.Printf("Command finished with error: %v\n", runErr)
		}

		success := runErr == nil
		status := "pass"
		if !success {
			status = "fail"
		}

		isr := &traceconsumerv1.IngestCollectionRequest{
			Run: &traceconsumerv1.TestCollectionRun{
				SuiteName:  SuiteName,
				SuiteRunId: SuiteRunID,
				CiSystem:   string(civars.System),
				Repository: civars.RepositoryOwner + "/" + civars.Repository,
				Branch:     civars.Branch,
				Status:     status,
				CommitHash: civars.CommitHash,
			},
		}

		switch args[0] {
		case "go":
			// go test output comes directly from stdout
			isr.Run.Output = []byte(out.String())
			isr.Run.OutputFormat = "gotest-json"

			// Don't send anything to backend if no tests were run.
			r := bytes.NewBuffer(isr.Run.Output)
			scanner := bufio.NewScanner(r)
			numLines := 0
			line := &GoTestLine{}
			for scanner.Scan() {
				bts := scanner.Bytes()
				err := json.Unmarshal(bts, line)
				if err != nil {
					fmt.Printf("error parsing output line: %v\n", line)
				}
				numLines++
				if numLines > 3 {
					break
				}
			}
			if numLines == 3 {
				// we have exactly three lines. Check last line
				if line.Action == "skip" {
					// No tests ever ran. There's nothing to report.
					fmt.Println("No tests were run for this collection.")
					os.Exit(0)
				}
			}

		case "pytest":
			var outBytes []byte
			f, err := os.Open(testLogOutputFile)
			if err != nil {
				fmt.Printf("unable to read open log output file: %v\n", err)
			} else {
				outBytes, err = io.ReadAll(f)
				if err != nil {
					fmt.Printf("Unable to read log output bytes: %v\n", err)
				}
			}
			isr.Run.Output = outBytes
			isr.Run.OutputFormat = "pytest-reportlog"
			f.Close()
		}

		var cbts []byte
		cf, err := os.Open(coverProfileFile)
		if err != nil {
			fmt.Printf("Unable to open coverage file: %v\n", err)
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
		resp, err := client.IngestTestCollectionRun(context.Background(), req)
		if err != nil {
			fmt.Printf("Call to CodeComet failed: %v\n", err)
		}
		fmt.Printf("CodeComet returned %v\n", resp)
	},
	Args: cobra.MinimumNArgs(1),
}

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&SuiteName, "suite", "s", "", "Provide a name for this test suite. Use the same test suite name for test runs you want to group together.")
	rootCmd.PersistentFlags().StringVarP(&SuiteRunID, "runid", "r", "", "Provide a run ID. Defaults to your CI system's run ID, if any.")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("CODECOMET")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rewriteArgsForGotest(args []string) ([]string, string) {
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
	return args, coverProfileFile
}

func rewriteArgsForPytest(args []string) ([]string, string, string) {
	var reportLogFile, coverProfileFile string
	var reportLogFound bool
	for idx, arg := range args {
		if strings.HasPrefix(arg, "--report-log=") {
			reportLogFound = true
			reportLogFile = strings.TrimPrefix(arg, "--report-log=")
			continue
		} else if arg == "--report-log" {
			reportLogFound = true
			if idx+1 > len(args)-1 {
				panic("--report-log argument missing file")
			}
			reportLogFile = args[idx+1]
		} else if strings.HasPrefix(arg, "--cov") && !strings.HasPrefix(arg, "--cov-") {
			panic("We currently do not support an existing --cov argument. Please remove it and we will add the supported coverage flags.")
		} else if strings.HasPrefix(arg, "--cov-report") {
			panic("We currently do not support an existing --cov-report argument. Please remove it and we will add the supported coverage flags.")
		}
	}
	if !reportLogFound {
		tf, err := os.CreateTemp("", "reportlog")
		if err != nil {
			panic(err)
		}
		reportLogFile = tf.Name()
		tf.Close()
		args = shiftSlice(args, 1, "--report-log="+reportLogFile)
	}

	cf, err := os.CreateTemp("", "cover*.json")
	if err != nil {
		panic(err)
	}
	coverProfileFile = cf.Name()
	cf.Close()
	args = shiftSlice(args, 1, "--cov")
	args = shiftSlice(args, 1, "--cov-report")
	args = shiftSlice(args, 2, "json:"+coverProfileFile)

	return args, reportLogFile, coverProfileFile
}
