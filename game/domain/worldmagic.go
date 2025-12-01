package domain

type WorldMagic struct {
	Name        string
	Cost        int
	Description string
}

var AllWorldMagics = []*WorldMagic{
	{
		Name:        "Sword of Resistance",
		Cost:        400,
		Description: "Grants magical resistance against enemy attacks",
	},
	{
		Name:        "Quickening",
		Cost:        300,
		Description: "Increases movement speed and reaction time",
	},
	{
		Name:        "Leap of Fate",
		Cost:        300,
		Description: "Allows teleportation to distant locations",
	},
	{
		Name:        "Ring of the Guardian",
		Cost:        500,
		Description: "Provides powerful protective barriers",
	},
	{
		Name:        "Haggler's Coin",
		Cost:        250,
		Description: "Reduces costs of all purchases",
	},
	{
		Name:        "Tome of Enlightenment",
		Cost:        300,
		Description: "Enhances learning and spell effectiveness",
	},
	{
		Name:        "Sleight of Hand",
		Cost:        300,
		Description: "Improves dexterity and stealth abilities",
	},
	{
		Name:        "Staff of Thunder",
		Cost:        100,
		Description: "Channels elemental lightning magic",
	},
	{
		Name:        "Conjurer's Will",
		Cost:        300,
		Description: "Strengthens magical focus and willpower",
	},
	{
		Name:        "Dwarven Pick",
		Cost:        125,
		Description: "Enhances mining and resource gathering",
	},
	{
		Name:        "Amulet of Swampwalk",
		Cost:        125,
		Description: "Allows safe passage through marshlands",
	},
	{
		Name:        "Fruit of Sustenance",
		Cost:        50,
		Description: "Provides eternal nourishment and vitality",
	},
}
