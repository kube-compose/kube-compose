package config

import "regexp"

func buildRegexpMatchMap(r *regexp.Regexp, matches []string) map[string]string {
	subexpNames := r.SubexpNames()
	n := len(subexpNames)
	matchMap := map[string]string{}
	for i := 1; i < n; i++ {
		if len(subexpNames[i]) > 0 {
			matchMap[subexpNames[i]] = matches[i]
		}
	}
	return matchMap
}