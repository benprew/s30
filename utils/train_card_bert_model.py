import json
import os
import re

import numpy as np
import torch
import torch.nn as nn
from scipy.stats import spearmanr
from sklearn.model_selection import train_test_split
from torch.utils.data import DataLoader, Dataset
from transformers import AutoModel, AutoTokenizer

# ── Config ─────────────────────────────────────────────────────────────────────

MODEL_NAME = "google/bert_uncased_L-4_H-256_A-4"
OUTPUT_DIR = "./mtg-ranker"
MAX_LENGTH = 128
BATCH_SIZE = 8
EPOCHS = 20
LR = 2e-5
NUM_COLOR_GROUPS = 6  # W, U, B, R, G, colorless

SCRYFALL_PATH = "assets/card_info/scryfall_cards.json"
SCORES_PATH = "heuristic_scores.txt"

device = torch.device(
    "cuda"
    if torch.cuda.is_available()
    else "mps"
    if torch.backends.mps.is_available()
    else "cpu"
)
print(f"Using device: {device}")

COLOR_TO_IDX = {"W": 0, "U": 1, "B": 2, "R": 3, "G": 4}
COLORLESS_IDX = 5


# ── Data loading ───────────────────────────────────────────────────────────────


def parse_scores(path: str) -> dict[str, float]:
    """Parse heuristic_scores.txt into {card_name: score}."""
    scores = {}
    with open(path) as f:
        for line in f:
            m = re.match(
                r"\s+\d+\s+([\d.]+)\s+[\d.]+\s+[\d.]+\s+[\d.]+\s+[\d.]+\s+\S+\s+(.+)",
                line,
            )
            if m:
                scores[m.group(2).strip()] = float(m.group(1))
    return scores


def card_color_groups(card: dict) -> list[int]:
    """Return color group indices this card belongs to.

    Multicolor cards belong to each of their colors. Colorless cards get their
    own group. This means a multicolor card's score is informed by the context
    of each color pool it participates in.
    """
    ci = card.get("ColorIdentity") or []
    if not ci:
        return [COLORLESS_IDX]
    return [COLOR_TO_IDX[c] for c in ci if c in COLOR_TO_IDX]


def card_to_text(card: dict) -> str:
    parts = [
        f"Name: {card['CardName']}",
        f"Cost: {card['ManaCost'] or 'None'}",
        f"Type: {card['TypeLine']}",
    ]
    if card.get("Text"):
        parts.append(f"Text: {card['Text']}")
    if card.get("Power") is not None:
        parts.append(f"P/T: {card['Power']}/{card['Toughness']}")
    if card.get("Keywords"):
        parts.append(f"Keywords: {', '.join(card['Keywords'])}")
    return " | ".join(parts)


def load_data(scryfall_path: str, scores_path: str) -> tuple[list[dict], list[dict]]:
    """Load cards and return (scored_records, all_cards_by_color_group)."""
    scores = parse_scores(scores_path)
    with open(scryfall_path) as f:
        all_cards = json.load(f)

    # Build color groups: each group contains all cards of that color
    color_groups: dict[int, list[dict]] = {i: [] for i in range(NUM_COLOR_GROUPS)}
    for card in all_cards:
        for gid in card_color_groups(card):
            color_groups[gid].append(card)

    records = []
    for card in all_cards:
        name = card["CardName"]
        if name not in scores:
            continue
        records.append(
            {
                "text": card_to_text(card),
                "score": scores[name] / 100.0,
                "name": name,
                "color_groups": card_color_groups(card),
                "card": card,
            }
        )

    print(f"Loaded {len(records)} cards with scores")
    for gid in range(NUM_COLOR_GROUPS):
        label = list(COLOR_TO_IDX.keys())[gid] if gid < 5 else "C"
        print(f"  Color group {label}: {len(color_groups[gid])} cards")

    return records, color_groups


# ── Color pool embedding cache ────────────────────────────────────────────────


def build_color_pool_cache(
    color_groups: dict[int, list[dict]],
    tokenizer,
    encoder,
    max_length: int,
) -> dict[int, torch.Tensor]:
    """Precompute mean CLS embedding per color group.

    Each color pool embedding captures the "environment" a card competes in.
    Artifacts are strong partly because the colorless pool includes Moxen and
    Sol Ring; red burn is strong partly because of the red card pool's density
    of efficient damage.
    """
    encoder.eval()
    cache = {}

    with torch.no_grad():
        for gid, cards in color_groups.items():
            if not cards:
                continue
            texts = [card_to_text(c) for c in cards]
            all_embs = []
            chunk = 8
            for start in range(0, len(texts), chunk):
                enc = tokenizer(
                    texts[start : start + chunk],
                    truncation=True,
                    padding="max_length",
                    max_length=max_length,
                    return_tensors="pt",
                )
                out = encoder(
                    input_ids=enc["input_ids"].to(device),
                    attention_mask=enc["attention_mask"].to(device),
                )
                all_embs.append(out.last_hidden_state[:, 0, :].cpu())

            label = list(COLOR_TO_IDX.keys())[gid] if gid < 5 else "C"
            print(f"  Cached color group {label} ({len(cards)} cards)")
            cache[gid] = torch.cat(all_embs, dim=0).mean(dim=0)

    return cache


