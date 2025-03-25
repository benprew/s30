# Character Sprite Sheet Loader Design

## Overview

This document outlines the design for a sprite loading system in Go using the Ebiten game engine, tailored to the specific sprite sheet naming and shadow file conventions.

## File Naming Convention

### Character Sprite Colors/Types
- `W_`: White
- `Bk_`: Black
- `Bu_`: Blue
- `G_`: Green
- `R_`: Red
- `Dg_`: Dragon
- `M_`: Multicolor

### Shadow File Naming Rules
- Shadow files are prefixed with 'S'
- Shadow file names correspond to their character sprite files:
  - `Sb_Lrd.spr.png` is the shadow for `Bk_Lrd.spr.png`
  - `Su_Lrd.spr.png` is the shadow for `Bu_Lrd.spr.png`
  - `Strl.spr.png` is the shadow for `Troll.spr.png`

## Design Considerations

1. **Flexible Shadow Matching**: 
   - Handles both color-prefixed shadows (`Sb_`, `Su_`) 
   - Supports single-name sprites like `Troll`
   - Gracefully handles missing shadows

2. **Color-Based Classification**: 
   - Clear constants for character colors
   - Easy filtering and selection of sprites

3. **Naming Convention Adherence**:
   - Directly maps to the provided file naming rules
   - Minimal assumptions about file structure

## Potential Future Enhancements

1. **Metadata Extraction**: 
   - Parse additional information from filenames
   - Support custom metadata for sprites

2. **Dynamic Color Mapping**:
   - Allow runtime addition of new color types
   - Support for more complex color categorizations

## Appendix
### Prompt

write a character sprite loader in go using ebiten. Sprite sheets are single png files that contain multiple images. These images have rows and columns. 

These sheets are loaded using go's embed system.

Character sprite sheets have 8 rows and 5 columns. each row represents a character direction and each row is a animation step.

Characters have 2 png files: a sheet of the character and a sheet of the characters shadow. The shadow sheets follow the same format as the character sheet.

These sprites are saved in the art/sprites/world/characters directory.

There are no classes, but there are types of characters, most correspond to a color, but some do no:
W_ : White
Bk_ : Black
Bu_ : Blue
G_ : Green
R_ : Red
Dg_ : Dragon
M_ : Multicolor

Files prefixed with 'S' are shadown files.

examples:
 Sb_Lrd.spr.png is the shadow file for Bk_Lrd.spr.png
 Su_Lrd.spr.png is the shadow file for Bu_Lrd.spr.png
 Strl.spr.png is the shadow file for Troll.spr.png

The character sheets are:

Bk_Amg.spr.png
Bk_Djn.spr.png
Bk_Fwz.spr.png
Bk_Kht.spr.png
Bk_Lrd.spr.png
Bk_Mwz.spr.png
Bk_Wg.spr.png
B_Sfr.spr.png
Bu_Amg.spr.png
Bu_Djn.spr.png
Bu_Fwz.spr.png
Bu_Lrd.spr.png
Bu_Mwz.spr.png
Bu_Wrm.spr.png
Dg_Bru.spr.png
Dg_Gwr.spr.png
Dg_Rbg.spr.png
Dg_Uwb.spr.png
Dg_Wug.spr.png
Ego_F.spr.png
Ego_M.spr.png
G_Amg.spr.png
G_Djn.spr.png
G_Fwz.spr.png
G_Kht.spr.png
G_Lrd.spr.png
G_Mwz.spr.png
G_Wrm.spr.png
M_Ape.spr.png
M_Cen2.spr.png
M_Cen.spr.png
M_Fng.spr.png
M_Fwz.spr.png
M_Kht.spr.png
M_Lrd.spr.png
M_Trl.spr.png
M_Tsk.spr.png
M_Wg.spr.png
R_Amg.spr.png
R_Djn.spr.png
R_Fwz.spr.png
R_Lrd.spr.png
R_Mwz.spr.png
R_Wrm.spr.png
Troll.spr.png
W_Amg.spr.png
W_Fwz.spr.png
W_Kht.spr.png
W_Lrd.spr.png
W_Mwz.spr.png
W_Wg.spr.png

And the shadow sprites are:

Sb_Amg.spr.png
Sb_Kht.spr.png
Sbk_Wg.spr.png
Sb_Lrd.spr.png
Sb_Sft.spr.png
S_Dg.spr.png
Sdjn.spr.png
Sego_F.spr.png
Sego_M.spr.png
Sfwz.spr.png
Sg_Amg.spr.png
Sg_Lrd.spr.png
Skht.spr.png
Sm_Ape.spr.png
Sm_Cen2.spr.png
Sm_Cen.spr.png
Sm_Fng.spr.png
Sm_Lrd.spr.png
Sm_Trl.spr.png
Sm_Tsk.spr.png
Sm_Wg.spr.png
Smwz.spr.png
Sr_Amg.spr.png
Sr_Fwz.spr.png
Sr_Lrd.spr.png
Sr_Wrm.spr.png
Strl.spr.png
Su_Amg.spr.png
Su_Lrd.spr.png
Sw_Amg.spr.png
Sw_Lrd.spr.png
Sw_Mwz.spr.png
Swrm.spr.png
Sw_Wg.spr.png

