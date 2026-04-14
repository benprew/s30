#!/usr/bin/env python3
"""Score Magic cards by power level using card attributes alone.

Computes Impact, Efficiency, and Reliability scores (from judging_magic_cards.md)
plus Quadrant Theory coverage (from quadrant_theory.md) to produce a composite
power score. No ML training — scores are derived purely from card mechanics.

Uses card_tiers.toml only for *validation* (Spearman rank correlation).

Usage:
    python3 utils/train_card_classifier.py                        # score all cards
    python3 utils/train_card_classifier.py --bottom 25            # show worst 25%
    python3 utils/train_card_classifier.py --top 10               # show best 10%
    python3 utils/train_card_classifier.py --type creature        # filter by type
    python3 utils/train_card_classifier.py --bottom 30 --type creature  # worst 30% of creatures
    python3 utils/train_card_classifier.py --output scores.tsv
"""

import argparse
import json
import re
import tomllib
from pathlib import Path

import numpy as np

ASSETS = Path(__file__).resolve().parent.parent / "assets"
TIERS_PATH = ASSETS / "configs" / "card_tiers.toml"
CARDS_PATH = ASSETS / "card_info" / "scryfall_cards.json"
PARSED_PATH = ASSETS / "card_info" / "parsed_cards.json"

TIER_ORDER = [
    "mandatory_cards",
    "almost_mandatory",
    "staples",
    "played_in_most_decks",
    "played_quite_often",
    "played_from_time_to_time",
    "played_in_specific_archetypes",
    "rarely_played",
    "almost_never_played",
    "meme_card",
]

NUM_TIERS = len(TIER_ORDER)
TIER_SCORES = {
    name: 100 * (NUM_TIERS - 1 - i) / (NUM_TIERS - 1)
    for i, name in enumerate(TIER_ORDER)
}


def parse_mana_cost(mana_cost: str) -> dict[str, float]:
    """Extract numeric features from a mana cost string like '{2}{U}{U}'."""
    if not mana_cost:
        return {"cmc": 0, "generic": 0, "colored": 0, "num_colors": 0, "has_x": 0}

    generic = 0
    colored = 0
    has_x = 0
    color_set: set[str] = set()

    for symbol in re.findall(r"\{([^}]+)\}", mana_cost):
        if symbol == "X":
            has_x = 1
        elif symbol.isdigit():
            generic = int(symbol)
        elif symbol in ("W", "U", "B", "R", "G"):
            colored += 1
            color_set.add(symbol)

    return {
        "cmc": generic + colored,
        "generic": generic,
        "colored": colored,
        "num_colors": len(color_set),
        "has_x": has_x,
    }


def parse_activation_cost(mana_str: str) -> int:
    """Count total mana in an activation cost string like '{1}{R}{R}'."""
    total = 0
    for symbol in re.findall(r"\{([^}]+)\}", mana_str):
        if symbol == "T":
            continue
        elif symbol.isdigit():
            total += int(symbol)
        elif symbol in ("W", "U", "B", "R", "G", "C"):
            total += 1
    return total


def extract_parsed_mechanics(abilities: list[dict]) -> dict:
    """Extract structured mechanics from parsed_cards.json abilities."""
    m: dict = {
        "has_destroy": False,
        "has_exile": False,
        "has_damage": False,
        "damage_amount": 0,
        "damage_target_type": "",
        "has_counter": False,
        "has_draw": False,
        "draw_amount": 0,
        "has_tutor": False,
        "has_discard": False,
        "has_pump": False,
        "pump_power": 0,
        "pump_toughness": 0,
        "pump_cost": 0,
        "has_mana_ability": False,
        "mana_types": [],
        "any_color_mana": False,
        "etb_counters": 0,
        "counter_power": 0,
        "counter_toughness": 0,
        "has_tap_target": False,
        "doesnt_untap": False,
        "activated_removal": False,
        "activated_removal_cost": 0,
        "activated_removal_needs_tap": False,
        "activated_removal_target": "",
        "has_board_wipe": False,
        "has_sacrifice_cost": False,
    }

    for a in abilities:
        atype = a.get("Type", "")
        effect = a.get("Effect") or {}
        cost = a.get("Cost") or {}
        target = a.get("TargetSpec") or {}
        raw = (a.get("RawText") or "").lower()

        # Mana abilities
        if atype == "Mana":
            m["has_mana_ability"] = True
            m["mana_types"] = effect.get("ManaTypes", [])
            m["any_color_mana"] = effect.get("AnyColor", False)

        # ETB counters (Triskelion, Clockwork Beast)
        if effect.get("ETBCounters"):
            m["etb_counters"] = effect["ETBCounters"]
            m["counter_power"] = effect.get("CounterPower", 0)
            m["counter_toughness"] = effect.get("CounterTough", 0)

        # Pump abilities
        if effect.get("PowerBoost") or effect.get("ToughnessBoost"):
            m["has_pump"] = True
            m["pump_power"] = effect.get("PowerBoost", 0)
            m["pump_toughness"] = effect.get("ToughnessBoost", 0)
            m["pump_cost"] = parse_activation_cost(cost.get("Mana", "")) if cost else 0

        # Doesn't untap
        if effect.get("DoesNotUntap"):
            m["doesnt_untap"] = True

        # Tap target
        if effect.get("TapTarget"):
            m["has_tap_target"] = True

        # Sacrifice cost
        if cost.get("Sacrifice"):
            m["has_sacrifice_cost"] = True

        # Destruction / damage via activated/spell abilities
        target_type = target.get("Type", "")

        if effect.get("Destroy") or "destroy" in raw:
            if "destroy all" in raw:
                m["has_board_wipe"] = True
            elif target_type:
                m["has_destroy"] = True
                if atype == "Activated":
                    m["activated_removal"] = True
                    m["activated_removal_cost"] = parse_activation_cost(cost.get("Mana", ""))
                    m["activated_removal_needs_tap"] = cost.get("Tap", False)
                    m["activated_removal_target"] = target_type

        if "exile" in raw and target_type:
            m["has_exile"] = True

        if effect.get("Amount") and target_type:
            m["has_damage"] = True
            m["damage_amount"] = max(m["damage_amount"], effect["Amount"])
            m["damage_target_type"] = target_type
            if atype == "Activated" and target_type == "any":
                m["activated_removal"] = True
                m["activated_removal_cost"] = parse_activation_cost(cost.get("Mana", ""))
                m["activated_removal_needs_tap"] = cost.get("Tap", False)
                m["activated_removal_target"] = "any"

        if "counter target spell" in raw:
            m["has_counter"] = True

        if "draw" in raw and "card" in raw:
            m["has_draw"] = True
            # Try to get amount
            dm = re.search(r"draws?\s+(a|\d+|two|three)", raw)
            if dm:
                word = dm.group(1)
                word_map = {"a": 1, "two": 2, "three": 3}
                m["draw_amount"] = max(
                    m["draw_amount"],
                    word_map.get(word) or (int(word) if word.isdigit() else 1),
                )

        if "search" in raw and "library" in raw:
            m["has_tutor"] = True

        if "discard" in raw and ("opponent" in raw or "player" in raw or "they" in raw):
            m["has_discard"] = True

    return m


