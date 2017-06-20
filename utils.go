package ginm

func inArray(s string, a []string) bool {
	for _, i := range a {
		if i == s {
			return true
		}
	}
	return false
}
