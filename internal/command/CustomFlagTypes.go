package command

import (
	"fmt"
	"strings"
)

type StringArray []string

func (arr *StringArray) String() string {
	return fmt.Sprintf("%v", *arr)
}

func (arr *StringArray) Set(s string) error {
	*arr = strings.Split(s, " ")
	return nil
}
