package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"gopkg.in/src-d/go-billy.v3/memfs"
	"gopkg.in/src-d/go-billy.v3/osfs"
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
		fmt.Println("looking in", filepath.Join(cwd, ".git"))
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

func MakeTempCopyOfRepo(repoPath string, tempPath string) (*git.Repository, error) {
	// fs is a filesystem abstraction that contains the files in the repo
	fs := osfs.New(tempPath)

	// storer holds the .git database
	storer := memory.NewStorage()

	return git.Clone(storer, fs, &git.CloneOptions{
		URL: repoPath,
	})
}

func OpenRepoInMemory(path string) (*git.Repository, error) {
	// Filesystem abstraction based on memory
	fs := memfs.New()
	// Git objects storer based on memory
	storer := memory.NewStorage()

	// Clones the repository into the worktree (fs) and storer all the .git
	// content into the storer
	return git.Clone(storer, fs, &git.CloneOptions{
		URL: path,
	})
}

type HistoryObserver func(*object.Commit, *git.Worktree) error

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func WalkUpRepoHistory(repo *git.Repository, observer HistoryObserver) error {
	headRef, err := repo.Head()
	if err != nil {
		return err
	}
	head, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	for {
		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: head.Hash,
		})
		if err != nil {
			return err
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
			return err
		}
	}
}
