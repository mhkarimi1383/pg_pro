package utils

import "fmt"

func GetBytes(key any) []byte {
	if key == nil {
		return nil
	}
	return []byte(fmt.Sprintf("%v", key))
}
