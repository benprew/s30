# Alternate basic land art

Drop a `landArt` folder next to the game executable and your own art will be
used for basic lands.

```
landArt/
├── Swamp.png                  # loose files are fine
├── beta/
│   ├── Forest.full.jpg
│   └── Island.full.jpg
└── unlimited/
    ├── forest-2.png
    └── forest-3.png
```

- **Naming:** a filename counts if it *starts with* a basic land name —
  `Plains`, `Island`, `Swamp`, `Mountain`, or `Forest`. Case is ignored and
  anything may follow, so `Island.full.jpg`, `forest-2.png`, and
  `MOUNTAIN_beta.JPEG` all work. A file named `beta_forest.png` does **not**:
  the land name has to come first.
- **Formats:** `.jpg`, `.jpeg`, and `.png`.
- **Layout:** subfolders are searched, as deep as you like, so you can keep one
  folder per set or pile everything into one. Give a land several images and
  each land gets one at random.
- **Images should be whole card faces**, like Scryfall's `border_crop` renders
  or an MTG Forge `.full.jpg` picture pack — frame, name box and all, not just
  the illustration. Anything close to a 245x342 ratio looks right.

Art is picked per card and stays put for as long as you are looking at it: the
lands in a duel keep their art for that whole duel, and the deck editor picks
again next time you open it.

On startup the game reports what it found. If you see

```
Found a landArt folder but no usable images
```

the filenames do not start with a basic land name. Alternate art is a desktop
feature — the browser build has no folder to read.

[Back to the README](README.md)
