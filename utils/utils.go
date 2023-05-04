package utils

import (
	"encoding/json"
	"fmt"
)

func StrSliceToSet(strs []string) map[string]bool {
	strSet := map[string]bool{}
	for _, str := range strs {
		strSet[str] = true
	}
	return strSet
}

func DeepCopy(dst, src interface{}) {
	bytes, err := json.Marshal(src)
	if err != nil {
		panic(fmt.Sprintf("json Marshal with failed, error %s", err.Error()))
	}
	err = json.Unmarshal(bytes, dst)
	if err != nil {
		panic(fmt.Sprintf("json UnMarshal with failed, error %s", err.Error()))
	}
}
