package echos

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

func GenerateMeetRoomID(segc, chc int) (string, error) {
	const chars = "abcdefghijkmnpqrstuvwxyz"

	segments := make([]string, segc)

	for i := 0; i < segc; i++ {
		segment := make([]byte, chc)

		for j := 0; j < segc; j++ {
			randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			if err != nil {
				return "", fmt.Errorf("failed to generate random number: %w", err)
			}

			segment[j] = chars[randomIndex.Int64()]
		}

		segments[i] = string(segment)
	}

	return strings.Join(segments, "-"), nil
}
