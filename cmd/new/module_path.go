package new

import (
	"fmt"
)

func defaultModulePath(appName string) string {
	return fmt.Sprintf("github.com/my-github-user/%s", appName)
}
