#!/usr/bin/env python3

import csv
import re
import time
from collections import defaultdict
from typing import Dict, List, Optional

import requests
from bs4 import BeautifulSoup

HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 "
        "(KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
    )
}

BASE_URL = "https://mtgdecks.net"
OLD_SCHOOL_TOURNAMENTS_URL = "https://mtgdecks.net/Old-school/tournaments"


def get_tournament_deck_urls(limit_tournaments: int = 5) -> List[Dict[str, str]]:
    """Scrapes tournament listing pages to collect individual deck links."""
    resp = requests.get(OLD_SCHOOL_TOURNAMENTS_URL, headers=HEADERS)
    resp.raise_for_status()
    soup = BeautifulSoup(resp.text, "html.parser")

    deck_links: List[Dict[str, str]] = []
    # Find tournament table links
    t_links: List[str] = []
    for a in soup.find_all("a", href=True):
        href = str(a["href"])
        if "/Old-school/" in href and "-tournament-" in href:
            full_t_url = BASE_URL + href if href.startswith("/") else href
            if full_t_url not in t_links:
                t_links.append(full_t_url)
        if len(t_links) >= limit_tournaments:
            break

    # Extract deck links within each tournament
    for t_url in t_links:
        time.sleep(1.0)  # Courtesy delay
        t_resp = requests.get(t_url, headers=HEADERS)
        if t_resp.status_code != 200:
            continue
        t_soup = BeautifulSoup(t_resp.text, "html.parser")

        for row in t_soup.select("tr"):
            deck_a = row.find("a", href=re.compile(r"/Old-school/[^/]+-decklist-\d+"))
            if deck_a and isinstance(deck_a.get("href"), str):
                deck_href = str(deck_a["href"])
                deck_url = BASE_URL + deck_href
                deck_name = deck_a.get_text(strip=True)
                deck_links.append(
                    {"name": deck_name, "url": deck_url, "tournament": t_url}
                )

    return deck_links


def parse_deck_text(deck_url: str) -> Optional[Dict[str, int]]:
    """
    Fetches the deck text export from MTGDecks and extracts card counts.
    MTGDecks deck pages provide a direct text download endpoint via /dec or /txt.
    """
    # MTGDecks exposes text exports by appending /dec or /txt to the deck URL
    txt_url = f"{deck_url}/dec"
    resp = requests.get(txt_url, headers=HEADERS)

    # Fallback to direct page parsing if export endpoint is unavailable
    if resp.status_code != 200 or not resp.text.strip():
        resp = requests.get(deck_url, headers=HEADERS)
        if resp.status_code != 200:
            return None
        soup = BeautifulSoup(resp.text, "html.parser")
        cards = defaultdict(int)
        for row in soup.select(".cardItem, .deck-card"):
            qty = row.find(class_=re.compile("number|qty"))
            name = row.find(class_=re.compile("name|card"))
            if qty and name:
                try:
                    q = int(qty.get_text(strip=True).replace("x", ""))
                    cards[name.get_text(strip=True)] += q
                except ValueError:
                    continue
        return dict(cards) if cards else None

    # Parse plain text format: "<qty> <Card Name>" or "SB: <qty> <Card Name>"
    cards = defaultdict(int)
    for line in resp.text.splitlines():
        line = line.strip()
        if not line or line.startswith("//"):
            continue

        # Strip sideboard prefix if present to count global copies in 75
        match = re.match(r"^(?:SB:\s*)?(\d+)\s+([A-Za-z0-9\s',/\-]+)$", line)
        if match:
            qty = int(match.group(1))
            name = match.group(2).strip()
            cards[name] += qty

    return dict(cards) if cards else None


def build_datasets(deck_records: List[Dict]):
    """Generates a card-frequency aggregation and a deck-by-card feature matrix."""
    card_totals = defaultdict(int)
    card_deck_counts = defaultdict(int)
    total_decks = len(deck_records)

    all_cards = set()
    for rec in deck_records:
        for card, qty in rec["cards"].items():
            card_totals[card] += qty
            card_deck_counts[card] += 1
            all_cards.add(card)

    all_cards = sorted(list(all_cards))

    # 1. Summary CSV: Global frequency, meta share, average copies when played
    with open("oldschool_card_frequencies.csv", "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(
            [
                "card_name",
                "total_copies",
                "decks_included",
                "inclusion_rate_pct",
                "avg_copies_when_present",
            ]
        )
        for card in sorted(
            card_totals.keys(), key=lambda c: card_totals[c], reverse=True
        ):
            deck_count = card_deck_counts[card]
            pct = round((deck_count / total_decks) * 100, 2)
            avg_copies = round(card_totals[card] / deck_count, 2)
            writer.writerow([card, card_totals[card], deck_count, pct, avg_copies])

    # 2. Matrix CSV: Decks as rows, cards as feature columns (for ML/regression)
    with open("oldschool_deck_matrix.csv", "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["deck_name", "url"] + all_cards)
        for rec in deck_records:
            row = [rec["name"], rec["url"]]
            for card in all_cards:
                row.append(rec["cards"].get(card, 0))
            writer.writerow(row)

    print(f"Exported {total_decks} decks across {len(all_cards)} unique cards.")


def main():
    print("Collecting tournament deck links...")
    deck_meta = get_tournament_deck_urls(limit_tournaments=3)
    print(f"Found {len(deck_meta)} deck listings. Parsing deck contents...")

    parsed_decks = []
    for meta in deck_meta:
        time.sleep(0.8)
        cards = parse_deck_text(meta["url"])
        if cards:
            parsed_decks.append(
                {"name": meta["name"], "url": meta["url"], "cards": cards}
            )

    if parsed_decks:
        build_datasets(parsed_decks)
    else:
        print("No deck data extracted. Verify network connectivity or target URLs.")


if __name__ == "__main__":
    main()
