package utils

import "fmt"

func GetBytes(key interface{}) []byte {
	if key == nil {
		return nil
	}
	return []byte(fmt.Sprintf("%v", key))
}
