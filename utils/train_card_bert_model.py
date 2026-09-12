import argparse
import json
import os
import re
import tomllib
from pathlib import Path
from typing import Any

import numpy as np
import torch
import torch.nn as nn
from scipy.stats import kendalltau, spearmanr
from sklearn.model_selection import train_test_split
from torch.utils.data import DataLoader, Dataset
from transformers import AutoModel, AutoTokenizer

try:
    from utils.train_card_classifier import (
        TIER_ORDER,
        TIER_SCORES,
        extract_parsed_mechanics,
        parse_mana_cost,
        score_card,
    )
except ImportError:
    from train_card_classifier import (  # type: ignore[no-redef]
        TIER_ORDER,
        TIER_SCORES,
        extract_parsed_mechanics,
        parse_mana_cost,
        score_card,
    )

ASSETS = Path(__file__).resolve().parent.parent / "assets"
DEFAULT_TIERS_PATH = ASSETS / "configs" / "card_tiers.toml"
DEFAULT_SCRYFALL_PATH = ASSETS / "card_info" / "scryfall_cards.json"
DEFAULT_PARSED_PATH = ASSETS / "card_info" / "parsed_cards.json"
DEFAULT_OUTPUT_DIR = "./mtg-ranker"

DEFAULT_MODEL_NAME = "distilbert-base-uncased"
MAX_LENGTH = 128
BATCH_SIZE = 16
EPOCHS = 25
LR = 2e-5
NUM_COLOR_GROUPS = 6

COLOR_TO_IDX = {"W": 0, "U": 1, "B": 2, "R": 3, "G": 4}
COLORLESS_IDX = 5

device = torch.device(
    "cuda"
    if torch.cuda.is_available()
    else "mps"
    if torch.backends.mps.is_available()
    else "cpu"
)


def load_card_tiers(path: Path | str) -> dict[str, str]:
    with open(path, "rb") as f:
        tiers = tomllib.load(f)
    name_to_tier: dict[str, str] = {}
    for tier_name in TIER_ORDER:
        for card_name in tiers.get(tier_name, []):
            name_to_tier[card_name] = tier_name
    return name_to_tier


def score_to_tier_name(score: float) -> str:
    return min(TIER_ORDER, key=lambda t: abs(TIER_SCORES[t] - score))


def tier_index(tier_name: str) -> int:
    try:
        return TIER_ORDER.index(tier_name)
    except ValueError:
        return len(TIER_ORDER) - 1


def card_color_groups(card: dict) -> list[int]:
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


