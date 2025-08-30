#!/usr/bin/env python3

import os
import sys

# Helper to add configs to .toml files. The images for walking and faces aren't
# controlled by config files in the original version, so we have to do the mapping
# manually.

# Character names and full names from the Go enemy_types.go file
enemies = [
    ("W_Amg", "High Priest"),
    ("W_Fwz", "Priestess"),
    ("W_Kht", "Crusader"),
    ("W_Lrd", "Paladin"),
    ("W_Mwz", "Cleric"),
    ("W_Wg", "Arch Angel"),
    ("Bk_Amg", "Necromancer"),
    ("Bk_Djn", "Aga Galneer"),
    ("Bk_Fwz", "Witch"),
    ("Bk_Kht", "Undead Knight"),
    ("Bk_Lrd", "Vampire Lord"),
    ("Bk_Mwz", "Warlock"),
    ("Bk_Wg", "Nether Fiend"),
    ("Bu_Amg", "Thought Invoker"),
    ("Bu_Djn", "Saltrem Tor"),
    ("Bu_Fwz", "Seer"),
    ("Bu_Lrd", "Merfolk Shaman"),
    ("Bu_Mwz", "Conjurer"),
    ("Bu_Wrm", "Sea Drake"),
    ("Bu_Sft", "Shapeshifter"),
    ("R_Amg", "War Mage"),
    ("R_Djn", "Queltosh"),
    ("R_Fwz", "Sorceress"),
    ("R_Lrd", "Goblin Warlord"),
    ("R_Mwz", "Sorcerer"),
    ("R_Wrm", "Crag Hydra"),
    ("Troll", "Troll Shaman"),
    ("G_Amg", "Summoner"),
    ("G_Djn", "Alt_A_Kesh"),
    ("G_Fwz", "Enchantress"),
    ("G_Kht", "Beast Master"),
    ("G_Lrd", "Elvish Magi"),
    ("G_Mwz", "Druid"),
    ("G_Wrm", "Forest Dragon"),
    ("Dg_Bru", "Mandurang"),
    ("Dg_Gwr", "Prismat"),
    ("Dg_Rbg", "Dracur"),
    ("Dg_Uwb", "Whim"),
    ("Dg_Wug", "Kiska_Ra"),
    ("M_Ape", "Ape Lord"),
    ("M_Cen", "Centaur Warchief"),
    ("M_Cen2", "Centaur Shaman"),
    ("M_Fng", "Fungus Master"),
    ("M_Fwz", "Elementalist"),
    ("M_Kht", "Lord of Fate"),
    ("M_Lrd", "Mind Stealer"),
    ("M_Trl", "Sedge Beast"),
    ("M_Tsk", "Guardian of the Tusk"),
    ("M_Wg", "Winged Stallion"),
]

rogues_dir = "assets/configs/rogues"


# Add face images to rogue .toml files
def add_faces_to_rogues_toml():
    if not os.path.exists(rogues_dir):
        print(f"Error: Directory {rogues_dir} does not exist")
        sys.exit(1)

    face_imgs = os.listdir("assets/art/sprites/rogues")

    for filename in os.listdir(rogues_dir):
        if not filename.endswith(".toml"):
            continue

        filepath = os.path.join(rogues_dir, filename)
        base_filename = os.path.splitext(filename)[0]
        face_file = ""
        for f in face_imgs:
            fclean = f.lower().replace(".png", "").replace("mps_", "").replace("-", "_")
            if fclean == base_filename:
                face_file = f
        if face_file == "":
            raise ValueError(f"Unable to find face for {base_filename}")

        # Read existing content
        with open(filepath, "r") as f:
            content = f.read()

        # Check if face line already exist
        if "face" in content:
            print(f"Skipping {filename} - face line already exist")
            continue

        # Append face line
        face_line = f'face = "{face_file}"\n'
        with open(filepath, "a") as f:
            if not content.endswith("\n"):
                f.write("\n")
            f.write(face_line)

        print(f"Updated {filename} with face {face_file}")


def add_world_sprites_to_rogues_toml():
    if not os.path.exists(rogues_dir):
        print(f"Error: Directory {rogues_dir} does not exist")
        sys.exit(1)

    updated_count = 0

    for enemy_code, full_name in enemies:
        shadow = shadow_name(enemy_code)
        filename = full_name.lower().replace(" ", "_") + ".toml"
        filepath = os.path.join(rogues_dir, filename)

        if not os.path.exists(filepath):
            print(f"Error: File {filepath} does not exist")
            continue

        # Read existing content
        with open(filepath, "r") as f:
            content = f.read()

        # Check if sprite lines already exist
        if "walking_sprite" in content or "walking_shadow_sprite" in content:
            print(f"Skipping {filename} - sprite lines already exist")
            continue

        # Append sprite lines
        sprite_lines = f'walking_sprite = "{enemy_code}.spr.png"\nwalking_shadow_sprite = "{shadow}.spr.png"\n'

        with open(filepath, "a") as f:
            if not content.endswith("\n"):
                f.write("\n")
            f.write(sprite_lines)

        print(f"Updated {filename}")
        updated_count += 1

    print(f"\nUpdated {updated_count} files")


def shadow_name(name):
    """Convert character name to shadow sprite name - matches Go ShadowName function"""
    xref = {
        "Kht": "Skht",
        "Djn": "Sdjn",
        "Fwz": "Sfwz",
        "Mwz": "Smwz",
        "Trl": "Strl",
        "Troll": "Strl",
        "Wrm": "Swrm",
        "Dg_": "S_Dg",
        "Ego_F": "Sego_F",
        "Ego_M": "Sego_M",
    }

    # Check special cases first
    for pattern, shadow in xref.items():
        if pattern in name:
            return shadow

    # Handle short names
    if len(name) < 4:
        return "S" + name

    # Handle underscore-separated names
    parts = name.split("_")
    if len(parts) < 2:
        return "S" + name

    prefix = parts[0]
    base = parts[1]

    shadow_prefix = "S"
    if prefix == "W":
        shadow_prefix = "Sw_"
        base = name[2:]  # Remove "W_" prefix
    elif prefix == "Bk":
        shadow_prefix = "Sb_"
    elif prefix == "Bu":
        shadow_prefix = "Su_"
    elif prefix == "G":
        shadow_prefix = "Sg_"
    elif prefix == "R":
        shadow_prefix = "Sr_"
    elif prefix == "M":
        shadow_prefix = "Sm_"

    return shadow_prefix + base


if __name__ == "__main__":
    add_faces_to_rogues_toml()
