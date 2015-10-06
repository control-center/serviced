package utils

const digits = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func Base62(n uint) string {
	b62 := ""
	if n == 0 {
		return "0"
	}
	for n != 0 {
		r := n % 62
		n = n / 62
		b62 = string(digits[r]) + b62
	}
	return b62
}
