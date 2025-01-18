package git

import (
	"errors"
    "fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ebaldebo/dockge-gitops/internal/env"
	"github.com/ebaldebo/dockge-gitops/internal/files"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func HandleRepo(repoUrl, pat, dirPath, stackPath string, noLocalChanges bool) error {
    if noLocalChanges {
        return updateRepo(repoUrl, pat, stackPath) 
    } else {
        return CloneOrPullRepo(repoUrl, pat, dirPath, stackPath)
    }
}

func updateRepo(repoUrl, pat, stackPath string) error {
	var r *git.Repository = nil;
	var err error;
    urlWithAuth, err := buildUrl(repoUrl, pat)
    if err != nil {
		return fmt.Errorf(buildingUrlErr, err)
	}
    fmt.Printf("Attempting clone of %s into %s..\n", repoUrl, stackPath)
    r, err = git.PlainClone(stackPath, false, &git.CloneOptions{
		URL: urlWithAuth,
    });
    if err != nil {
        if errors.Is(err, git.ErrRepositoryAlreadyExists) {
            fmt.Printf("Repo already exists. Fetching..\n")
            r, err = git.PlainOpen(stackPath)
            if err != nil || r == nil {
                return fmt.Errorf(openRepoErr, r, err)
            }
            if err := r.Fetch(&git.FetchOptions{
            }); err != nil && err != git.NoErrAlreadyUpToDate {
                return fmt.Errorf(fetchingRepoErr, err)
            }
            fmt.Printf("git reset --hard HEAD\n")
            head, err := r.Head();
            if err != nil {
                return fmt.Errorf(gettingHeadRepoErr, err)
            }
            w, err := r.Worktree();
            if err != nil {
                return fmt.Errorf(workTreeErr, err)
            }
            if err := w.Reset(&git.ResetOptions{
                Commit: head.Hash(),
                Mode: git.HardReset,
            }); err != nil {
                return fmt.Errorf(resettingRepoErr, err)
            }
            fmt.Printf("git merge '@{u}'\n")
            err = r.Merge(*head, git.MergeOptions{
                Strategy: git.FastForwardMerge,
            }); 
            if err != nil {
                return fmt.Errorf(mergingRepoErr, err)
            }
            return nil

        } else {
            return fmt.Errorf(cloningRepoErr, err)
        }
    } 
    fmt.Println(repoClonedMsg)
    return nil
}

func CloneOrPullRepo(repoUrl, pat, dirPath, stackPath string) error {
    if err := os.MkdirAll(dirPath, 0755); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
    }
	url, err := buildUrl(repoUrl, pat)
	if err != nil {
		return fmt.Errorf(buildingUrlErr, err)
	}

    gitPath := filepath.Join(dirPath, ".git")
    if _, err := os.Stat(gitPath); os.IsNotExist(err) {
        return cloneRepo(url, dirPath, stackPath)
    }

    if err != nil {
        return fmt.Errorf(checkingIfRepoExistsErr, err)
    }

    shouldPull, err := remoteHasUpdate(dirPath)
    if err != nil {
        return fmt.Errorf(checkingIfRepoHasUpdateErr, err)
    }

    if !shouldPull {
        fmt.Println(repoUpToDateMsg)
        return nil
    }

    fmt.Printf(repoNotUpToDateMsg, repoUrl, dirPath)
	return pullRepo(dirPath, stackPath)
}

func ClearRepoFolder(dirPath string) error {
	files, err := filepath.Glob(filepath.Join(dirPath, "*"))
	if err != nil {
		return fmt.Errorf(gettingFilesFromRepoDirErr, err)
	}

	for _, file := range files {
		if err := os.RemoveAll(file); err != nil {
			return fmt.Errorf(removingFileErr, file, err)
		}
	}

	return nil
}

func cloneRepo(repoUrl, targetDirPath, sourceDirPath string) error {
	files, err := os.ReadDir(targetDirPath)
	if err != nil {
		return fmt.Errorf(readingDirErr, err)
	}

	if len(files) > 0 {
		return fmt.Errorf(targetDirNotEmptyErr)
	}

	// We have asserted that there is no repo in targetDirPath.
    fmt.Printf(repoNotExistsCloningMsg, targetDirPath)
    
    if _, err := git.PlainClone(targetDirPath, false, &git.CloneOptions{
		URL: repoUrl,
    }); err != nil {
		return fmt.Errorf(cloningRepoErr, err)
	}
	fmt.Println(repoClonedMsg)

	if err := copyFilesToDir(targetDirPath, sourceDirPath); err != nil {
		return fmt.Errorf(copyingSubfoldersErr, err)
	}

	return nil
}

