package nagerdate

import (
	"fmt"
	"time"
)

func currentYear() int {
	return time.Now().Year()
}

func currentYearStr() string {
	return fmt.Sprintf("%d", currentYear())
}
