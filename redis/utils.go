package redis

import "fmt"

func SequenceNumberKey(userID string) string {
	return fmt.Sprintf("%s:%s", SequenceNumberPrefix, userID)
}
