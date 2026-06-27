package duel

import (
	"image"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func TestManaCostColorRank_WUBRGThenMulticolorThenColorless(t *testing.T) {
	ranks := []struct {
		cost string
		want int
	}{
		{"{W}", int(core.White)},
		{"{U}", int(core.Blue)},
		{"{B}", int(core.Black)},
		{"{R}", int(core.Red)},
		{"{G}", int(core.Green)},
		{"{W}{B}", int(core.Green) + 1}, // multicolor
		{"{2}", int(core.Green) + 2},    // colorless
	}
	for _, r := range ranks {
		if got := manaCostColorRank(core.ParseManaCost(r.cost)); got != r.want {
			t.Fatalf("colorRank(%q) = %d, want %d", r.cost, got, r.want)
		}
	}
	// White must sort before colorless, multicolor between colored and colorless.
	if manaCostColorRank(core.ParseManaCost("{W}")) >= manaCostColorRank(core.ParseManaCost("{W}{B}")) {
		t.Fatal("monocolor should rank before multicolor")
	}
	if manaCostColorRank(core.ParseManaCost("{W}{B}")) >= manaCostColorRank(core.ParseManaCost("{2}")) {
		t.Fatal("multicolor should rank before colorless")
	}
}

func TestLessCardSortKey(t *testing.T) {
	land := cardSortKey{isLand: true, name: "Forest"}
	spell := cardSortKey{isLand: false, name: "Bolt", colorRank: 4, cmc: 1}
	if !lessCardSortKey(land, spell) {
		t.Fatal("lands should sort before spells")
	}
	if lessCardSortKey(spell, land) {
		t.Fatal("spells should not sort before lands")
	}

	red1 := cardSortKey{colorRank: 4, cmc: 1, name: "Shock"}
	red3 := cardSortKey{colorRank: 4, cmc: 3, name: "Char"}
	if !lessCardSortKey(red1, red3) {
		t.Fatal("lower mana value should sort first within a color")
	}

	a := cardSortKey{colorRank: 4, cmc: 1, name: "Bolt"}
	b := cardSortKey{colorRank: 4, cmc: 1, name: "Shock"}
	if !lessCardSortKey(a, b) {
		t.Fatal("same color and cmc should break ties by name")
	}
}

func names(cards []interactive.CardState) []string {
	out := make([]string, len(cards))
	for i, c := range cards {
		out[i] = c.Name
	}
	return out
}

func TestHandDisplayOrder_LandsFirstThenColorThenCMC(t *testing.T) {
	hand := []interactive.CardState{
		{Name: "Lightning Bolt", ManaCost: "{R}"},
		{Name: "Plains", IsLand: true},
		{Name: "Counterspell", ManaCost: "{U}{U}"},
		{Name: "Bayou", IsLand: true},
		{Name: "Shock", ManaCost: "{R}"},
		{Name: "Healing Salve", ManaCost: "{W}"},
		{Name: "Forest", IsLand: true},
		{Name: "Fireball", ManaCost: "{X}{R}"},
		{Name: "Vindicate", ManaCost: "{1}{W}{B}"},
		{Name: "Sol Ring", ManaCost: "{1}"},
	}

	want := []string{
		// Lands grouped first, sorted by name.
		"Bayou", "Forest", "Plains",
		// Then spells by color (WUBRG), then mana value, then name.
		"Healing Salve",                       // White
		"Counterspell",                        // Blue
		"Fireball", "Lightning Bolt", "Shock", // Red, all mana value 1, by name
		"Vindicate", // Multicolor
		"Sol Ring",  // Colorless
	}

	got := names(handDisplayOrder(hand))
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hand order mismatch at %d:\n got  %v\n want %v", i, got, want)
		}
	}
}

func TestHandDisplayOrder_DoesNotMutateInput(t *testing.T) {
	hand := []interactive.CardState{
		{Name: "Lightning Bolt", ManaCost: "{R}"},
		{Name: "Forest", IsLand: true},
	}
	handDisplayOrder(hand)
	if hand[0].Name != "Lightning Bolt" || hand[1].Name != "Forest" {
		t.Fatalf("handDisplayOrder mutated its input: %v", names(hand))
	}
}

