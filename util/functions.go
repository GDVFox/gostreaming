package util

// FindStringIndex возвращает индекс первого вхождения строки s в массиве a.
func FindStringIndex(a []string, s string) int {
	for i, str := range a {
		if str == s {
			return i
		}
	}
	return -1
}
