package lib

import (
	"io"

	"github.com/pkg/errors"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type MetricCalculator interface {
	CalculateMetrics(repo *git.Worktree) (map[string]int, error)
}
type GraphRenderer interface {
	RenderGraph(samples []Sample, output io.Writer) error
}

type GitMetric interface {
	MetricCalculator
	GraphRenderer
	IsReadOnly() bool
}

type Sample struct {
	Commit       *object.Commit
	Measurements map[string]int
}

func CalculateMetrics(repo *git.Repository, m MetricCalculator) ([]Sample, error) {
	var samples []Sample
	err := WalkUpRepoHistory(repo, func(c *object.Commit, w *git.Worktree) error {
		measurements, err := m.CalculateMetrics(w)
		if err != nil {
			return errors.Wrapf(err, "failed to calculate metric on commit %s", c.Hash)
		}
		samples = append(samples, Sample{
			Commit:       c,
			Measurements: measurements,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// reverse samples so oldest is first
	for i := len(samples)/2 - 1; i >= 0; i-- {
		opp := len(samples) - 1 - i
		samples[i], samples[opp] = samples[opp], samples[i]
	}

	return samples, nil
}
