package helper

func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

func RemoveDuplicates[T comparable](input []T) []T {
	seen := make(map[T]bool, len(input))
	result := make([]T, 0, len(input))

	for _, v := range input {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