func pullRepo(dirPath, stackPath string) error {
	repo, err := git.PlainOpen(dirPath)
	if err != nil {
		return fmt.Errorf(openingRepoErr, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf(gettingWorkTreeErr, err)
	}

	err = worktree.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf(pullingRepoErr, err)
	}

	if err == git.NoErrAlreadyUpToDate {
		fmt.Println(repoUpToDateMsg)
	} else {
		fmt.Println(repoPulledMsg)
	}

	if err := copyFilesToDir(dirPath, stackPath); err != nil {
		return fmt.Errorf(copyingSubfoldersErr, err)
	}

	return nil
}

func buildUrl(repoUrl, pat string) (string, error) {
	parsedUrl, err := url.Parse(repoUrl)
	if err != nil {
		return "", fmt.Errorf(urlParseErr, err)
	}

	if pat == "" {
		return repoUrl, nil
	}

	return fmt.Sprintf("https://%s@%s%s", pat, parsedUrl.Host, parsedUrl.Path), nil
}

func remoteHasUpdate(dirpath string) (bool, error) {
	r, err := git.PlainOpen(dirpath)
	if err != nil {
		return false, fmt.Errorf(openingRepoErr, err)
	}

	ref, err := r.Head()
	if err != nil {
		return false, fmt.Errorf(getLocalCommitErr, err)
	}

	currentBranch := ref.Name().Short()

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return false, fmt.Errorf(getLocalCommitErr, err)
	}

	err = r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return false, fmt.Errorf(gitFetchErr, err)
	}

	remoteBranchRef := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", currentBranch))

	remoteRef, err := r.Reference(remoteBranchRef, true)
	if err != nil {
		return false, fmt.Errorf(getRemoteErr, err)
	}

	remoteCommit, err := r.CommitObject(remoteRef.Hash())
	if err != nil {
		return false, fmt.Errorf(getLocalCommitErr, err)
	}

	return commit.Hash != remoteCommit.Hash, nil
}

func copyFilesToDir(dirPath, newDirPath string) error {
	fmt.Printf(copyingFilesMsg, dirPath, newDirPath)
	clearDestination(newDirPath)

	items, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf(gettingSubDirsErr, err)
	}

	for _, item := range items {
		if !item.Type().IsDir() || item.Name() == ".git" {
			continue
		}
		sourcePath := filepath.Join(dirPath, item.Name())
		destPath := filepath.Join(newDirPath, item.Name())
		dgoFilePath := filepath.Join(destPath, dgoFileName)

		destInfo, err := os.Stat(destPath)
		if err == nil && destInfo.IsDir() {
			if _, err := os.Stat(dgoFilePath); os.IsNotExist(err) {
				return fmt.Errorf(conflictingStackErr, item.Name(), err)
			}
		}

		if err := files.CopyDirectory(sourcePath, destPath); err != nil {
			return fmt.Errorf(copyingSubfoldersErr, err)
		}

		if err := os.WriteFile(dgoFilePath, []byte(dgoContent), 0644); err != nil {
			return fmt.Errorf(writingDgoFileErr, err)
		}

		if env.EnvFileExists(envFilePath) {
			destEnvFilePath := filepath.Join(destPath, filepath.Base(envFilePath))
			if err := files.CopyFile(envFilePath, destEnvFilePath); err != nil {
				return fmt.Errorf(copyingEnvFileErr, err)
			}
		}
	}

	fmt.Println(filesCopiedMsg)
	return nil
}

func clearDestination(newDirPath string) error {
	dirs, err := filepath.Glob(newDirPath + "/*")
	if err != nil {
		return fmt.Errorf(gettingFilesFromDestinationErr, err)
	}

	fmt.Printf(clearDestinationMsg, newDirPath)
	for _, dir := range dirs {
		dgoFilePath := filepath.Join(dir, dgoFileName)
		if _, err := os.Stat(dgoFilePath); err == nil {
            fmt.Printf(clearDirMsg, dir)
			err := os.RemoveAll(dir)
			if err != nil {
				return fmt.Errorf(removingFilesFromDestinationErr, err)
			}
		}
	}

	return nil
}