# ── Dataset ────────────────────────────────────────────────────────────────────


class MTGDataset(Dataset):
    def __init__(
        self,
        records: list[dict],
        tokenizer,
        color_pool_cache: dict[int, torch.Tensor],
        max_length: int,
    ):
        self.records = records
        self.tokenizer = tokenizer
        self.cache = color_pool_cache
        self.max_length = max_length
        self.hidden_size = next(iter(color_pool_cache.values())).shape[0]

    def __len__(self):
        return len(self.records)

    def __getitem__(self, idx):
        rec = self.records[idx]
        enc = self.tokenizer(
            rec["text"],
            truncation=True,
            padding="max_length",
            max_length=self.max_length,
            return_tensors="pt",
        )
        # Average the color pool embeddings for this card's color groups
        pool_embs = [self.cache[gid] for gid in rec["color_groups"]]
        pool_emb = torch.stack(pool_embs).mean(dim=0)

        return {
            "input_ids": enc["input_ids"].squeeze(0),
            "attention_mask": enc["attention_mask"].squeeze(0),
            "pool_emb": pool_emb,
            "label": torch.tensor(rec["score"], dtype=torch.float),
        }


# ── Model ──────────────────────────────────────────────────────────────────────


class CardRanker(nn.Module):
    """Scores a card in the context of its color pool.

    The card's CLS embedding captures what the card does. The color pool
    embedding captures what other cards are available in the same colors.
    The regression head learns how the card's abilities interact with the
    format's card pool to determine power level.
    """

    def __init__(self, model_name: str):
        super().__init__()
        self.encoder = AutoModel.from_pretrained(model_name)
        hidden = self.encoder.config.hidden_size

        # card_emb (hidden) + pool_emb (hidden) → score
        self.head = nn.Sequential(
            nn.Linear(hidden * 2, 256),
            nn.GELU(),
            nn.Dropout(0.1),
            nn.Linear(256, 64),
            nn.GELU(),
            nn.Dropout(0.1),
            nn.Linear(64, 1),
            nn.Sigmoid(),
        )

    def forward(self, input_ids, attention_mask, pool_emb):
        out = self.encoder(input_ids=input_ids, attention_mask=attention_mask)
        card_emb = out.last_hidden_state[:, 0, :]
        combined = torch.cat([card_emb, pool_emb], dim=-1)
        return self.head(combined).squeeze(-1)


# ── Training ───────────────────────────────────────────────────────────────────


def train():
    os.environ.setdefault("PYTORCH_ENABLE_MPS_FALLBACK", "1")

    records, color_groups = load_data(SCRYFALL_PATH, SCORES_PATH)
    train_records, eval_records = train_test_split(
        records, test_size=0.15, random_state=42
    )

    tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)

    model = CardRanker(MODEL_NAME).to(device)

    print("Precomputing color pool embeddings...")
    color_pool_cache = build_color_pool_cache(
        color_groups, tokenizer, model.encoder, MAX_LENGTH
    )

    train_ds = MTGDataset(train_records, tokenizer, color_pool_cache, MAX_LENGTH)
    eval_ds = MTGDataset(eval_records, tokenizer, color_pool_cache, MAX_LENGTH)

    train_loader = DataLoader(train_ds, batch_size=BATCH_SIZE, shuffle=True)
    eval_loader = DataLoader(eval_ds, batch_size=BATCH_SIZE * 2)

    optimizer = torch.optim.AdamW(model.parameters(), lr=LR, weight_decay=0.01)
    loss_fn = nn.HuberLoss()

    total_steps = len(train_loader) * EPOCHS
    scheduler = torch.optim.lr_scheduler.OneCycleLR(
        optimizer, max_lr=LR, total_steps=total_steps
    )

    best_spearman = -1.0
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    for epoch in range(1, EPOCHS + 1):
        model.train()
        train_losses = []
        for batch in train_loader:
            input_ids = batch["input_ids"].to(device)
            attention_mask = batch["attention_mask"].to(device)
            pool_emb = batch["pool_emb"].to(device)
            labels = batch["label"].to(device)

            preds = model(input_ids, attention_mask, pool_emb)
            loss = loss_fn(preds, labels)

            optimizer.zero_grad()
            loss.backward()
            torch.nn.utils.clip_grad_norm_(model.parameters(), 1.0)
            optimizer.step()
            scheduler.step()

            train_losses.append(loss.item())

        model.eval()
        all_preds, all_labels = [], []
        with torch.no_grad():
            for batch in eval_loader:
                input_ids = batch["input_ids"].to(device)
                attention_mask = batch["attention_mask"].to(device)
                pool_emb = batch["pool_emb"].to(device)
                preds = model(input_ids, attention_mask, pool_emb)
                all_preds.extend(preds.cpu().numpy())
                all_labels.extend(batch["label"].numpy())

        all_preds = np.array(all_preds)
        all_labels = np.array(all_labels)
        spearman = spearmanr(all_preds, all_labels).correlation
        mae = np.mean(np.abs(all_preds - all_labels)) * 100

        print(
            f"Epoch {epoch:>2}/{EPOCHS} | "
            f"loss={np.mean(train_losses):.4f} | "
            f"spearman={spearman:.4f} | "
            f"mae={mae:.2f}"
        )

        if spearman > best_spearman:
            best_spearman = spearman
            torch.save(model.state_dict(), f"{OUTPUT_DIR}/best_model.pt")
            tokenizer.save_pretrained(OUTPUT_DIR)
            print(f"  -> saved (best spearman={best_spearman:.4f})")

    print(f"\nTraining complete. Best spearman: {best_spearman:.4f}")


