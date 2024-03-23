package pkg

import "fmt"

var (
	// These variables are here only to show current version. They are set in makefile during build process
	YproxyVersion         = "devel"
	GitRevision           = "devel"
	YproxyVersionRevision = fmt.Sprintf("%s-%s", YproxyVersion, GitRevision)
)
