#!/usr/bin/env python3

import cv2
import sys
from skimage.metrics import structural_similarity as ssim


def compare_images(img1, img2):
    # Convert to grayscale
    gray1 = cv2.cvtColor(img1, cv2.COLOR_BGR2GRAY)
    gray2 = cv2.cvtColor(img2, cv2.COLOR_BGR2GRAY)

    # Compute SSIM
    score, _ = ssim(gray1, gray2, full=True)
    return score


def find_closest_match(reference_path, candidate_paths):
    # Load reference image
    ref = cv2.imread(reference_path)
    if ref is None:
        raise ValueError(f"Could not load reference image: {reference_path}")

    best_score = -1
    best_match = None

    for candidate in candidate_paths:
        img = cv2.imread("assets/art/sprites/rogues_tmp/" + candidate)
        if img is None:
            raise ValueError(f"Skipping {candidate}, could not load.")

        # Resize candidate to reference size (important!)
        if img.shape != ref.shape:
            img = cv2.resize(img, (ref.shape[1], ref.shape[0]))

        score = compare_images(ref, img)
        print(f"{candidate}: SSIM = {score:.4f}")

        if score > best_score:
            best_score = score
            best_match = candidate

    return best_match, best_score


# Example usage
if __name__ == "__main__":
    reference = sys.argv[1]
    candidates = [
        "Chunk.pic.png",
        "Cutiepie.pic.png",
        "Desdemona.pic.png",
        "Face.pic.png",
        "Greenie.pic.png",
        "Lance.pic.png",
        "Lizzy.pic.png",
        "Lumpy.pic.png",
        "Medusa.pic.png",
        "Ophelia.pic.png",
        "Rogue01.pic.png",
        "Rogue02.pic.png",
        "Rogue03.pic.png",
        "Rogue04.pic.png",
        "Rogue05.pic.png",
        "Rogue06.pic.png",
        "Rogue07.pic.png",
        "Rogue08.pic.png",
        "Rogue09.pic.png",
        "Rogue10.pic.png",
        "Rogue11.pic.png",
        "Rogue12.pic.png",
        "Rogue13.pic.png",
        "Rogue14.pic.png",
        "Rogue15.pic.png",
        "Rogue16.pic.png",
        "Rogue17.pic.png",
        "Rogue19.pic.png",
        "Rogue20.pic.png",
        "Rogue21.pic.png",
        "Rogue22.pic.png",
        "Rogue23.pic.png",
        "Rogue24.pic.png",
        "Rogue25.pic.png",
        "Rogue26.pic.png",
        "Rogue27.pic.png",
        "Rogue28.pic.png",
        "Rogue29.pic.png",
        "Rogue30.pic.png",
        "Rogue32.pic.png",
        "Rogue33.pic.png",
        "Rogue34.pic.png",
        "Rogue35.pic.png",
        "Rogue37.pic.png",
        "Rogue38.pic.png",
        "Rogue39.pic.png",
        "Rogue40.pic.png",
        "Rogue41.pic.png",
        "Rogue42.pic.png",
        "Rogue43.pic.png",
        "Rogue45.pic.png",
        "Rogue46.pic.png",
        "Rogue47.pic.png",
        "Rogue48.pic.png",
        "Rogue49.pic.png",
        "Rogue50.pic.png",
        "Rogue51.pic.png",
        "Rogue52.pic.png",
        "Rogue53.pic.png",
        "Rogue54.pic.png",
        "Rogue55.pic.png",
        "Rogue56.pic.png",
        "Rogue57.pic.png",
        "Rogue58.pic.png",
        "Rogue60.pic.png",
        "Rogue61.pic.png",
        "Rogue63.pic.png",
        "Rogue64.pic.png",
        "Rogue65.pic.png",
        "Rogue66.pic.png",
        "Rogue67.pic.png",
        "Rogue68.pic.png",
        "Rogue69.pic.png",
        "Rogue70.pic.png",
        "Rogue71.pic.png",
        "Rogue72.pic.png",
        "Splinter.pic.png",
    ]

    match, score = find_closest_match(reference, candidates)
    print(f"\n{reference} = {match} | SSIM = {score:.4f}")
