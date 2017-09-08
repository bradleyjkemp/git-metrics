package lib

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"gopkg.in/src-d/go-billy.v3/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func FindRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		_, err := os.Stat(filepath.Join(cwd, ".git"))
		if !os.IsNotExist(err) {
			return cwd, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			return "", errors.New("failed to find .git in any parent directory")
		}
		cwd = parent
	}
}

// Clone the repo into a temp folder
func MakeTempCopyOfRepo(repoPath string, tempPath string) (*git.Repository, error) {
	return git.PlainClone(tempPath, false, &git.CloneOptions{
		URL: repoPath,
	})
}

// Clone the repo in memory
func OpenRepoInMemory(path string) (*git.Repository, error) {
	fs := memfs.New()
	storer := memory.NewStorage()

	// Clones the repository into the worktree (fs) and storer all the .git
	// content into the storer
	return git.Clone(storer, fs, &git.CloneOptions{
		URL: path,
	})
}

type HistoryObserver func(*object.Commit, *git.Worktree) error

type Options struct {
	CommitLimit int
	TimeLimit   time.Time
}

func WalkUpRepoHistory(repo *git.Repository, observer HistoryObserver) error {
	return Options{}.WalkUpRepoHistory(repo, observer)
}

func (o Options) WalkUpRepoHistory(repo *git.Repository, observer HistoryObserver) error {
	headRef, err := repo.Head()
	if err != nil {
		return err
	}
	head, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return errors.Wrapf(err, "failed to get commit object for %s", headRef.Hash())
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "failed to get worktree for repo")
	}

	commitsObserver := 0
	for {
		if head.Committer.When.Before(o.TimeLimit) {
			return nil
		}
		if o.CommitLimit != 0 && commitsObserver > o.CommitLimit {
			return nil
		}

		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: head.Hash,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to checkout commit %s", head.Hash)
		}

		err = observer(head, worktree)
		if err != nil {
			return errors.Wrap(err, "failed running observer")
		}

		if len(head.ParentHashes) == 0 {
			// reached root of tree
			return nil
		}

		// first parent hash is from this branch, any others are from merged branches
		head, err = repo.CommitObject(head.ParentHashes[0])
		if err != nil {
			return errors.Wrapf(err, "failed to get commit object for %s", head.ParentHashes[0])
		}
	}
}
