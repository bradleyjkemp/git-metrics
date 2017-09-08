package main

import (
	"fmt"
	"io/ioutil"
	"os"

	git "gopkg.in/src-d/go-git.v4"

	"github.com/bradleyjkemp/git-metrics/lib"
	"github.com/bradleyjkemp/git-metrics/metrics"
	"github.com/spf13/pflag"
)

var (
	repoPath          string
	metric            string
	resultFile        string
	metricCalculators = map[string]lib.GitMetric{
		"filetypes": &metrics.Filetypes{},
	}
)

func init() {
	pflag.StringVarP(&repoPath, "repo", "r", "", "Path to repository. Defaults to the repository containing the current working directory")
	pflag.StringVarP(&metric, "metric", "m", "", "Name of metric to calculate")
	pflag.StringVarP(&resultFile, "out", "o", "result.html", "File to output result to")
}

func exit(messageFormat string, values ...interface{}) {
	fmt.Fprintf(os.Stderr, messageFormat+"\n", values...)
	pflag.Usage()
	os.Exit(1)
}

func main() {
	pflag.Parse()
	if metric == "" {
		exit("please specify a metric to calculate")
	}
	metricCalculator := metricCalculators[metric]
	if metricCalculator == nil {
		exit("unknown metric: %s", metric)
	}

	if repoPath == "" {
		var err error
		repoPath, err = lib.FindRepoRoot()
		if err != nil {
			exit("could not find repo root. Are you in a git repository? %s", err)
		}
	}

	var repo *git.Repository
	var err error
	fmt.Print("Cloning repo...")
	if metricCalculator.IsReadOnly() {
		repo, err = lib.OpenRepoInMemory(repoPath)
	} else {
		tempDir, err := ioutil.TempDir("", "git-metrics")
		if err != nil {
			exit("failed to create temp directory: %s", err)
		}
		defer func() {
			err := os.RemoveAll(tempDir)
			if err != nil {
				exit("failed to remove temporary directory: %s", tempDir)
			}
		}()

		repo, err = lib.MakeTempCopyOfRepo(repoPath, tempDir)
	}
	if err != nil {
		exit("failed to copy repo: %s", err)
	}
	fmt.Println("Done")

	fmt.Print("Calculating metrics...")
	samples, err := lib.CalculateMetrics(repo, metricCalculator)
	if err != nil {
		exit("failed to calculate samples: %s", err)
	}
	fmt.Println("Done")

	f, err := os.Create(resultFile)
	if err != nil {
		exit("failed to create result file: %s", err)
	}
	defer f.Close()
	fmt.Print("Rendering graph...")
	metricCalculator.RenderGraph(samples, f)
	fmt.Println("Done")
}