def score_card(card: dict, parsed_abilities: list[dict] | None = None) -> dict[str, float]:
    """Compute Impact, Efficiency, Reliability, and Quadrant scores for a card.

    Each sub-score is 0-10, following the judging_magic_cards.md scale.
    Returns individual sub-scores and a composite total.
    """
    mana = parse_mana_cost(card["ManaCost"])
    cmc = mana["cmc"]
    colored = mana["colored"]
    num_colors = mana["num_colors"]

    # Extract structured mechanics from parsed data when available
    pm = extract_parsed_mechanics(parsed_abilities) if parsed_abilities else None

    tl = card["TypeLine"].lower()
    is_creature = "creature" in tl or "summon" in tl
    is_instant = "instant" in tl
    is_sorcery = "sorcery" in tl
    is_enchantment = "enchantment" in tl
    is_artifact = "artifact" in tl
    is_land = "land" in tl

    text = card["Text"] or ""
    text_lower = text.lower()
    card_name_lower = card["CardName"].lower()
    keywords = card["Keywords"] or []

    power_str = card["Power"]
    toughness_str = card["Toughness"]
    p = float(power_str) if power_str and power_str.isdigit() else 0
    t = float(toughness_str) if toughness_str and toughness_str.isdigit() else 0

    # Adjust effective P/T for +1/+1 counters (Triskelion, Clockwork Beast, etc.)
    if pm and pm["etb_counters"]:
        p += pm["etb_counters"] * pm["counter_power"]
        t += pm["etb_counters"] * pm["counter_toughness"]
        counters = pm["etb_counters"]
    else:
        counter_match = re.search(r"enters.*?with\s+(\w+)\s+\+1/\+1\s+counter", text_lower)
        if counter_match:
            word = counter_match.group(1)
            word_map = {"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7}
            counters = word_map.get(word) or (int(word) if word.isdigit() else 0)
            p += counters
            t += counters
        else:
            counters = 0

    mp = card.get("ManaProduction") or []
    colored_mana_produced = [m for m in mp if m in ("W", "U", "B", "R", "G")]

    has_flying = "Flying" in keywords or "flying" in text_lower
    has_first_strike = "First strike" in keywords or "first strike" in text_lower
    has_trample = "Trample" in keywords or "trample" in text_lower
    has_haste = "Haste" in keywords or "haste" in text_lower
    has_vigilance = "Vigilance" in keywords or "vigilance" in text_lower
    has_evasion = has_flying or bool(
        re.search(r"(fear|intimidate|menace|shadow|can't be blocked)", text_lower)
    )
    has_protection = bool(
        re.search(r"(hexproof|shroud|indestructible|protection from)", text_lower)
        or "Protection" in keywords
    )
    self_regenerates = is_creature and bool(
        re.search(
            r"regenerate\s+(this|~|" + re.escape(card_name_lower) + r")",
            text_lower,
        )
    )

    # ── Detect card mechanics ──

    exiles_creature = bool(
        re.search(r"exile\s+target\s+(\w+\s+)?creature", text_lower)
    )
    destroys_creature = bool(
        re.search(r"destroy\s+target\s+(\w+\s+)?creature", text_lower)
        or exiles_creature
    )
    destroys_artifact = bool(re.search(r"destroy\s+target\s+artifact", text_lower))
    destroys_land = bool(re.search(r"destroy\s+target\s+land", text_lower))
    destroys_permanent = bool(re.search(r"destroy\s+target\s+permanent", text_lower))
    destroys_enchantment = bool(
        re.search(
            r"destroy\s+target\s+(enchantment|aura|artifact or enchantment)",
            text_lower,
        )
    )
    board_wipe = bool(
        re.search(r"destroy\s+all\s+(creature|permanent|artifact|land)", text_lower)
    )

    # Repeatable abilities on permanents — split into hard removal vs pinging
    hard_removal_pattern = r"(destroy\s+target|exile\s+target)"
    ping_pattern = r"deals?\s+\d+\s+damage\s+to\s+any\s+target"
    activated_prefix = r"(?:\{[^}]+\}[\s,]*)+:\s*.*"

    is_permanent = is_creature or is_artifact or is_enchantment
    repeatable_hard_removal = is_permanent and bool(
        re.search(activated_prefix + hard_removal_pattern, text, re.I)
    )
    repeatable_ping = is_permanent and bool(
        re.search(activated_prefix + ping_pattern, text, re.I)
    )
    repeatable_removal = repeatable_hard_removal or repeatable_ping

    # Activation cost for repeatable abilities: count mana symbols before the colon
    activation_cost = 0
    needs_tap = False
    if repeatable_removal:
        pattern = hard_removal_pattern if repeatable_hard_removal else ping_pattern
        act_match = re.search(
            r"^((?:\{[^}]+\}[\s,]*)+):\s*.*" + pattern, text, re.MULTILINE | re.I
        )
        if act_match:
            symbols = re.findall(r"\{([^}]+)\}", act_match.group(1))
            for s in symbols:
                if s == "T":
                    needs_tap = True
                elif s.isdigit():
                    activation_cost += int(s)
                elif s in ("W", "U", "B", "R", "G", "C"):
                    activation_cost += 1

    dmg_match = re.search(r"deals?\s+(\d+)\s+damage", text_lower)
    damage_amount = int(dmg_match.group(1)) if dmg_match else 0
    damage_any_target = bool(
        re.search(r"deals?\s+\d+\s+damage\s+to\s+any\s+target", text_lower)
    )

    # Repeatable/splittable damage (Triskelion: remove counter for 1 damage x N)
    counter_dmg_match = re.search(
        r"remove.*counter.*deals?\s+(\d+)\s+damage\s+to\s+any\s+target", text_lower
    )
    if counter_dmg_match and counters:
        damage_amount = max(damage_amount, int(counter_dmg_match.group(1)) * counters)
        damage_any_target = True
    damage_to_creature = bool(
        re.search(
            r"deals?\s+\d+\s+damage\s+to\s+target\s+(creature|attacking|blocking)",
            text_lower,
        )
    )

    hard_counter = bool(
        re.search(r"counter\s+target\s+spell(?!\s+with|\s+unless)", text_lower)
    )
    soft_counter = bool(
        re.search(r"counter\s+target\s+spell\s+(with|unless)", text_lower)
    )

    draw_match = re.search(
        r"draws?\s+(a|\d+|two|three|four|five|six|seven)\s+card", text_lower
    )
    if draw_match:
        word = draw_match.group(1)
        word_map = {
            "a": 1,
            "two": 2,
            "three": 3,
            "four": 4,
            "five": 5,
            "six": 6,
            "seven": 7,
        }
        cards_drawn = word_map.get(word) or (int(word) if word.isdigit() else 1)
    else:
        cards_drawn = 0

    tutors = bool(re.search(r"search\s+(your|their)\s+library", text_lower))
    forces_discard = bool(
        re.search(r"(opponent|player|they)\s+discard", text_lower)
        or re.search(r"discard\s+(a|two|three|\d+)\s+card", text_lower)
    )
    gains_control = bool(re.search(r"gain\s+control\s+of\s+target", text_lower))
    extra_turn = bool(re.search(r"extra\s+turn", text_lower))
    adds_mana = bool(
        re.search(r"add\s+\{", text_lower)
        or re.search(r"add\s+(one|two|three)\s+mana", text_lower)
    )

    # How much mana does this card produce per activation?
    mana_symbols_added = len(re.findall(r"add\s+(\{[WUBRGC]\})+", text))
    any_color_match = re.search(r"add\s+(one|two|three)\s+mana\s+of\s+any", text_lower)
    if any_color_match:
        word_map = {"one": 1, "two": 2, "three": 3}
        mana_symbols_added = max(
            mana_symbols_added, word_map.get(any_color_match.group(1), 1)
        )

    # Pump abilities: {cost}: gets +P/+T
    pump_match = re.search(
        r"\{([^}]+)\}:\s*(?:this creature |it )?gets?\s+\+(\d+)/\+(\d+)", text, re.I
    )
    pump_power = 0
    pump_toughness = 0
    pump_cost = 0
    if pump_match:
        pump_symbol = pump_match.group(1)
        pump_power = int(pump_match.group(2))
        pump_toughness = int(pump_match.group(3))
        if pump_symbol.isdigit():
            pump_cost = int(pump_symbol)
        else:
            pump_cost = 1  # single colored mana

    # Chaos Orb: unique removal that hits any nontoken permanent
    destroys_on_flip = bool(re.search(r"flip.*destroy", text_lower))

    # Downsides that reduce effective stats
    doesnt_untap = bool(re.search(r"doesn't untap", text_lower))
    cumulative_upkeep = bool(re.search(r"cumulative upkeep", text_lower))
    # Self-damage on activated abilities (Orcish Artillery, Psionic Entity)
    activation_self_damage = re.search(
        r"and\s+(\d+)\s+damage\s+to\s+you", text_lower
    )
    activation_self_damage_amt = int(activation_self_damage.group(1)) if activation_self_damage else 0
    # Opponent-choice drawback (Cuombajj Witches — opponent gets to ping something)
    opponent_choice_drawback = bool(re.search(r"opponent'?s?\s+choice", text_lower))

    # Upkeep costs: scaled by severity
    # 0 = no upkeep cost, 1 = trivial, 2 = moderate, 3 = severe
    upkeep_cost = 0
    upkeep_text = ""
    upkeep_match = re.search(r"(at the beginning of your upkeep|during your upkeep),?\s*(.*?)(?:\.|$)", text_lower)
    if upkeep_match:
        upkeep_text = upkeep_match.group(2)

    if upkeep_text:
        if re.search(r"sacrifice a creature", upkeep_text):
            upkeep_cost = 3  # Lord of the Pit: must feed it creatures
        elif re.search(r"sacrifice a land", upkeep_text):
            upkeep_cost = 2  # Serendib Djinn: lose lands over time
        elif re.search(r"sacrifice an artifact", upkeep_text):
            upkeep_cost = 2  # Yawgmoth Demon: needs artifact support
        elif re.search(r"deals?\s+(\d+)\s+damage\s+to\s+you\s+unless", upkeep_text):
            # Conditional damage: depends on mana payment (Force of Nature)
            upkeep_cost = 2
        elif re.search(r"deals?\s+[1-2]\s+damage\s+to\s+you", upkeep_text):
            upkeep_cost = 1  # Juzám Djinn, Serendib Efreet: trivial ping
        elif re.search(r"deals?\s+\d+\s+damage\s+to\s+you", upkeep_text):
            upkeep_cost = 2  # Larger self-damage

    narrow_targeting = bool(
        re.search(
            r"target\s+(Elephant|Wall|Djinn|Efreet|Goblin|Merfolk|Zombie|Knight|Vampire)",
            text,
            re.I,
        )
        and not re.search(
            r"target\s+(creature|permanent|player|opponent|artifact|land|enchantment)",
            text_lower,
        )
    )
    # "You can't die" effects (Ali from Cairo, Worship-style)
    prevents_death = bool(
        re.search(
            r"(life total.*(less than|reduce.*to) (1|0)|can't lose the game)",
            text_lower,
        )
    )

    gains_life_only = bool(re.search(r"gain[s]?\s+\d*\s*life", text_lower)) and not (
        destroys_creature
        or damage_amount
        or hard_counter
        or cards_drawn
        or board_wipe
        or forces_discard
        or adds_mana
        or tutors
    )
    vintage_restricted = card.get("VintageRestricted", False)

    # ── Override regex detections with parsed data when available ──
    if pm:
        if pm["has_destroy"] or pm["has_exile"]:
            destroys_creature = True
        if pm["has_board_wipe"]:
            board_wipe = True
        if pm["has_damage"]:
            damage_amount = max(damage_amount, pm["damage_amount"])
            if pm["damage_target_type"] == "any":
                damage_any_target = True
            elif pm["damage_target_type"] == "creature":
                damage_to_creature = True
        if pm["has_counter"]:
            hard_counter = True
        if pm["has_draw"]:
            cards_drawn = max(cards_drawn, pm["draw_amount"])
        if pm["has_tutor"]:
            tutors = True
        if pm["has_discard"]:
            forces_discard = True
        if pm["has_mana_ability"]:
            adds_mana = True
        if pm["doesnt_untap"]:
            doesnt_untap = True
        if pm["has_pump"]:
            pump_power = pm["pump_power"]
            pump_toughness = pm["pump_toughness"]
            pump_cost = max(pm["pump_cost"], 1)
        if pm["activated_removal"]:
            repeatable_removal = True
            activation_cost = pm["activated_removal_cost"]
            needs_tap = pm["activated_removal_needs_tap"]

    # ── IMPACT (0-10) ──
    # How much does this card alter the game state?

    impact = 3.0  # baseline: exists, does something minor

    if board_wipe:
        impact += 5.0
    if extra_turn:
        impact += 5.0
    if gains_control:
        impact += 3.0
    if prevents_death:
        impact += 5.0

    # Card advantage
    if cards_drawn >= 3:
        impact += 4.0
    elif cards_drawn == 2:
        impact += 2.5
    elif cards_drawn == 1:
        impact += 1.5
    if tutors:
        impact += 3

    # Removal — repeatable hard removal (destroy/exile) is very powerful,
    # repeatable pinging (deal 1-2 damage) is useful but much weaker.
    if repeatable_hard_removal:
        # Royal Assassin tier: answers any creature every turn
        if activation_cost == 0:
            hard_removal_bonus = 4.0
        elif activation_cost <= 2:
            hard_removal_bonus = 2.5
        else:
            hard_removal_bonus = 1.0
        if opponent_choice_drawback:
            hard_removal_bonus *= 0.5

    ping_bonus = 0.0
    if repeatable_ping:
        # Tim tier: 1 damage per turn is useful but doesn't kill big threats
        if activation_cost == 0:
            ping_bonus = 2.0
        elif activation_cost <= 2:
            ping_bonus = 1.0
        else:
            ping_bonus = 0.5
        if opponent_choice_drawback:
            ping_bonus *= 0.5

    if destroys_creature or destroys_permanent:
        if repeatable_hard_removal:
            impact += hard_removal_bonus
        else:
            impact += 2.0
    elif destroys_artifact or destroys_enchantment or destroys_land:
        impact += 1.5
    if damage_any_target and damage_amount >= 3:
        impact += 2.5
    elif damage_any_target:
        impact += ping_bonus if repeatable_ping else 1.5
    elif damage_to_creature:
        impact += 1.0

    # Counterspells
    if hard_counter:
        impact += 2.5
    elif soft_counter:
        impact += 1.5

    # Disruption
    if forces_discard:
        impact += 1.5

    # Creature threat level — Defender can never win the game
    is_defender = "Defender" in keywords or "defender" in text_lower or "wall" in tl
    if is_creature:
        if is_defender:
            impact -= 2.0  # can't attack, can't close games
        elif p >= 5 and has_evasion:
            impact += 3.0
        elif p >= 4 and has_evasion:
            impact += 2.0
        elif p >= 3 and has_evasion:
            impact += 1.0
        elif p >= 4:
            impact += 1.0
        # Small creatures (p<2, t<=2) are outclassed in a format full of 2/2s.
        # Exception: utility creatures whose value is their ability, not body,
        # and creatures with relevant combat keywords (evasion, first strike).
        is_utility_creature = (
            adds_mana  # mana dorks (Birds, Elves)
            or repeatable_removal  # Royal Assassin, Prodigal Sorcerer
            or forces_discard  # Hypnotic Specter
            or cards_drawn > 0
            or tutors
            or has_evasion  # Flying Men, Scryb Sprites still trade up
            or has_first_strike  # Tundra Wolves blocks 2/2s
        )
        if p < 2 and t <= 2 and not is_utility_creature:
            impact -= 2.0

    # Mana acceleration: more mana produced = more impact
    if is_land and len(colored_mana_produced) >= 2:
        impact += 3.0
    elif adds_mana:
        if mana_symbols_added >= 3:
            impact += 4.0  # Black Lotus, Dark Ritual
        elif mana_symbols_added >= 2:
            impact += 2.0  # Sol Ring
        else:
            impact += 1.0  # Mox, Llanowar Elves

    # Chaos Orb: unique unconditional removal
    if destroys_on_flip:
        impact += 4.0

    # Penalties
    if narrow_targeting:
        impact -= 2.0
    if gains_life_only:
        impact -= 1.5

    if vintage_restricted:
        impact += 2.0

    impact = max(1.0, min(10.0, impact))

    # ── EFFICIENCY (0-10) ──
    # What do you get for the mana spent?

    efficiency = 5.0  # baseline: average

    if is_creature and cmc > 0:
        total_stats = p + t

        # Pump abilities add effective stats, but discounted since they cost mana.
        # Compare to the total mana investment (cmc + pump mana) not just cmc.
        pump_mana = 0
        if pump_power > 0 or pump_toughness > 0:
            pumps_possible = 3 // max(pump_cost, 1)
            total_stats += pumps_possible * (pump_power + pump_toughness)
            pump_mana = pumps_possible * max(pump_cost, 1)

        # Vanilla test: total stats vs total mana investment (cast + pump)
        total_mana = cmc + pump_mana
        expected = total_mana * 2
        stat_ratio = total_stats / expected if expected > 0 else 0
        # Scale: ratio of 1.0 = average (5), 1.5 = great (7.5), 0.5 = bad (2.5)
        efficiency = 5.0 * stat_ratio

        # Keyword bonuses on top of stats
        kw_bonus = 0
        if has_flying:
            kw_bonus += 1.5
        if has_first_strike:
            kw_bonus += 1.0
        if has_trample:
            kw_bonus += 0.5
        if has_haste:
            kw_bonus += 0.5
        if has_vigilance:
            kw_bonus += 0.5
        if has_protection:
            kw_bonus += 1.0
        if self_regenerates:
            kw_bonus += 0.5
        # Mana dorks: efficiency comes from the mana ability, not stats
        if adds_mana:
            kw_bonus += 3.0
        # Downsides that eat into effective value
        if doesnt_untap:
            kw_bonus -= 3.0  # effectively costs mana every turn to use
        if cumulative_upkeep:
            kw_bonus -= 2.0
        if upkeep_cost == 1:
            kw_bonus -= 0.5  # trivial ping (Juzám, Serendib Efreet)
        elif upkeep_cost == 2:
            kw_bonus -= 1.5  # real cost (sacrifice land/artifact, conditional dmg)
        elif upkeep_cost >= 3:
            kw_bonus -= 3.0  # severe (sacrifice creatures)
        if activation_self_damage_amt >= 3:
            kw_bonus -= 1.5  # significant self-harm per use
        elif activation_self_damage_amt >= 1:
            kw_bonus -= 0.5
        # Small bodies trade poorly in a format of 2/2s
        if p < 2 and t <= 2 and not is_utility_creature:
            kw_bonus -= 1.5
        # Defenders: power is wasted since they can't attack
        if is_defender:
            kw_bonus -= 1.5
        efficiency += kw_bonus

    elif is_creature and cmc == 0:
        # Free creature: great efficiency if it has stats
        efficiency = 7.0 if (p + t) >= 2 else 4.0

    elif is_land:
        if len(colored_mana_produced) >= 2:
            efficiency = 9.0  # dual land, no mana cost
        elif len(colored_mana_produced) == 1 and len(mp) > 1:
            efficiency = 7.0  # utility land producing colored
        elif len(mp) > 0:
            efficiency = 5.0  # colorless land
        else:
            efficiency = 3.0  # no mana production
        # Utility lands that also remove things are extremely efficient
        # (the effect is free — no card slot cost beyond the land itself)
        if destroys_creature or destroys_land or destroys_permanent or destroys_on_flip:
            efficiency += 2.0
        if cards_drawn > 0:
            efficiency += 2.0

    else:
        # Spells: cheaper = more efficient for the same effect
        if cmc == 0:
            efficiency = 9.0  # free spell
        elif cmc == 1:
            efficiency = 7.0
        elif cmc == 2:
            efficiency = 6.0
        elif cmc == 3:
            efficiency = 5.0
        elif cmc <= 5:
            efficiency = 4.0
        else:
            efficiency = 3.0

        # Damage efficiency
        if damage_amount > 0 and cmc > 0:
            dmg_per_mana = damage_amount / cmc
            if dmg_per_mana >= 3:
                efficiency += 2.0  # Lightning Bolt tier
            elif dmg_per_mana >= 2:
                efficiency += 1.0

        # Card draw efficiency
        if cards_drawn > 0 and cmc > 0:
            draw_per_mana = cards_drawn / cmc
            if draw_per_mana >= 3:
                efficiency += 3.0  # Ancestral Recall tier
            elif draw_per_mana >= 1:
                efficiency += 1.5

        # Mana rocks that pay for themselves quickly
        if adds_mana and is_artifact and cmc <= 2:
            efficiency += 2.0

    # Colored mana penalty: harder to cast = less efficient
    if num_colors >= 3:
        efficiency -= 1.5
    elif num_colors >= 2:
        efficiency -= 0.5

    efficiency = max(1.0, min(10.0, efficiency))

    # ── RELIABILITY (0-10) ──
    # How often is this card useful? How many situations does it cover?

    reliability = 4.0  # baseline

    # Creatures are inherently versatile (attack + block)
    if is_creature:
        reliability += 1.0
        if has_vigilance:
            reliability += 0.5
        if has_evasion:
            reliability += 0.5

    # Instant speed = more flexible
    if is_instant or "flash" in text_lower:
        reliability += 1.0

    # Repeatable removal/ping: useful in every game state
    if repeatable_hard_removal:
        rep_rel = 0.5
        if activation_cost == 0:
            rep_rel = 2.0
        elif activation_cost <= 2:
            rep_rel = 1.0
        if opponent_choice_drawback:
            rep_rel *= 0.5
        reliability += rep_rel
    elif repeatable_ping:
        rep_rel = 0.25
        if activation_cost == 0:
            rep_rel = 1.0
        elif activation_cost <= 2:
            rep_rel = 0.5
        if opponent_choice_drawback:
            rep_rel *= 0.5
        reliability += rep_rel

    # Multiple abilities = more modes
    num_abilities = text.count("\n") + (1 if text else 0)
    if num_abilities >= 3:
        reliability += 1.5
    elif num_abilities >= 2:
        reliability += 0.5

    # Self-protection means it sticks around
    if has_protection or self_regenerates:
        reliability += 1.0
    if prevents_death:
        reliability += 1.5

    # Broad answers are more reliable than narrow ones
    if hard_counter:
        reliability += 2.0  # answers anything
    elif soft_counter:
        reliability += 1.0
    if destroys_on_flip:
        reliability += 2.0  # Chaos Orb: hits anything
    elif destroys_permanent:
        reliability += 2.0
    elif destroys_creature:
        reliability += 1.0
    if damage_any_target:
        reliability += 1.0  # hits creatures or players

    # Mana producers: always useful
    if adds_mana:
        reliability += 1.0
    if is_land and len(colored_mana_produced) >= 2:
        reliability += 2.0

    # Card draw: always useful
    if cards_drawn > 0:
        reliability += 1.0

    # X spells scale with the game
    if mana["has_x"]:
        reliability += 0.5

    # Penalties for narrow effects and downsides
    if narrow_targeting:
        reliability -= 3.0
    if gains_life_only:
        reliability -= 1.0
    if doesnt_untap:
        reliability -= 1.5
    if cumulative_upkeep:
        reliability -= 1.0
    if upkeep_cost == 2:
        reliability -= 0.5  # needs support or careful play
    elif upkeep_cost >= 3:
        reliability -= 1.5  # needs dedicated deck support

    # Colored mana makes it less splashable
    if colored >= 3:
        reliability -= 1.0
    elif num_colors >= 2:
        reliability -= 0.5

    # Defender/Wall can't attack — only useful defensively
    if is_defender:
        reliability -= 2.0

    reliability = max(1.0, min(10.0, reliability))

    # ── QUADRANT THEORY BONUS (0-4) ──
    # Cards good in multiple quadrants get a bonus.

    quadrants_good = 0

    # Developing: cheap plays that establish board presence
    if (is_creature and 1 <= cmc <= 3 and (p + t) >= 2) or (
        cmc <= 2 and damage_amount >= 2
    ):
        quadrants_good += 1
    if adds_mana and cmc <= 2:
        quadrants_good += 1

    # Parity: breaks stalls
    if cards_drawn >= 2 or board_wipe or (is_creature and p >= 4 and has_evasion):
        quadrants_good += 1
    if tutors or forces_discard:
        quadrants_good += 1

    # Winning: closes the game
    if is_creature and p >= 3 and has_evasion:
        quadrants_good += 1
    if damage_any_target and damage_amount >= 3:
        quadrants_good += 1

    # Losing: stabilizes
    if board_wipe:
        quadrants_good += 1
    if is_creature and t >= 4 and cmc <= 4:
        quadrants_good += 1
    if destroys_creature or destroys_permanent:
        quadrants_good += 1
    if prevents_death:
        quadrants_good += 1

    quadrant_bonus = min(4.0, quadrants_good * 0.8)

    # ── HIGH CMC PENALTY ──
    # In Old School, expensive cards need to nearly win the game on the spot.
    # A 5+ CMC card that doesn't dominate is too slow for the format.
    # Reduced for genuine finishers (high power + evasion/trample) and
    # game-ending effects (board wipes, extra turns, etc.)

    is_finisher = (
        (is_creature and p >= 5 and (has_evasion or has_trample))
        or board_wipe
        or extra_turn
        or cards_drawn >= 3
    )

    high_cmc_penalty = 0.0
    if cmc >= 7:
        high_cmc_penalty = 3.0 if is_finisher else 6.0
    elif cmc >= 6:
        high_cmc_penalty = 2.0 if is_finisher else 4.0
    elif cmc >= 5:
        high_cmc_penalty = 1.0 if is_finisher else 2.0

    # ── COMPOSITE ──

    # Total out of 34 (impact 10 + efficiency 10 + reliability 10 + quadrant 4)
    raw = impact + efficiency + reliability + quadrant_bonus - high_cmc_penalty
    # Normalize to 0-100
    composite = max(0.0, (raw / 34.0) * 100.0)

    return {
        "impact": round(impact, 1),
        "efficiency": round(efficiency, 1),
        "reliability": round(reliability, 1),
        "quadrant_bonus": round(quadrant_bonus, 1),
        "score": round(composite, 1),
    }


def build_name_to_tier() -> dict[str, str]:
    """Map card names to their tier label from the TOML."""
    with open(TIERS_PATH, "rb") as f:
        tiers = tomllib.load(f)
    result: dict[str, str] = {}
    for tier_name in TIER_ORDER:
        for card_name in tiers.get(tier_name, []):
            result[card_name] = tier_name
    return result


def load_parsed_abilities() -> dict[str, list[dict]]:
    """Load parsed card abilities, keyed by card name."""
    if not PARSED_PATH.exists():
        return {}
    with open(PARSED_PATH) as f:
        parsed = json.load(f)
    return {c["card_name"]: c["abilities"] for c in parsed}


def score_all_cards() -> list[tuple[str, float, str, dict[str, float]]]:
    """Score every non-basic card.

    Returns (name, composite, tier_label, sub_scores) sorted best-to-worst.
    """
    with open(CARDS_PATH) as f:
        cards_list = json.load(f)

    name_to_tier = build_name_to_tier()
    parsed = load_parsed_abilities()

    results: list[tuple[str, float, str, dict[str, float]]] = []
    seen: set[str] = set()
    for card in cards_list:
        name = card["CardName"]
        if name in seen:
            continue
        if "Basic Land" in card["TypeLine"]:
            continue
        seen.add(name)

        scores = score_card(card, parsed.get(name))
        scores["type_line"] = card["TypeLine"]
        tier = name_to_tier.get(name, "")
        results.append((name, scores["score"], tier, scores))

    results.sort(key=lambda r: -r[1])
    return results


def validate(results: list[tuple[str, float, str, dict[str, float]]]) -> None:
    """Compare computed scores against tier labels using rank correlation."""
    from scipy.stats import spearmanr

    labeled = [(name, score, tier) for name, score, tier, _ in results if tier]
    if not labeled:
        print("No tier labels found for validation.")
        return

    pred_scores = [s for _, s, _ in labeled]
    tier_scores = [TIER_SCORES[t] for _, _, t in labeled]

    rho, pval = spearmanr(tier_scores, pred_scores)
    print(f"\nValidation against card_tiers.toml ({len(labeled)} labeled cards):")
    print(f"  Spearman rho: {rho:.3f}  (p={pval:.2e})")

    mae = np.mean(np.abs(np.array(pred_scores) - np.array(tier_scores)))
    print(f"  MAE vs tier scores: {mae:.1f} points")

    # Quartile agreement
    pred_arr = np.array(pred_scores)
    tier_arr = np.array(tier_scores)
    for pct in [25, 50]:
        true_bottom = tier_arr <= np.percentile(tier_arr, pct)
        pred_bottom = pred_arr <= np.percentile(pred_arr, pct)
        agreement = np.mean(true_bottom == pred_bottom)
        print(f"  Bottom-{pct}% agreement: {agreement:.1%}")

    # Show biggest misranks
    diffs = [(name, score, TIER_SCORES[tier], tier) for name, score, tier in labeled]
    diffs.sort(key=lambda x: abs(x[1] - x[2]), reverse=True)
    print(f"\nBiggest misranks (top 10):")
    print(f"  {'Card':30s} {'Computed':>8s} {'Tier':>8s} {'Diff':>6s}  Tier label")
    for name, comp, tier_sc, tier_name in diffs[:10]:
        diff = comp - tier_sc
        print(f"  {name:30s} {comp:8.1f} {tier_sc:8.1f} {diff:+6.1f}  {tier_name}")


def print_ranked(
    results: list[tuple[str, float, str, dict[str, float]]],
    top_pct: float | None,
    bottom_pct: float | None,
    type_filter: str = "",
) -> None:
    """Print a ranked card list, optionally filtered by percentile and/or card type."""
    # Apply type filter first
    if type_filter:
        tf = type_filter.lower()
        results = [r for r in results if tf in r[3].get("type_line", "").lower()]
        # Re-sort after filtering
        results.sort(key=lambda r: -r[1])

    n = len(results)
    all_scores = [s for _, s, _, _ in results]

    label_parts = []
    if type_filter:
        label_parts.append(f"type={type_filter}")

    if bottom_pct is not None:
        threshold = np.percentile(all_scores, bottom_pct)
        filtered = [r for r in results if r[1] <= threshold]
        label_parts.append(f"bottom {bottom_pct:.0f}%, score <= {threshold:.1f}")
        filtered.sort(key=lambda r: r[1])
    elif top_pct is not None:
        threshold = np.percentile(all_scores, 100 - top_pct)
        filtered = [r for r in results if r[1] >= threshold]
        label_parts.append(f"top {top_pct:.0f}%, score >= {threshold:.1f}")
        filtered.sort(key=lambda r: -r[1])
    else:
        filtered = results
        label_parts.append(f"all {n} cards")

    print(f"\n--- {', '.join(label_parts)} ---")

    print(
        f"{'Rank':>5}  {'Score':>5}  {'Imp':>4} {'Eff':>4} {'Rel':>4} {'Q':>3}"
        f"  {'Tier':<28s}  Card"
    )
    print(
        f"{'----':>5}  {'-----':>5}  {'---':>4} {'---':>4} {'---':>4} {'--':>3}"
        f"  {'----':<28s}  ----"
    )

    name_to_rank = {name: i + 1 for i, (name, _, _, _) in enumerate(results)}
    for name, score, tier, sub in filtered:
        rank = name_to_rank[name]
        tier_display = tier if tier else "-"
        print(
            f"{rank:>5}  {score:>5.1f}  {sub['impact']:>4.1f} {sub['efficiency']:>4.1f}"
            f" {sub['reliability']:>4.1f} {sub['quadrant_bonus']:>3.1f}"
            f"  {tier_display:<28s}  {name}"
        )

    print(f"\n{len(filtered)} cards shown out of {n} total")


def write_scores(
    results: list[tuple[str, float, str, dict[str, float]]],
    output_path: str,
) -> None:
    """Write scores to a TSV file."""
    with open(output_path, "w") as f:
        f.write(
            "rank\tscore\timpact\tefficiency\treliability\tquadrant\ttier\tcard_name\n"
        )
        for i, (name, score, tier, sub) in enumerate(results, 1):
            f.write(
                f"{i}\t{score:.1f}\t{sub['impact']:.1f}\t{sub['efficiency']:.1f}"
                f"\t{sub['reliability']:.1f}\t{sub['quadrant_bonus']:.1f}"
                f"\t{tier}\t{name}\n"
            )
    print(f"\nWrote {len(results)} card scores to {output_path}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Score Magic cards by power level")
    parser.add_argument(
        "--top",
        type=float,
        metavar="PCT",
        help="Show only the top N%% of cards",
    )
    parser.add_argument(
        "--bottom",
        type=float,
        metavar="PCT",
        help="Show only the bottom N%% of cards",
    )
    parser.add_argument(
        "--type",
        type=str,
        default="",
        help="Filter by card type (creature, instant, sorcery, enchantment, artifact, land)",
    )
    parser.add_argument(
        "--output",
        type=str,
        default="",
        help="Write full ranked list to a TSV file",
    )
    args = parser.parse_args()

    print("Scoring all cards...")
    results = score_all_cards()
    print(f"Scored {len(results)} cards")

    validate(results)
    print_ranked(results, top_pct=args.top, bottom_pct=args.bottom, type_filter=args.type)

    if args.output:
        write_scores(results, args.output)


if __name__ == "__main__":
    main()