def extract_card_features(
    card: dict,
    parsed_abilities: list[dict] | None = None,
) -> np.ndarray:
    mana = parse_mana_cost(card.get("ManaCost") or "")
    cmc = float(mana["cmc"])
    generic = float(mana["generic"])
    colored = float(mana["colored"])
    num_colors = float(mana["num_colors"])
    has_x = float(mana["has_x"])

    mana_str = card.get("ManaCost") or ""
    pip_w = float("{W}" in mana_str)
    pip_u = float("{U}" in mana_str)
    pip_b = float("{B}" in mana_str)
    pip_r = float("{R}" in mana_str)
    pip_g = float("{G}" in mana_str)

    tl = (card.get("TypeLine") or "").lower()
    is_creature = float("creature" in tl or "summon" in tl)
    is_land = float("land" in tl)
    is_instant = float("instant" in tl)
    is_sorcery = float("sorcery" in tl)
    is_enchantment = float("enchantment" in tl)
    is_artifact = float("artifact" in tl)

    p_str = card.get("Power")
    t_str = card.get("Toughness")
    p = float(p_str) if p_str and p_str.isdigit() else 0.0
    t = float(t_str) if t_str and t_str.isdigit() else 0.0

    pm = extract_parsed_mechanics(parsed_abilities) if parsed_abilities else None
    if pm and pm["etb_counters"]:
        p += pm["etb_counters"] * pm["counter_power"]
        t += pm["etb_counters"] * pm["counter_toughness"]

    total_pt = p + t
    eff_cmc = max(cmc, 1.0)
    pt_ratio = total_pt / eff_cmc if is_creature else 0.0
    p_ratio = p / eff_cmc if is_creature else 0.0

    kws = set(card.get("Keywords") or [])
    text_lower = (card.get("Text") or "").lower()
    kw_flying = float("Flying" in kws or "flying" in text_lower)
    kw_first_strike = float("First strike" in kws or "first strike" in text_lower)
    kw_trample = float("Trample" in kws or "trample" in text_lower)
    kw_haste = float("Haste" in kws or "haste" in text_lower)
    kw_vigilance = float("Vigilance" in kws or "vigilance" in text_lower)
    kw_protection = float(
        "Protection" in kws
        or bool(re.search(r"protection from|hexproof|shroud", text_lower))
    )
    kw_defender = float("Defender" in kws or "defender" in text_lower or "wall" in tl)
    kw_regenerate = float(bool(re.search(r"regenerate", text_lower)))

    has_destroy = float(
        pm["has_destroy"] if pm else bool(re.search(r"destroy target", text_lower))
    )
    has_exile = float(
        pm["has_exile"] if pm else bool(re.search(r"exile target", text_lower))
    )
    has_board_wipe = float(
        pm["has_board_wipe"] if pm else bool(re.search(r"destroy all", text_lower))
    )
    has_counter = float(
        pm["has_counter"]
        if pm
        else bool(re.search(r"counter target spell", text_lower))
    )
    cards_drawn = float(
        pm["draw_amount"]
        if pm
        else (3.0 if "ancestral" in card["CardName"].lower() else 0.0)
    )
    has_tutor = float(
        pm["has_tutor"] if pm else bool(re.search(r"search.*library", text_lower))
    )
    has_discard = float(
        pm["has_discard"] if pm else bool(re.search(r"discard", text_lower))
    )
    has_mana_ability = float(
        pm["has_mana_ability"]
        if pm
        else bool(re.search(r"add \{|add .* mana", text_lower))
    )
    damage_amount = float(pm["damage_amount"] if pm else 0.0)
    extra_turn = float(bool(re.search(r"extra turn", text_lower)))
    prevents_death = float(
        bool(re.search(r"can't lose the game|life total.*less than", text_lower))
    )
    doesnt_untap = float(
        pm["doesnt_untap"] if pm else bool(re.search(r"doesn't untap", text_lower))
    )
    vintage_restricted = float(card.get("VintageRestricted", False))

    heuristic = score_card(card, parsed_abilities)
    h_impact = heuristic["impact"] / 10.0
    h_efficiency = heuristic["efficiency"] / 10.0
    h_reliability = heuristic["reliability"] / 10.0
    h_quadrant = heuristic["quadrant_bonus"] / 4.0
    h_score = heuristic["score"] / 100.0

    features = [
        cmc,
        generic,
        colored,
        num_colors,
        has_x,
        pip_w,
        pip_u,
        pip_b,
        pip_r,
        pip_g,
        is_creature,
        is_land,
        is_instant,
        is_sorcery,
        is_enchantment,
        is_artifact,
        p,
        t,
        total_pt,
        pt_ratio,
        p_ratio,
        kw_flying,
        kw_first_strike,
        kw_trample,
        kw_haste,
        kw_vigilance,
        kw_protection,
        kw_defender,
        kw_regenerate,
        has_destroy,
        has_exile,
        has_board_wipe,
        has_counter,
        cards_drawn,
        has_tutor,
        has_discard,
        has_mana_ability,
        damage_amount,
        extra_turn,
        prevents_death,
        doesnt_untap,
        vintage_restricted,
        h_impact,
        h_efficiency,
        h_reliability,
        h_quadrant,
        h_score,
    ]
    return np.array(features, dtype=np.float32)


