package pkg

import (
	"fmt"
	"strconv"
)

func CheckLuna(num string) bool {
	var sum int
	var n int
	var err error
	lenNum := len(num)
	even := lenNum % 2
	fmt.Printf("len num: %d\n", lenNum)
	for i, s := range num {
		n, err = strconv.Atoi(string(s))
		if err != nil {
			return false
		}
		//fmt.Println(string(s))
		if i%2 == even {
			n = n * 2
			if n > 9 {
				n = n - 9
			}
		}
		sum = sum + n
		//fmt.Printf("%d+", n)
	}
	check := sum % 10
	//fmt.Printf("\nsum: %d\n", sum)
	//fmt.Printf("check: %d\n", check)
	if lenNum%2 == 0 {
		if check == 0 {
			return true
		}
	} else {
		return check == n
	}
	return false
}
