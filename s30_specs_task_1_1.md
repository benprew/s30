Task 1.1: Set up a Basic Noise-Based System

We’ll use Perlin or Simplex noise to generate the terrain. Here’s an outline of what needs to be done:

    Choose a Noise Library:
    Since you're using Go, we can use a package like github.com/aquilax/perc for Perlin noise or another library for Simplex noise. Which one would you prefer, or should I go with Perlin noise for simplicity?

    Generate a Noise Map:
    Create a 2D noise map based on the world’s size (1000x1000). This map will give us the height values that we can map to terrain types.

    Adjust Noise Scale and Octaves:
    Set up the noise parameters, including scale and octaves, to ensure smooth transitions between terrain types. We can tweak these parameters later for a more natural-looking world.

Would you like to proceed with Perlin noise, or do you have a different preference?

Types of terrain:
- plains
- water
- desert
- forest
- marsh 
- ice
