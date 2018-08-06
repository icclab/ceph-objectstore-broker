package broker

import ()

func sliceContains(needle string, haystack []string) bool {
	for _, element := range haystack {
		if element == needle {
			return true
		}
	}
	return false
}

func getElementIndex(s string, slice []string) int {
	for i, x := range slice {
		if s == x {
			return i
		}
	}

	return -1
}

func removeFromSlice(s string, slice []string) []string {
	i := getElementIndex(s, slice)

	length := len(slice)
	t := slice[length-1]
	slice[length-1] = slice[i]
	slice[i] = t
	return slice[:length-1]
}