def load_dataset_records(
    scryfall_path: Path | str,
    tiers_path: Path | str,
    parsed_path: Path | str,
) -> tuple[list[dict], list[dict], dict[int, list[dict]]]:
    with open(scryfall_path) as f:
        all_cards = json.load(f)

    parsed_map: dict[str, list[dict]] = {}
    if Path(parsed_path).exists():
        with open(parsed_path) as f:
            parsed_data = json.load(f)
            parsed_map = {c["card_name"]: c["abilities"] for c in parsed_data}

    name_to_tier = load_card_tiers(tiers_path)

    color_groups: dict[int, list[dict]] = {i: [] for i in range(NUM_COLOR_GROUPS)}
    for card in all_cards:
        for gid in card_color_groups(card):
            color_groups[gid].append(card)

    labeled_records: list[dict] = []
    all_records: list[dict] = []
    seen: set[str] = set()

    for card in all_cards:
        name = card["CardName"]
        if name in seen:
            continue
        if "Basic Land" in card.get("TypeLine", ""):
            continue
        seen.add(name)

        parsed_ab = parsed_map.get(name)
        feat_vec = extract_card_features(card, parsed_ab)
        tier_label = name_to_tier.get(name, "")
        score_val = TIER_SCORES.get(tier_label, -1.0)

        record = {
            "name": name,
            "card": card,
            "text": card_to_text(card),
            "features": feat_vec,
            "color_groups": card_color_groups(card),
            "tier_label": tier_label,
            "score": score_val / 100.0 if score_val >= 0 else None,
        }
        all_records.append(record)
        if tier_label and score_val >= 0:
            labeled_records.append(record)

    print(f"Loaded {len(all_records)} unique non-basic cards.")
    fn = Path(tiers_path).name
    print(f"  -> {len(labeled_records)} cards matched ground truth in {fn}")
    for gid in range(NUM_COLOR_GROUPS):
        label = list(COLOR_TO_IDX.keys())[gid] if gid < 5 else "Colorless"
        print(f"  Color group {label}: {len(color_groups[gid])} cards")

    return labeled_records, all_records, color_groups


def build_color_pool_cache(
    color_groups: dict[int, list[dict]],
    tokenizer: Any,
    encoder: nn.Module,
    max_length: int,
) -> dict[int, torch.Tensor]:
    encoder.eval()
    cache = {}
    with torch.no_grad():
        for gid, cards in color_groups.items():
            if not cards:
                continue
            texts = [card_to_text(c) for c in cards]
            all_embs = []
            chunk = 16
            for start in range(0, len(texts), chunk):
                enc = tokenizer(
                    texts[start : start + chunk],
                    truncation=True,
                    padding="max_length",
                    max_length=max_length,
                    return_tensors="pt",
                )
                input_ids = enc["input_ids"].to(device)
                attention_mask = enc["attention_mask"].to(device)

                out = encoder(input_ids=input_ids, attention_mask=attention_mask)
                if hasattr(out, "last_hidden_state"):
                    emb = out.last_hidden_state[:, 0, :].cpu()
                else:
                    emb = out[0][:, 0, :].cpu()
                all_embs.append(emb)

            label = list(COLOR_TO_IDX.keys())[gid] if gid < 5 else "Colorless"
            print(f"  Cached color pool {label} ({len(cards)} cards)")
            cache[gid] = torch.cat(all_embs, dim=0).mean(dim=0)
    return cache


class MTGDataset(Dataset):
    def __init__(
        self,
        records: list[dict],
        tokenizer: Any,
        color_pool_cache: dict[int, torch.Tensor],
        max_length: int,
    ):
        self.records = records
        self.tokenizer = tokenizer
        self.cache = color_pool_cache
        self.max_length = max_length

    def __len__(self) -> int:
        return len(self.records)

    def __getitem__(self, index: int) -> dict[str, torch.Tensor]:
        rec = self.records[index]
        enc = self.tokenizer(
            rec["text"],
            truncation=True,
            padding="max_length",
            max_length=self.max_length,
            return_tensors="pt",
        )
        pool_embs = [self.cache[gid] for gid in rec["color_groups"]]
        pool_emb = torch.stack(pool_embs).mean(dim=0)
        label = rec["score"] if rec["score"] is not None else 0.0

        return {
            "input_ids": enc["input_ids"].squeeze(0),
            "attention_mask": enc["attention_mask"].squeeze(0),
            "pool_emb": pool_emb,
            "features": torch.tensor(rec["features"], dtype=torch.float32),
            "label": torch.tensor(label, dtype=torch.float32),
        }


