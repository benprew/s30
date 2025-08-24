#!/usr/bin/env python3
import re
import sys, json
from collections import defaultdict

def q(s):  # safely quote & escape like JSON
    return json.dumps(s, ensure_ascii=False)


main_cards = defaultdict(str)
sideboard_cards = defaultdict(str)

deck = main_cards

header = ""

for line in sys.stdin if len(sys.argv) == 1 else open(sys.argv[1], "r", encoding="utf-8"):
    if header == "":
        header = line.rstrip("\n")
        continue

    line = line.rstrip("\n")
    if ".vNone" in line:
        deck = sideboard_cards

        
    if not line:
        continue
    parts = line.split("\t", 2)
    if len(parts) < 3:
        continue
    count = parts[1].strip()
    name = parts[2].strip()

    deck[name] = max(deck[name], count)

name = header.split(" (")[0]

# regex replace all non alphanumeric characters with _
filename = f"{re.sub(r'\W+', '_', name).lower()}.toml"

with(open(filename, "w", encoding="utf-8")) as f:
    # print(f"###########\n{header} - {filename}\n###########", file=f)
    print(f"name = \"{name}\"", file=f)
    print("main_cards = [", file=f)
    for card, count in main_cards.items():
        print(f'[{q(count)},\t{q(card)}],', file=f)

    print("]", file=f)
    print("sideboard_cards = [", file=f)
    for card, count in sideboard_cards.items():
        print(f'[{q(count)},\t{q(card)}],', file=f)
    print("]", file=f)
