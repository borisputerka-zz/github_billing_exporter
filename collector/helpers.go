package collector

import "strings"

func parseArg(arg string) []string {
	return strings.Split(strings.ReplaceAll(arg, " ", ""), ",")
}
