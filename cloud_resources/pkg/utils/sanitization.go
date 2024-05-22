package sanitization

type ShortNumber interface {
	uint8 | int8
}

// Clamp given value between a min and a max
func Clamp[K ShortNumber](val, min, max K) K {
	if val < min {
		val = min
	} else if val > max {
		val = max
	}
	return val
}
