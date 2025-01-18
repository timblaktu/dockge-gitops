package config

import (
    "strconv"

    "github.com/ebaldebo/dockge-gitops/internal/env"
)

type Config struct {
	RepoUrl         string
	Pat             string
	PollingRate     string
	DockgeStacksDir string
    NoLocalChanges	bool
}

const (
	repoUrlEnv         = "REPO_URL"
	patEnv             = "PAT"
	pollingRateEnv     = "POLLING_RATE"
	dockgeStacksDirEnv = "DOCKGE_STACKS_DIR"
    noLocalChangesEnv  = "NO_LOCAL_CHANGES"
)

func New() (*Config, error) {
	repoUrl, err := env.GetEnvVar(true, repoUrlEnv, "")
	if err != nil {
		return nil, err
	}

	pat, err := env.GetEnvVar(false, patEnv, "")
	if err != nil {
		return nil, err
	}

	pollingRate, err := env.GetEnvVar(false, pollingRateEnv, "5m")
	if err != nil {
		return nil, err
	}

	dockgeStacksDir, err := env.GetEnvVar(false, dockgeStacksDirEnv, "/opt/stacks")
	if err != nil {
		return nil, err
	}

	noLocalChangesString, err := env.GetEnvVar(false, noLocalChangesEnv, "true")
	if err != nil {
		return nil, err
	}
	noLocalChanges, err := strconv.ParseBool(noLocalChangesString);
    if err != nil {
        return nil, err
    }

	return &Config{
		RepoUrl:         repoUrl,
		Pat:             pat,
		PollingRate:     pollingRate,
		DockgeStacksDir: dockgeStacksDir,
        NoLocalChanges:  noLocalChanges,
	}, err
}
