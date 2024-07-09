package main

import (
	"fmt"
	"strconv"
	"strings"
)

func ConvertStringToUint16Slice(input string) ([]uint16, error) {
	numberStrings := strings.Split(input, ",")
	fmt.Println(numberStrings)

	var result []uint16

	for _, numberString := range numberStrings {
		numberString = strings.TrimSpace(numberString)

		number, err := strconv.ParseUint(numberString, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("error parsing '%s': %v", numberString, err)
		}

		result = append(result, uint16(number))
	}

	return result, nil
}

func main() {
	xx, _ := ConvertStringToUint16Slice("")
	fmt.Println(xx)
}
