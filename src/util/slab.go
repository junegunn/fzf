package util

type Slab struct {
	I16 []int16
	I32 []int32
}

func MakeSlab(size16 int, size32 int) *Slab {
	return &Slab{
		I16: make([]int16, size16),
		I32: make([]int32, size32)}
}
