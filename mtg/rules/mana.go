package mtg

type ManaPool struct {
	White     int
	Blue      int
	Black     int
	Red       int
	Green     int
	Colorless int
}

func (m *ManaPool) AddMana(color string, amount int) {
	switch color {
	case "White":
		m.White += amount
	case "Blue":
		m.Blue += amount
	case "Black":
		m.Black += amount
	case "Red":
		m.Red += amount
	case "Green":
		m.Green += amount
	case "Colorless":
		m.Colorless += amount
	}
}

func (m *ManaPool) RemoveMana(color string, amount int) {
	switch color {
	case "White":
		m.White -= amount
	case "Blue":
		m.Blue -= amount
	case "Black":
		m.Black -= amount
	case "Red":
		m.Red -= amount
	case "Green":
		m.Green -= amount
	case "Colorless":
		m.Colorless -= amount
	}
}

func (m *ManaPool) CanPay(cost string) bool {
	// TODO: Implement mana cost parsing and payment logic
	return false
}
