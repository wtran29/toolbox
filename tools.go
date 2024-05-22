package toolbox

import (
	"crypto/rand"
	"math/big"
)

var randomRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+_1234567890")

// Tools is the type used to instantiate module. Any variable of this type
// will have access to the methods with receiver *Tools.
type Tools struct{}

// RandomString generates a random string of length using characters from randomRunes
func (t *Tools) RandomString(n int) string {
	runes := []rune(randomRunes)
	result := make([]rune, n)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(runes))))
		result[i] = runes[num.Int64()]
	}
	return string(result)
}
