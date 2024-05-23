package utils

import "math/rand"

const KPValidCharsForFilename = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const KPFilenameLength = 12

func RandomValue(length int) string {
	if length == 0 {
		length = KPFilenameLength
	}

	chars := make([]byte, length)

	for i := 0; i < length; i++ {
		chars[i] = KPValidCharsForFilename[rand.Intn(len(chars))]
	}

	return string(chars)
}
