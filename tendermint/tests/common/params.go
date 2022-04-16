package common

type SystemParams struct {
	N int
	F int
}

func NewSystemParams(n int) *SystemParams {
	return &SystemParams{
		N: n,
		F: int((n - 1) / 3),
	}
}
