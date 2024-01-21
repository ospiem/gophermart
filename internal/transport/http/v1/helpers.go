package v1

const luhnAlgoDivisor = 10

func isValidByLuhnAlgo(numbers []int) bool {
	var sum int
	isSecond := false
	for _, d := range numbers {
		if isSecond {
			d *= 2
		}
		sum += d / luhnAlgoDivisor
		sum += d % luhnAlgoDivisor
		isSecond = !isSecond
	}
	return sum%luhnAlgoDivisor == 0
}
