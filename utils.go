package main

// Filter slice过滤
func Filter[S ~[]E, E any](slice S, condition func(E) bool) S {
	var filtered []E
	for _, item := range slice {
		if condition(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
