package validate

import (
	"github.com/OlegMzhelskiy/gophermart/internal/models"
	"strconv"
)

func CheckLuna(num models.OrderNumber) bool {
	var sum int
	var n int
	var err error
	lenNum := len(num)
	even := lenNum % 2
	for i, s := range num {
		n, err = strconv.Atoi(string(s))
		if err != nil {
			return false
		}
		if i%2 == even {
			n = n * 2
			if n > 9 {
				n = n - 9
			}
		}
		sum = sum + n
	}
	check := sum % 10
	return check == 0
}