class CardRanker(nn.Module):
    def __init__(self, model_name: str, num_features: int):
        super().__init__()
        self.encoder = AutoModel.from_pretrained(model_name)
        hidden = self.encoder.config.hidden_size

        self.tab_proj = nn.Sequential(
            nn.Linear(num_features, 64),
            nn.LayerNorm(64),
            nn.GELU(),
            nn.Dropout(0.1),
        )

        fusion_dim = hidden * 2 + 64
        self.head = nn.Sequential(
            nn.Linear(fusion_dim, 256),
            nn.LayerNorm(256),
            nn.GELU(),
            nn.Dropout(0.15),
            nn.Linear(256, 64),
            nn.GELU(),
            nn.Dropout(0.1),
            nn.Linear(64, 1),
            nn.Sigmoid(),
        )

    def forward(
        self,
        input_ids: torch.Tensor,
        attention_mask: torch.Tensor,
        pool_emb: torch.Tensor,
        features: torch.Tensor,
    ) -> torch.Tensor:
        out = self.encoder(input_ids=input_ids, attention_mask=attention_mask)
        if hasattr(out, "last_hidden_state"):
            card_emb = out.last_hidden_state[:, 0, :]
        else:
            card_emb = out[0][:, 0, :]

        tab_emb = self.tab_proj(features)
        combined = torch.cat([card_emb, pool_emb, tab_emb], dim=-1)
        return self.head(combined).squeeze(-1)


def pairwise_ranking_loss(
    preds: torch.Tensor, labels: torch.Tensor, margin: float = 0.05
) -> torch.Tensor:
    n = preds.size(0)
    if n < 2:
        return torch.tensor(0.0, device=preds.device)

    diff_labels = labels.unsqueeze(0) - labels.unsqueeze(1)
    diff_preds = preds.unsqueeze(0) - preds.unsqueeze(1)

    mask = torch.abs(diff_labels) > 0.04
    if not mask.any():
        return torch.tensor(0.0, device=preds.device)

    target_sign = torch.sign(diff_labels[mask])
    pred_delta = diff_preds[mask]

    loss = torch.relu(margin - target_sign * pred_delta)
    return loss.mean()


def evaluate_model(
    model: nn.Module,
    loader: DataLoader,
    records: list[dict],
) -> dict[str, float]:
    model.eval()
    all_preds: list[float] = []
    all_labels: list[float] = []

    with torch.no_grad():
        for batch in loader:
            input_ids = batch["input_ids"].to(device)
            attention_mask = batch["attention_mask"].to(device)
            pool_emb = batch["pool_emb"].to(device)
            features = batch["features"].to(device)
            preds = model(input_ids, attention_mask, pool_emb, features)

            all_preds.extend(preds.cpu().numpy().tolist())
            all_labels.extend(batch["label"].cpu().numpy().tolist())

    preds_np = np.array(all_preds) * 100.0
    labels_np = np.array(all_labels) * 100.0

    rho, _ = spearmanr(preds_np, labels_np)
    tau, _ = kendalltau(preds_np, labels_np)
    mae = float(np.mean(np.abs(preds_np - labels_np)))

    pred_tiers = [score_to_tier_name(p) for p in preds_np]
    true_tiers = [r["tier_label"] for r in records]

    exact_matches = sum(p == t for p, t in zip(pred_tiers, true_tiers))
    exact_acc = exact_matches / len(records) * 100.0

    off_by_one = sum(
        abs(tier_index(p) - tier_index(t)) <= 1 for p, t in zip(pred_tiers, true_tiers)
    )
    off_by_one_acc = off_by_one / len(records) * 100.0

    return {
        "spearman": float(rho),
        "kendall": float(tau),
        "mae": mae,
        "exact_acc": exact_acc,
        "off_by_one_acc": off_by_one_acc,
    }


