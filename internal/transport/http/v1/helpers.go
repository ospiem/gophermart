package v1

func isValidByLuhnAlgo(numbers []int) bool {
	var sum int
	isSecond := false
	for _, d := range numbers {
		if isSecond {
			d = d * 2
		}
		sum += d / 10
		sum += d % 10
		isSecond = !isSecond
	}
	return sum%10 == 0
}
