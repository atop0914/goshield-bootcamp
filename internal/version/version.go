package version

import "fmt"

// Build-time variables set via ldflags.
var (
    Version   = "dev"
    GitCommit = "unknown"
    BuildDate = "unknown"
)

// Info contains version information.
type Info struct {
    Version   string `json:"version"`
    GitCommit string `json:"git_commit"`
    BuildDate string `json:"build_date"`
}

// Get returns the current version info.
func Get() Info {
    return Info{
        Version:   Version,
        GitCommit: GitCommit,
        BuildDate: BuildDate,
    }
}

// String returns a formatted version string.
func (i Info) String() string {
    return fmt.Sprintf("goshield %s (commit: %s, built: %s)", i.Version, i.GitCommit, i.BuildDate)
}
