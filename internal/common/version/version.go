package version

import "fmt"

const (
	unknown      = "unknown"
	unknownTag   = "v0.0.0"
	commitLength = 7
)

var (
	gitTag    = unknownTag
	gitCommit = unknown
	gitBranch = unknown
	buildDate = unknown
)

func Version() string {
	return gitTag
}

func Short() string {
	return fmt.Sprintf("%s (%s)", gitTag, gitCommit[:commitLength])
}

func Full() string {
	return fmt.Sprintf("%s (%s) [%s] %s", gitTag, gitCommit[:commitLength], gitBranch, buildDate)
}
