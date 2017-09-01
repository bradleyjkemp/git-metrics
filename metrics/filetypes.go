package metrics

import (
	"fmt"
	"io"
	"math/rand"
	"path/filepath"
	"sort"

	"github.com/bradleyjkemp/git-metrics/lib"
	chartjs "github.com/brentp/go-chartjs"
	"github.com/brentp/go-chartjs/types"
	"gopkg.in/src-d/go-billy.v3"
	"gopkg.in/src-d/go-git.v4"
)

type Filetypes struct{}

func (*Filetypes) IsReadOnly() bool {
	return true
}

func (*Filetypes) CalculateMetrics(repo *git.Worktree) (map[string]int, error) {
	fs := repo.Filesystem
	return countFiletypesInDirectory("/", fs)
}

func countFiletypesInDirectory(path string, fs billy.Filesystem) (map[string]int, error) {
	result := map[string]int{}
	ls, err := fs.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, fileInfo := range ls {
		if fileInfo.IsDir() {
			subDirResult, err := countFiletypesInDirectory(filepath.Join(path, fileInfo.Name()), fs)
			if err != nil {
				return nil, err
			}
			for filetype, count := range subDirResult {
				result[filetype] += count
			}
		} else {
			ext := filepath.Ext(fileInfo.Name())
			if ext != "" {
				result[ext] += 1
			}
		}
	}

	return result, nil
}

type xyPair struct {
	x int
	y float64
}

type xyPairs []xyPair

func (xys xyPairs) append(p xyPair) xyPairs {
	return append(xys, p)
}

func (xys xyPairs) Xs() []float64 {
	xs := []float64{}
	for _, pair := range xys {
		xs = append(xs, float64(pair.x))
	}
	return xs
}

func (xys xyPairs) Ys() []float64 {
	ys := []float64{}
	for _, pair := range xys {
		ys = append(ys, pair.y)
	}
	return ys
}

func (xys xyPairs) Rs() []float64 {
	return nil
}

func randColor() *types.RGBA {
	return &types.RGBA{
		uint8(rand.Intn(256)),
		uint8(rand.Intn(256)),
		uint8(rand.Intn(256)),
		255,
	}
}

func (*Filetypes) RenderGraph(samples []lib.Sample, output io.Writer) error {
	allFiletypesMap := map[string]bool{}
	for _, sample := range samples {
		for filetype, _ := range sample.Measurements {
			allFiletypesMap[filetype] = true
		}
	}
	allFiletypes := []string{}
	for filetype, _ := range allFiletypesMap {
		allFiletypes = append(allFiletypes, filetype)
	}
	sort.Slice(allFiletypes, func(i, j int) bool {
		return samples[len(samples)-1].Measurements[allFiletypes[i]] < samples[len(samples)-1].Measurements[allFiletypes[j]]
	})
	fmt.Println(allFiletypes)

	datasets := map[string]xyPairs{}
	// make sure there is a dataset for each filetype ever seen
	for _, filetype := range allFiletypes {
		datasets[filetype] = xyPairs{}
	}

	var commitNum int
	for _, sample := range samples {
		// fmt.Println(sample)
		// fmt.Println(sample.Commit.Message[:lib.Min(20, len(sample.Commit.Message)-1)], sample.Commit.Hash)
		var totalFiles float64
		for _, count := range sample.Measurements {
			totalFiles += float64(count)
		}

		currentPercent := 0.0
		for _, filetype := range allFiletypes {
			dataset := datasets[filetype]
			currentPercent += float64(sample.Measurements[filetype]) / totalFiles * 100.0
			datasets[filetype] = dataset.append(xyPair{
				commitNum,
				currentPercent,
			})
		}
		commitNum++
	}

	chart := &chartjs.Chart{
		Label: "By filetype",
		Options: chartjs.Options{
			Tooltip: &chartjs.Tooltip{
				Enabled: types.True,
			},
		},
	}
	for _, filetype := range allFiletypes {
		dataset := datasets[filetype]
		color := randColor()
		chart.AddDataset(chartjs.Dataset{
			Data:            dataset,
			Type:            chartjs.Line,
			Label:           filetype,
			BorderColor:     color,
			BackgroundColor: color,
			//  Fill: types.False,
			// 			SteppedLine:     types.True,
		})
	}

	chart.AddXAxis(chartjs.Axis{
		Type:     chartjs.Linear,
		Position: chartjs.Bottom,
	})
	chart.AddYAxis(chartjs.Axis{
		Type:     chartjs.Linear,
		Position: chartjs.Left,
	})

	return chartjs.SaveCharts(output, nil, *chart)
}