def train(
    model_name: str = DEFAULT_MODEL_NAME,
    epochs: int = EPOCHS,
    lr: float = LR,
    output_dir: str = DEFAULT_OUTPUT_DIR,
    scryfall_path: Path | str = DEFAULT_SCRYFALL_PATH,
    tiers_path: Path | str = DEFAULT_TIERS_PATH,
    parsed_path: Path | str = DEFAULT_PARSED_PATH,
) -> None:
    os.environ.setdefault("PYTORCH_ENABLE_MPS_FALLBACK", "1")

    labeled_records, _, color_groups = load_dataset_records(
        scryfall_path, tiers_path, parsed_path
    )
    if not labeled_records:
        raise ValueError(f"No labeled cards found matching {tiers_path}")

    train_records, eval_records = train_test_split(
        labeled_records, test_size=0.15, random_state=42, shuffle=True
    )
    print(
        f"Training on {len(train_records)} cards, "
        f"validating on {len(eval_records)} cards."
    )

    tokenizer = AutoTokenizer.from_pretrained(model_name)
    assert tokenizer is not None
    num_features = len(labeled_records[0]["features"])
    model = CardRanker(model_name, num_features=num_features).to(device)

    print("Precomputing color pool embeddings...")
    color_pool_cache = build_color_pool_cache(
        color_groups, tokenizer, model.encoder, MAX_LENGTH
    )

    train_ds = MTGDataset(train_records, tokenizer, color_pool_cache, MAX_LENGTH)
    eval_ds = MTGDataset(eval_records, tokenizer, color_pool_cache, MAX_LENGTH)

    train_loader = DataLoader(train_ds, batch_size=BATCH_SIZE, shuffle=True)
    eval_loader = DataLoader(eval_ds, batch_size=BATCH_SIZE * 2, shuffle=False)

    optimizer = torch.optim.AdamW(model.parameters(), lr=lr, weight_decay=0.01)
    reg_loss_fn = nn.HuberLoss(delta=0.1)

    total_steps = len(train_loader) * epochs
    scheduler = torch.optim.lr_scheduler.OneCycleLR(
        optimizer, max_lr=lr, total_steps=total_steps
    )

    best_spearman = -1.0
    os.makedirs(output_dir, exist_ok=True)

    print(f"\nStarting training on {device} for {epochs} epochs...")
    hdr = (
        f"{'Epoch':>5} | {'Loss':>7} | {'Spearman':>8} | {'Kendall':>7} | "
        f"{'MAE':>6} | {'Exact%':>6} | {'±1 Tier%':>8}"
    )
    print(hdr)
    sep = (
        f"{'-----':>5}-+-{'-------':>7}-+-{'--------':>8}-+-{'-------':>7}-+-"
        f"{'------':>6}-+-{'------':>6}-+-{'--------':>8}"
    )
    print(sep)

    for epoch in range(1, epochs + 1):
        model.train()
        train_losses: list[float] = []

        for batch in train_loader:
            input_ids = batch["input_ids"].to(device)
            attention_mask = batch["attention_mask"].to(device)
            pool_emb = batch["pool_emb"].to(device)
            features = batch["features"].to(device)
            labels = batch["label"].to(device)

            preds = model(input_ids, attention_mask, pool_emb, features)

            loss_reg = reg_loss_fn(preds, labels)
            loss_rank = pairwise_ranking_loss(preds, labels, margin=0.04)
            loss = loss_reg + 0.5 * loss_rank

            optimizer.zero_grad()
            loss.backward()
            torch.nn.utils.clip_grad_norm_(model.parameters(), 1.0)
            optimizer.step()
            scheduler.step()

            train_losses.append(loss.item())

        metrics = evaluate_model(model, eval_loader, eval_records)
        epoch_loss = float(np.mean(train_losses))

        print(
            f"{epoch:5d} | {epoch_loss:7.4f} | {metrics['spearman']:8.3f} | "
            f"{metrics['kendall']:7.3f} | {metrics['mae']:6.1f} | "
            f"{metrics['exact_acc']:5.1f}% | {metrics['off_by_one_acc']:7.1f}%",
            end="",
        )

        if metrics["spearman"] > best_spearman:
            best_spearman = metrics["spearman"]
            torch.save(
                {
                    "model_state_dict": model.state_dict(),
                    "model_name": model_name,
                    "num_features": num_features,
                },
                f"{output_dir}/best_model.pt",
            )
            tokenizer.save_pretrained(output_dir)
            print("  ★ (Saved best)")
        else:
            print()

    print(f"\nTraining finished. Best Validation Spearman: {best_spearman:.3f}")


