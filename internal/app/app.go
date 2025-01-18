package app

import (
	"fmt"
	"io/ioutil"
    "os"
	"time"

	"github.com/ebaldebo/dockge-gitops/internal/app/config"
	"github.com/ebaldebo/dockge-gitops/internal/git"
	"github.com/ebaldebo/dockge-gitops/internal/polling"
)

const (
	repoDir = "/tmp/repo"
)

func Run(cfg *config.Config) {
    files, err := ioutil.ReadDir(cfg.DockgeStacksDir)
	handleError(err)
    for _, file := range files {
        fmt.Println(file.Name(), file.IsDir())
    }

    pollingRateDuration, err := polling.ParsePollingRate(cfg.PollingRate)
	handleError(err)

	ticker := time.NewTicker(pollingRateDuration)
	defer ticker.Stop()

    gitErr := git.HandleRepo(cfg.RepoUrl, cfg.Pat, repoDir, cfg.DockgeStacksDir, cfg.NoLocalChanges)
    handleError(gitErr)
    for range ticker.C {
        gitErr := git.HandleRepo(cfg.RepoUrl, cfg.Pat, repoDir, cfg.DockgeStacksDir, cfg.NoLocalChanges)
        handleError(gitErr)
    }
}

func handleError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		err := git.ClearRepoFolder(repoDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		os.Exit(1)
	}
}