func TestFieldCardPos_OpponentZonesMirrorPlayer(t *testing.T) {
	s := &DuelScreen{
		self:          &duelPlayer{},
		opponent:      &duelPlayer{},
		cardPositions: make(map[uuid.UUID]image.Point),
	}
	perm := interactive.PermanentState{ID: uuid.New()}

	rowY := func(dp *duelPlayer, row permRow) int {
		return s.getFieldCardPos(perm, dp, 0, 1, row).Y
	}

	// Player: creatures nearest center (top of the bottom half), lands at the
	// back (bottom of the screen).
	if rowY(s.self, permRowCreature) >= rowY(s.self, permRowOther) ||
		rowY(s.self, permRowOther) >= rowY(s.self, permRowLand) {
		t.Fatalf("player rows out of order: creature=%d other=%d land=%d",
			rowY(s.self, permRowCreature), rowY(s.self, permRowOther), rowY(s.self, permRowLand))
	}

	// Opponent: mirror image — lands at the top (back), creatures at the bottom
	// (front, near center).
	if rowY(s.opponent, permRowLand) >= rowY(s.opponent, permRowOther) ||
		rowY(s.opponent, permRowOther) >= rowY(s.opponent, permRowCreature) {
		t.Fatalf("opponent rows not mirrored: land=%d other=%d creature=%d",
			rowY(s.opponent, permRowLand), rowY(s.opponent, permRowOther), rowY(s.opponent, permRowCreature))
	}
}

func TestFieldPerms_CreaturesSortedByPowerThenToughnessDesc(t *testing.T) {
	s := &DuelScreen{}
	ps := interactive.PlayerState{
		Battlefield: []interactive.PermanentState{
			{ID: uuid.New(), Name: "Grizzly Bears", IsCreature: true, Power: 2, Toughness: 2},
			{ID: uuid.New(), Name: "Shivan Dragon", IsCreature: true, Power: 5, Toughness: 5},
			{ID: uuid.New(), Name: "Wall of Stone", IsCreature: true, Power: 0, Toughness: 8},
			{ID: uuid.New(), Name: "Hill Giant", IsCreature: true, Power: 3, Toughness: 3},
			{ID: uuid.New(), Name: "Air Elemental", IsCreature: true, Power: 4, Toughness: 4},
			{ID: uuid.New(), Name: "Sea Serpent", IsCreature: true, Power: 5, Toughness: 5},
		},
	}

	perms := s.fieldPerms(ps, permRowCreature)
	var got []string
	for _, p := range perms {
		got = append(got, p.Name)
	}
	want := []string{
		"Sea Serpent",   // 5/5, tie broken by name
		"Shivan Dragon", // 5/5
		"Air Elemental", // 4/4
		"Hill Giant",    // 3/3
		"Grizzly Bears", // 2/2
		"Wall of Stone", // 0/8
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("creature order mismatch at %d:\n got  %v\n want %v", i, got, want)
		}
	}
}

func TestFieldPerms_LandsSortedByNameThenTapped(t *testing.T) {
	s := &DuelScreen{}
	ps := interactive.PlayerState{
		Battlefield: []interactive.PermanentState{
			{ID: uuid.New(), Name: "Mountain", IsLand: true, Tapped: true},
			{ID: uuid.New(), Name: "Forest", IsLand: true, Tapped: true},
			{ID: uuid.New(), Name: "Mountain", IsLand: true, Tapped: false},
			{ID: uuid.New(), Name: "Forest", IsLand: true, Tapped: false},
		},
	}

	perms := s.fieldPerms(ps, permRowLand)
	type kv struct {
		name   string
		tapped bool
	}
	var got []kv
	for _, p := range perms {
		got = append(got, kv{p.Name, p.Tapped})
	}
	want := []kv{
		{"Forest", false}, {"Forest", true},
		{"Mountain", false}, {"Mountain", true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("land order mismatch at %d:\n got  %v\n want %v", i, got, want)
		}
	}
}
