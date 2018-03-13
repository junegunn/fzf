// Various ways to generate single random colors

package colorful

import (
    "math/rand"
)

// Creates a random dark, "warm" color through a restricted HSV space.
func FastWarmColor() Color {
    return Hsv(
        rand.Float64() * 360.0,
        0.5 + rand.Float64()*0.3,
        0.3 + rand.Float64()*0.3)
}

// Creates a random dark, "warm" color through restricted HCL space.
// This is slower than FastWarmColor but will likely give you colors which have
// the same "warmness" if you run it many times.
func WarmColor() (c Color) {
    for c = randomWarm() ; !c.IsValid() ; c = randomWarm() {}
    return
}

func randomWarm() Color {
    return Hcl(
        rand.Float64() * 360.0,
        0.1 + rand.Float64()*0.3,
        0.2 + rand.Float64()*0.3)
}

// Creates a random bright, "pimpy" color through a restricted HSV space.
func FastHappyColor() Color {
    return Hsv(
        rand.Float64() * 360.0,
        0.7 + rand.Float64()*0.3,
        0.6 + rand.Float64()*0.3)
}

// Creates a random bright, "pimpy" color through restricted HCL space.
// This is slower than FastHappyColor but will likely give you colors which
// have the same "brightness" if you run it many times.
func HappyColor() (c Color) {
    for c = randomPimp() ; !c.IsValid() ; c = randomPimp() {}
    return
}

func randomPimp() Color {
    return Hcl(
        rand.Float64() * 360.0,
        0.5 + rand.Float64()*0.3,
        0.5 + rand.Float64()*0.3)
}