def load_trained_model(output_dir: str = DEFAULT_OUTPUT_DIR) -> tuple[CardRanker, Any]:
    checkpoint = torch.load(f"{output_dir}/best_model.pt", map_location=device)
    model_name = checkpoint.get("model_name", DEFAULT_MODEL_NAME)
    num_features = checkpoint.get("num_features", 47)

    tokenizer = AutoTokenizer.from_pretrained(output_dir)
    assert tokenizer is not None
    model = CardRanker(model_name, num_features=num_features).to(device)
    model.load_state_dict(checkpoint["model_state_dict"])
    model.eval()
    return model, tokenizer


def predict_and_evaluate(
    output_dir: str = DEFAULT_OUTPUT_DIR,
    scryfall_path: Path | str = DEFAULT_SCRYFALL_PATH,
    tiers_path: Path | str = DEFAULT_TIERS_PATH,
    parsed_path: Path | str = DEFAULT_PARSED_PATH,
    output_file: str = "",
) -> None:
    model, tokenizer = load_trained_model(output_dir)
    labeled_records, all_records, color_groups = load_dataset_records(
        scryfall_path, tiers_path, parsed_path
    )

    print("Precomputing color pool embeddings for prediction...")
    color_pool_cache = build_color_pool_cache(
        color_groups, tokenizer, model.encoder, MAX_LENGTH
    )

    ds = MTGDataset(all_records, tokenizer, color_pool_cache, MAX_LENGTH)
    loader = DataLoader(ds, batch_size=BATCH_SIZE * 2, shuffle=False)

    all_preds: list[float] = []
    with torch.no_grad():
        for batch in loader:
            input_ids = batch["input_ids"].to(device)
            attention_mask = batch["attention_mask"].to(device)
            pool_emb = batch["pool_emb"].to(device)
            features = batch["features"].to(device)
            preds = model(input_ids, attention_mask, pool_emb, features)
            all_preds.extend((preds.cpu().numpy() * 100.0).tolist())

    for rec, score in zip(all_records, all_preds):
        rec["pred_score"] = round(score, 1)
        rec["pred_tier"] = score_to_tier_name(score)

    all_records.sort(key=lambda r: r["pred_score"], reverse=True)

    labeled_subset = [r for r in all_records if r["tier_label"]]
    pred_scores = np.array([r["pred_score"] for r in labeled_subset])
    true_scores = np.array([TIER_SCORES[r["tier_label"]] for r in labeled_subset])

    rho, _ = spearmanr(pred_scores, true_scores)
    tau, _ = kendalltau(pred_scores, true_scores)
    mae = np.mean(np.abs(pred_scores - true_scores))
    exact_acc = (
        sum(r["pred_tier"] == r["tier_label"] for r in labeled_subset)
        / len(labeled_subset)
        * 100.0
    )
    off_by_one = (
        sum(
            abs(tier_index(r["pred_tier"]) - tier_index(r["tier_label"])) <= 1
            for r in labeled_subset
        )
        / len(labeled_subset)
        * 100.0
    )

    print("\n" + "═" * 78)
    msg = (
        f"MODEL PERFORMANCE AGAINST GROUND TRUTH ({len(labeled_subset)} labeled cards)"
    )
    print(msg)
    print("═" * 78)
    print(f"  Spearman rank correlation (rho): {rho:.3f}")
    print(f"  Kendall tau rank correlation:     {tau:.3f}")
    print(f"  Mean Absolute Error (MAE):       {mae:.1f} points")
    print(f"  Exact Tier Match:                {exact_acc:.1f}%")
    print(f"  Within ±1 Tier Match:            {off_by_one:.1f}%")

    diffs = [
        (
            r["name"],
            r["pred_score"],
            TIER_SCORES[r["tier_label"]],
            r["pred_score"] - TIER_SCORES[r["tier_label"]],
            r["pred_tier"],
            r["tier_label"],
        )
        for r in labeled_subset
    ]
    diffs.sort(key=lambda x: abs(x[3]), reverse=True)

    print("\nTop 15 Discrepancies vs card_tiers.toml:")
    hdr2 = (
        f"  {'Card Name':30s} {'Pred':>5s} {'True':>5s} {'Diff':>6s}  "
        f"{'Predicted Tier':<25s} {'Ground Truth Tier'}"
    )
    print(hdr2)
    print(f"  {'-' * 30} {'-' * 5} {'-' * 5} {'-' * 6}  {'-' * 25} {'-' * 20}")
    for name, pred, true, diff, pred_t, true_t in diffs[:15]:
        print(
            f"  {name:30s} {pred:5.1f} {true:5.1f} {diff:+6.1f}  {pred_t:<25s} {true_t}"
        )

    unranked = [r for r in all_records if not r["tier_label"]]
    if unranked:
        print(f"\nUnranked / New Cards Suggested Tiers ({len(unranked)} cards):")
        print(f"  {'Card Name':30s} {'Pred Score':>10s}  {'Suggested Tier'}")
        print(f"  {'-' * 30} {'-' * 10}  {'-' * 25}")
        for r in unranked[:20]:
            print(f"  {r['name']:30s} {r['pred_score']:10.1f}  {r['pred_tier']}")

    if output_file:
        with open(output_file, "w") as f:
            f.write("rank\tcard_name\tscore\tpredicted_tier\ttrue_tier\ttype_line\n")
            for i, r in enumerate(all_records, 1):
                f.write(
                    f"{i}\t{r['name']}\t{r['pred_score']:.1f}\t{r['pred_tier']}\t"
                    f"{r['tier_label']}\t{r['card'].get('TypeLine', '')}\n"
                )
        print(f"\nSaved full predictions for {len(all_records)} cards to {output_file}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Hybrid BERT Card Tier Ranker")
    parser.add_argument("--mode", choices=["train", "predict"], default="train")
    parser.add_argument(
        "--model", default=DEFAULT_MODEL_NAME, help="HuggingFace model backbone"
    )
    parser.add_argument(
        "--epochs", type=int, default=EPOCHS, help="Number of training epochs"
    )
    parser.add_argument("--lr", type=float, default=LR, help="Learning rate")
    parser.add_argument("--output-dir", default=DEFAULT_OUTPUT_DIR)
    parser.add_argument("--scryfall", default=DEFAULT_SCRYFALL_PATH)
    parser.add_argument("--tiers", default=DEFAULT_TIERS_PATH)
    parser.add_argument("--parsed", default=DEFAULT_PARSED_PATH)
    parser.add_argument("--output", default="", help="File to write prediction TSV to")
    args = parser.parse_args()

    if args.mode == "train":
        train(
            model_name=args.model,
            epochs=args.epochs,
            lr=args.lr,
            output_dir=args.output_dir,
            scryfall_path=args.scryfall,
            tiers_path=args.tiers,
            parsed_path=args.parsed,
        )
    elif args.mode == "predict":
        predict_and_evaluate(
            output_dir=args.output_dir,
            scryfall_path=args.scryfall,
            tiers_path=args.tiers,
            parsed_path=args.parsed,
            output_file=args.output,
        )
