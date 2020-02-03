package buildinfo

// Build Infos
var (
	version   string
	commit    string
	buildDate string
)

func Version() string {
	return version
}

func Commit() string {
	return commit
}

func BuildDate() string {
	return buildDate
}
