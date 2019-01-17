package main

var (
	// GitVersion is a placeholder to inject git version, result of `git describe`
	GitVersion = "n/a"
	// GitCommit is a placeholder to inject git commit, result of `git rev-parse HEAD`
	GitCommit = "n/a"
	// GitCommitShort is a placeholder to inject git commit, result of `git rev-parse --short HEAD`
	GitCommitShort = "n/a"
)

// Version describes application version
type Version struct {
	Version     string
	Commit      string
	CommitShort string
}
