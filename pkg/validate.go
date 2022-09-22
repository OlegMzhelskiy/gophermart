package pkg

import (
	"github.com/OlegMzhelskiy/gophermart/internal/models"
	"strconv"
)

func CheckLuna(num models.OrderNumber) bool {
	//return CalculateLuhn(num)

	var sum int
	var n int
	var err error
	lenNum := len(num)
	even := lenNum % 2
	//fmt.Printf("len num: %d\n", lenNum)
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
	//if lenNum%2 == 0 {
	//	if check == 0 {
	//		return true
	//	}
	//} else {
	//	return check == n
	//}
	//return false
	return check == 0
}

func CalculateLuhn(num models.OrderNumber) bool {
	number, err := strconv.Atoi(string(num))
	if err != nil {
		return false
	}
	checkNumber := checksum(number)
	return checkNumber == 0
	//if checkNumber == 0 {
	//	return 0
	//}
	//return 10 - checkNumber
}

func checksum(number int) int {
	var luhn int

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}
