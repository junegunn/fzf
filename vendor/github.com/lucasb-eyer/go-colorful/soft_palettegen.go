// Largely inspired by the descriptions in http://lab.medialab.sciences-po.fr/iwanthue/
// but written from scratch.

package colorful

import (
    "fmt"
    "math"
    "math/rand"
)

// The algorithm works in L*a*b* color space and converts to RGB in the end.
// L* in [0..1], a* and b* in [-1..1]
type lab_t struct {
    L, A, B float64
}

type SoftPaletteSettings struct {
    // A function which can be used to restrict the allowed color-space.
    CheckColor func(l, a, b float64) bool

    // The higher, the better quality but the slower. Usually two figures.
    Iterations int

    // Use up to 160000 or 8000 samples of the L*a*b* space (and thus calls to CheckColor).
    // Set this to true only if your CheckColor shapes the Lab space weirdly.
    ManySamples bool
}

// Yeah, windows-stype Foo, FooEx, screw you golang...
// Uses K-means to cluster the color-space and return the means of the clusters
// as a new palette of distinctive colors. Falls back to K-medoid if the mean
// happens to fall outside of the color-space, which can only happen if you
// specify a CheckColor function.
func SoftPaletteEx(colorsCount int, settings SoftPaletteSettings) ([]Color, error) {

    // Checks whether it's a valid RGB and also fulfills the potentially provided constraint.
    check := func(col lab_t) bool {
        c := Lab(col.L, col.A, col.B)
        return c.IsValid() && (settings.CheckColor == nil || settings.CheckColor(col.L, col.A, col.B))
    }

    // Sample the color space. These will be the points k-means is run on.
    dl := 0.05
    dab := 0.1
    if settings.ManySamples {
        dl = 0.01
        dab = 0.05
    }

    samples := make([]lab_t, 0, int(1.0/dl * 2.0/dab * 2.0/dab))
    for l := 0.0; l <= 1.0; l += dl {
        for a := -1.0; a <= 1.0; a += dab {
            for b := -1.0; b <= 1.0; b += dab {
                if check(lab_t{l,a,b}) {
                    samples = append(samples, lab_t{l, a, b})
                }
            }
        }
    }

    // That would cause some infinite loops down there...
    if len(samples) < colorsCount {
        return nil, fmt.Errorf("palettegen: more colors requested (%v) than samples available (%v). Your requested color count may be wrong, you might want to use many samples or your constraint function makes the valid color space too small.", colorsCount, len(samples))
    } else if len(samples) == colorsCount {
        return labs2cols(samples), nil // Oops?
    }

    // We take the initial means out of the samples, so they are in fact medoids.
    // This helps us avoid infinite loops or arbitrary cutoffs with too restrictive constraints.
    means := make([]lab_t, colorsCount)
    for i := 0; i < colorsCount; i++ {
        for means[i] = samples[rand.Intn(len(samples))] ; in(means, i, means[i]) ; means[i] = samples[rand.Intn(len(samples))] {
        }
    }

    clusters := make([]int, len(samples))
    samples_used := make([]bool, len(samples))

    // The actual k-means/medoid iterations
    for i := 0; i < settings.Iterations; i++ {
        // Reassing the samples to clusters, i.e. to their closest mean.
        // By the way, also check if any sample is used as a medoid and if so, mark that.
        for isample, sample := range samples {
            samples_used[isample] = false
            mindist := math.Inf(+1)
            for imean, mean := range means {
                dist := lab_dist(sample, mean)
                if dist < mindist {
                    mindist = dist
                    clusters[isample] = imean
                }

                // Mark samples which are used as a medoid.
                if lab_eq(sample, mean) {
                    samples_used[isample] = true
                }
            }
        }

        // Compute new means according to the samples.
        for imean := range means {
            // The new mean is the average of all samples belonging to it..
            nsamples := 0
            newmean := lab_t{0.0, 0.0, 0.0}
            for isample, sample := range samples {
                if clusters[isample] == imean {
                    nsamples++
                    newmean.L += sample.L
                    newmean.A += sample.A
                    newmean.B += sample.B
                }
            }
            if nsamples > 0 {
                newmean.L /= float64(nsamples)
                newmean.A /= float64(nsamples)
                newmean.B /= float64(nsamples)
            } else {
                // That mean doesn't have any samples? Get a new mean from the sample list!
                var inewmean int
                for inewmean = rand.Intn(len(samples_used)); samples_used[inewmean]; inewmean = rand.Intn(len(samples_used)) {
                }
                newmean = samples[inewmean]
                samples_used[inewmean] = true
            }

            // But now we still need to check whether the new mean is an allowed color.
            if nsamples > 0 && check(newmean) {
                // It does, life's good (TM)
                means[imean] = newmean
            } else {
                // New mean isn't an allowed color or doesn't have any samples!
                // Switch to medoid mode and pick the closest (unused) sample.
                // This should always find something thanks to len(samples) >= colorsCount
                mindist := math.Inf(+1)
                for isample, sample := range samples {
                    if !samples_used[isample] {
                        dist := lab_dist(sample, newmean)
                        if dist < mindist {
                            mindist = dist
                            newmean = sample
                        }
                    }
                }
            }
        }
    }
    return labs2cols(means), nil
}

// A wrapper which uses common parameters.
func SoftPalette(colorsCount int) ([]Color, error) {
    return SoftPaletteEx(colorsCount, SoftPaletteSettings{nil, 50, false})
}

func in(haystack []lab_t, upto int, needle lab_t) bool {
    for i := 0 ; i < upto && i < len(haystack) ; i++ {
        if haystack[i] == needle {
            return true
        }
    }
    return false
}

const LAB_DELTA = 1e-6
func lab_eq(lab1, lab2 lab_t) bool {
    return math.Abs(lab1.L - lab2.L) < LAB_DELTA &&
           math.Abs(lab1.A - lab2.A) < LAB_DELTA &&
           math.Abs(lab1.B - lab2.B) < LAB_DELTA
}

// That's faster than using colorful's DistanceLab since we would have to
// convert back and forth for that. Here is no conversion.
func lab_dist(lab1, lab2 lab_t) float64 {
    return math.Sqrt(sq(lab1.L-lab2.L) + sq(lab1.A-lab2.A) + sq(lab1.B-lab2.B))
}

func labs2cols(labs []lab_t) (cols []Color) {
    cols = make([]Color, len(labs))
    for k, v := range labs {
        cols[k] = Lab(v.L, v.A, v.B)
    }
    return cols
}