# ── Inference ──────────────────────────────────────────────────────────────────


def load_model():
    tokenizer = AutoTokenizer.from_pretrained(OUTPUT_DIR)
    model = CardRanker(MODEL_NAME).to(device)
    model.load_state_dict(
        torch.load(f"{OUTPUT_DIR}/best_model.pt", map_location=device)
    )
    model.eval()
    return model, tokenizer


def predict_card(
    card: dict,
    model,
    tokenizer,
    color_pool_cache: dict[int, torch.Tensor],
) -> float:
    enc = tokenizer(
        card_to_text(card),
        truncation=True,
        padding="max_length",
        max_length=MAX_LENGTH,
        return_tensors="pt",
    )
    groups = card_color_groups(card)
    pool_emb = torch.stack([color_pool_cache[g] for g in groups]).mean(dim=0)
    pool_emb = pool_emb.unsqueeze(0).to(device)

    with torch.no_grad():
        score = model(
            input_ids=enc["input_ids"].to(device),
            attention_mask=enc["attention_mask"].to(device),
            pool_emb=pool_emb,
        )
    return round(score.item() * 100, 2)


def rank_all(
    scryfall_path: str,
    model,
    tokenizer,
    color_pool_cache: dict[int, torch.Tensor],
) -> list[tuple[str, float]]:
    with open(scryfall_path) as f:
        cards = json.load(f)
    scored = [
        (c["CardName"], predict_card(c, model, tokenizer, color_pool_cache))
        for c in cards
    ]
    return sorted(scored, key=lambda x: x[1], reverse=True)


# ── Entry ──────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Train/predict MTG card power scores")
    parser.add_argument("--mode", choices=["train", "predict"], default="train")
    parser.add_argument("--scryfall", default=SCRYFALL_PATH)
    parser.add_argument("--scores", default=SCORES_PATH)
    args = parser.parse_args()

    SCRYFALL_PATH = args.scryfall
    SCORES_PATH = args.scores

    if args.mode == "train":
        train()

    elif args.mode == "predict":
        model, tokenizer = load_model()

        # Build color pool cache for inference
        with open(SCRYFALL_PATH) as f:
            all_cards = json.load(f)
        color_groups: dict[int, list[dict]] = {i: [] for i in range(NUM_COLOR_GROUPS)}
        for card in all_cards:
            for gid in card_color_groups(card):
                color_groups[gid].append(card)

        print("Computing color pool embeddings...")
        color_pool_cache = build_color_pool_cache(
            color_groups, tokenizer, model.encoder, MAX_LENGTH
        )

        ranked = rank_all(SCRYFALL_PATH, model, tokenizer, color_pool_cache)

        true_scores = parse_scores(SCORES_PATH)

        print(f"\n{'Rank':>4}  {'Score':>5}  {'True':>5}  Card")
        print(f"{'----':>4}  {'-----':>5}  {'-----':>5}  ----")
        for i, (name, score) in enumerate(ranked, 1):
            true = true_scores.get(name, -1)
            true_str = f"{true:5.1f}" if true >= 0 else "    -"
            print(f"{i:>4}  {score:>5.1f}  {true_str}  {name}")
