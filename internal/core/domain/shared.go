package domain

type ID string

func ValidateID(id string) bool {
	return len(id) == 24
}

type Amount int

func NewAmountFromCents(cents int) Amount {
	return Amount(cents)
}

func NewAmountFromValue(value int) Amount {
	return Amount(value * 100)
}

func (a Amount) Add(b Amount) Amount {
	return a + b
}

func (a Amount) Multiply(b int) Amount {
	return a * Amount(b)
}

func (a Amount) ToValue() int {
	return int(a) / 100
}

type Event interface {
	GetName() string
	GetEntityName() string
}
