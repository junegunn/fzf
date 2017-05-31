// The colorful package provides all kinds of functions for working with colors.
package colorful

import(
    "fmt"
    "math"
)

// A color is stored internally using sRGB (standard RGB) values in the range 0-1
type Color struct {
    R, G, B float64
}

// Implement the Go color.Color interface.
func (col Color) RGBA() (r, g, b, a uint32) {
    r = uint32(col.R*65535.0)
    g = uint32(col.G*65535.0)
    b = uint32(col.B*65535.0)
    a = 0xFFFF
    return
}

// Might come in handy sometimes to reduce boilerplate code.
func (col Color) RGB255() (r, g, b uint8) {
    r = uint8(col.R*255.0)
    g = uint8(col.G*255.0)
    b = uint8(col.B*255.0)
    return
}

// This is the tolerance used when comparing colors using AlmostEqualRgb.
const Delta = 1.0/255.0

// This is the default reference white point.
var D65 = [3]float64{0.95047, 1.00000, 1.08883}

// And another one.
var D50 = [3]float64{0.96422, 1.00000, 0.82521}

// Checks whether the color exists in RGB space, i.e. all values are in [0..1]
func (c Color) IsValid() bool {
    return 0.0 <= c.R && c.R <= 1.0 &&
           0.0 <= c.G && c.G <= 1.0 &&
           0.0 <= c.B && c.B <= 1.0
}

func clamp01(v float64) float64 {
    return math.Max(0.0, math.Min(v, 1.0))
}

// Returns Clamps the color into valid range, clamping each value to [0..1]
// If the color is valid already, this is a no-op.
func (c Color) Clamped() Color {
    return Color{clamp01(c.R), clamp01(c.G), clamp01(c.B)}
}

func sq(v float64) float64 {
    return v * v;
}

func cub(v float64) float64 {
    return v * v * v;
}

// DistanceRgb computes the distance between two colors in RGB space.
// This is not a good measure! Rather do it in Lab space.
func (c1 Color) DistanceRgb(c2 Color) float64 {
    return math.Sqrt(sq(c1.R-c2.R) + sq(c1.G-c2.G) + sq(c1.B-c2.B))
}

// Check for equality between colors within the tolerance Delta (1/255).
func (c1 Color) AlmostEqualRgb(c2 Color) bool {
    return math.Abs(c1.R - c2.R) +
           math.Abs(c1.G - c2.G) +
           math.Abs(c1.B - c2.B) < 3.0*Delta
}

// You don't really want to use this, do you? Go for BlendLab, BlendLuv or BlendHcl.
func (c1 Color) BlendRgb(c2 Color, t float64) Color {
    return Color{c1.R + t*(c2.R - c1.R),
                 c1.G + t*(c2.G - c1.G),
                 c1.B + t*(c2.B - c1.B)}
}

// Utility used by Hxx color-spaces for interpolating between two angles in [0,360].
func interp_angle(a0, a1, t float64) float64 {
    // Based on the answer here: http://stackoverflow.com/a/14498790/2366315
    // With potential proof that it works here: http://math.stackexchange.com/a/2144499
    delta := math.Mod(math.Mod(a1 - a0, 360.0) + 540, 360.0) - 180.0
    return math.Mod(a0 + t*delta + 360.0, 360.0)
}


/// HSV ///
///////////
// From http://en.wikipedia.org/wiki/HSL_and_HSV
// Note that h is in [0..360] and s,v in [0..1]

// Hsv returns the Hue [0..360], Saturation and Value [0..1] of the color.
func (col Color) Hsv() (h, s, v float64) {
    min := math.Min(math.Min(col.R, col.G), col.B)
    v    = math.Max(math.Max(col.R, col.G), col.B)
    C := v - min

    s = 0.0
    if v != 0.0 {
        s = C / v
    }

    h = 0.0  // We use 0 instead of undefined as in wp.
    if min != v {
        if v == col.R { h = math.Mod((col.G - col.B) / C, 6.0) }
        if v == col.G { h = (col.B - col.R) / C + 2.0 }
        if v == col.B { h = (col.R - col.G) / C + 4.0 }
        h *= 60.0
        if h < 0.0 { h += 360.0 }
    }
    return
}

// Hsv creates a new Color given a Hue in [0..360], a Saturation and a Value in [0..1]
func Hsv(H, S, V float64) Color {
    Hp := H/60.0
    C := V*S
    X := C*(1.0-math.Abs(math.Mod(Hp, 2.0)-1.0))

    m := V-C;
    r, g, b := 0.0, 0.0, 0.0

    switch {
    case 0.0 <= Hp && Hp < 1.0: r = C; g = X
    case 1.0 <= Hp && Hp < 2.0: r = X; g = C
    case 2.0 <= Hp && Hp < 3.0: g = C; b = X
    case 3.0 <= Hp && Hp < 4.0: g = X; b = C
    case 4.0 <= Hp && Hp < 5.0: r = X; b = C
    case 5.0 <= Hp && Hp < 6.0: r = C; b = X
    }

    return Color{m+r, m+g, m+b}
}

// You don't really want to use this, do you? Go for BlendLab, BlendLuv or BlendHcl.
func (c1 Color) BlendHsv(c2 Color, t float64) Color {
    h1, s1, v1 := c1.Hsv()
    h2, s2, v2 := c2.Hsv()

    // We know that h are both in [0..360]
    return Hsv(interp_angle(h1, h2, t), s1 + t*(s2 - s1), v1 + t*(v2 - v1))
}

/// HSL ///
///////////

// Hsl returns the Hue [0..360], Saturation [0..1], and Luminance (lightness) [0..1] of the color.
func (col Color) Hsl() (h, s, l float64) {
    min := math.Min(math.Min(col.R, col.G), col.B)
    max := math.Max(math.Max(col.R, col.G), col.B)

    l = (max + min) / 2

    if min == max {
        s = 0
        h = 0
    } else {
        if l < 0.5 {
            s = (max - min) / (max + min)
        } else {
            s = (max - min) / (2.0 - max - min)
        }

        if max == col.R {
            h = (col.G - col.B) / (max - min)
        } else if max == col.G {
            h = 2.0 + (col.B-col.R)/(max-min)
        } else {
            h = 4.0 + (col.R-col.G)/(max-min)
        }

        h *= 60

        if h < 0 {
            h += 360
        }
    }

    return
}

// Hsl creates a new Color given a Hue in [0..360], a Saturation [0..1], and a Luminance (lightness) in [0..1]
func Hsl(h, s, l float64) Color {
    if s == 0 {
        return Color{l, l, l}
    }

    var r, g, b float64
    var t1 float64
    var t2 float64
    var tr float64
    var tg float64
    var tb float64

    if l < 0.5 {
        t1 = l * (1.0 + s)
    } else {
        t1 = l + s - l*s
    }

    t2 = 2*l - t1
    h = h / 360
    tr = h + 1.0/3.0
    tg = h
    tb = h - 1.0/3.0

    if tr < 0 {
        tr += 1
    }
    if tr > 1 {
        tr -= 1
    }
    if tg < 0 {
        tg += 1
    }
    if tg > 1 {
        tg -= 1
    }
    if tb < 0 {
        tb += 1
    }
    if tb > 1 {
        tb -= 1
    }

    // Red
    if 6*tr < 1 {
        r = t2 + (t1-t2)*6*tr
    } else if 2*tr < 1 {
        r = t1
    } else if 3*tr < 2 {
        r = t2 + (t1-t2)*(2.0/3.0-tr)*6
    } else {
        r = t2
    }

    // Green
    if 6*tg < 1 {
        g = t2 + (t1-t2)*6*tg
    } else if 2*tg < 1 {
        g = t1
    } else if 3*tg < 2 {
        g = t2 + (t1-t2)*(2.0/3.0-tg)*6
    } else {
        g = t2
    }

    // Blue
    if 6*tb < 1 {
        b = t2 + (t1-t2)*6*tb
    } else if 2*tb < 1 {
        b = t1
    } else if 3*tb < 2 {
        b = t2 + (t1-t2)*(2.0/3.0-tb)*6
    } else {
        b = t2
    }

    return Color{r, g, b}
}

/// Hex ///
///////////

// Hex returns the hex "html" representation of the color, as in #ff0080.
func (col Color) Hex() string {
    // Add 0.5 for rounding
    return fmt.Sprintf("#%02x%02x%02x", uint8(col.R*255.0+0.5), uint8(col.G*255.0+0.5), uint8(col.B*255.0+0.5))
}

// Hex parses a "html" hex color-string, either in the 3 "#f0c" or 6 "#ff1034" digits form.
func Hex(scol string) (Color, error) {
    format := "#%02x%02x%02x"
    factor := 1.0/255.0
    if len(scol) == 4 {
        format = "#%1x%1x%1x"
        factor = 1.0/15.0
    }

    var r, g, b uint8
    n, err := fmt.Sscanf(scol, format, &r, &g, &b)
    if err != nil {
        return Color{}, err
    }
    if n != 3 {
        return Color{}, fmt.Errorf("color: %v is not a hex-color", scol)
    }

    return Color{float64(r)*factor, float64(g)*factor, float64(b)*factor}, nil
}

/// Linear ///
//////////////
// http://www.sjbrown.co.uk/2004/05/14/gamma-correct-rendering/
// http://www.brucelindbloom.com/Eqn_RGB_to_XYZ.html

func linearize(v float64) float64 {
    if v <= 0.04045 {
        return v / 12.92
    }
    return math.Pow((v + 0.055)/1.055, 2.4)
}

// LinearRgb converts the color into the linear RGB space (see http://www.sjbrown.co.uk/2004/05/14/gamma-correct-rendering/).
func (col Color) LinearRgb() (r, g, b float64) {
    r = linearize(col.R)
    g = linearize(col.G)
    b = linearize(col.B)
    return
}

// FastLinearRgb is much faster than and almost as accurate as LinearRgb.
func (col Color) FastLinearRgb() (r, g, b float64) {
    r = math.Pow(col.R, 2.2)
    g = math.Pow(col.G, 2.2)
    b = math.Pow(col.B, 2.2)
    return
}

func delinearize(v float64) float64 {
    if v <= 0.0031308 {
        return 12.92 * v
    }
    return 1.055 * math.Pow(v, 1.0/2.4) - 0.055
}

// LinearRgb creates an sRGB color out of the given linear RGB color (see http://www.sjbrown.co.uk/2004/05/14/gamma-correct-rendering/).
func LinearRgb(r, g, b float64) Color {
    return Color{delinearize(r), delinearize(g), delinearize(b)}
}

// FastLinearRgb is much faster than and almost as accurate as LinearRgb.
func FastLinearRgb(r, g, b float64) Color {
    return Color{math.Pow(r, 1.0/2.2), math.Pow(g, 1.0/2.2), math.Pow(b, 1.0/2.2)}
}

// XyzToLinearRgb converts from CIE XYZ-space to Linear RGB space.
func XyzToLinearRgb(x, y, z float64) (r, g, b float64) {
    r =  3.2404542*x - 1.5371385*y - 0.4985314*z
    g = -0.9692660*x + 1.8760108*y + 0.0415560*z
    b =  0.0556434*x - 0.2040259*y + 1.0572252*z
    return
}

func LinearRgbToXyz(r, g, b float64) (x, y, z float64) {
    x = 0.4124564*r + 0.3575761*g + 0.1804375*b
    y = 0.2126729*r + 0.7151522*g + 0.0721750*b
    z = 0.0193339*r + 0.1191920*g + 0.9503041*b
    return
}

/// XYZ ///
///////////
// http://www.sjbrown.co.uk/2004/05/14/gamma-correct-rendering/

func (col Color) Xyz() (x, y, z float64) {
    return LinearRgbToXyz(col.LinearRgb())
}

func Xyz(x, y, z float64) Color {
    return LinearRgb(XyzToLinearRgb(x, y, z))
}

/// xyY ///
///////////
// http://www.brucelindbloom.com/Eqn_XYZ_to_xyY.html

// Well, the name is bad, since it's xyY but Golang needs me to start with a
// capital letter to make the method public.
func XyzToXyy(X, Y, Z float64) (x, y, Yout float64) {
    return XyzToXyyWhiteRef(X, Y, Z, D65)
}

func XyzToXyyWhiteRef(X, Y, Z float64, wref [3]float64) (x, y, Yout float64) {
    Yout = Y
    N := X + Y + Z
    if math.Abs(N) < 1e-14 {
        // When we have black, Bruce Lindbloom recommends to use
        // the reference white's chromacity for x and y.
        x = wref[0] / (wref[0] + wref[1] + wref[2])
        y = wref[1] / (wref[0] + wref[1] + wref[2])
    } else {
        x = X / N
        y = Y / N
    }
    return
}

func XyyToXyz(x, y, Y float64) (X, Yout, Z float64) {
    Yout = Y

    if -1e-14 < y && y < 1e-14 {
        X = 0.0
        Z = 0.0
    } else {
        X = Y / y * x
        Z = Y / y * (1.0 - x - y)
    }

    return
}

// Converts the given color to CIE xyY space using D65 as reference white.
// (Note that the reference white is only used for black input.)
// x, y and Y are in [0..1]
func (col Color) Xyy() (x, y, Y float64) {
    return XyzToXyy(col.Xyz())
}

// Converts the given color to CIE xyY space, taking into account
// a given reference white. (i.e. the monitor's white)
// (Note that the reference white is only used for black input.)
// x, y and Y are in [0..1]
func (col Color) XyyWhiteRef(wref [3]float64) (x, y, Y float64) {
    X, Y2, Z := col.Xyz()
    return XyzToXyyWhiteRef(X, Y2, Z, wref)
}

// Generates a color by using data given in CIE xyY space.
// x, y and Y are in [0..1]
func Xyy(x, y, Y float64) Color {
    return Xyz(XyyToXyz(x, y, Y))
}

/// L*a*b* ///
//////////////
// http://en.wikipedia.org/wiki/Lab_color_space#CIELAB-CIEXYZ_conversions
// For L*a*b*, we need to L*a*b*<->XYZ->RGB and the first one is device dependent.

func lab_f(t float64) float64 {
    if t > 6.0/29.0 * 6.0/29.0 * 6.0/29.0 {
        return math.Cbrt(t)
    }
    return t/3.0 * 29.0/6.0 * 29.0/6.0 + 4.0/29.0
}

func XyzToLab(x, y, z float64) (l, a, b float64) {
    // Use D65 white as reference point by default.
    // http://www.fredmiranda.com/forum/topic/1035332
    // http://en.wikipedia.org/wiki/Standard_illuminant
    return XyzToLabWhiteRef(x, y, z, D65)
}

func XyzToLabWhiteRef(x, y, z float64, wref [3]float64) (l, a, b float64) {
    fy := lab_f(y/wref[1])
    l = 1.16*fy - 0.16
    a = 5.0*(lab_f(x/wref[0]) - fy)
    b = 2.0*(fy - lab_f(z/wref[2]))
    return
}

func lab_finv(t float64) float64 {
    if t > 6.0/29.0 {
        return t * t * t
    }
    return 3.0 * 6.0/29.0 * 6.0/29.0 * (t - 4.0/29.0)
}

func LabToXyz(l, a, b float64) (x, y, z float64) {
    // D65 white (see above).
    return LabToXyzWhiteRef(l, a, b, D65)
}

func LabToXyzWhiteRef(l, a, b float64, wref [3]float64) (x, y, z float64) {
    l2 := (l + 0.16) / 1.16
    x = wref[0] * lab_finv(l2 + a/5.0)
    y = wref[1] * lab_finv(l2)
    z = wref[2] * lab_finv(l2 - b/2.0)
    return
}

// Converts the given color to CIE L*a*b* space using D65 as reference white.
func (col Color) Lab() (l, a, b float64) {
    return XyzToLab(col.Xyz())
}

// Converts the given color to CIE L*a*b* space, taking into account
// a given reference white. (i.e. the monitor's white)
func (col Color) LabWhiteRef(wref [3]float64) (l, a, b float64) {
    x, y, z := col.Xyz()
    return XyzToLabWhiteRef(x, y, z, wref)
}

// Generates a color by using data given in CIE L*a*b* space using D65 as reference white.
func Lab(l, a, b float64) Color {
    return Xyz(LabToXyz(l, a, b))
}

// Generates a color by using data given in CIE L*a*b* space, taking
// into account a given reference white. (i.e. the monitor's white)
func LabWhiteRef(l, a, b float64, wref [3]float64) Color {
    return Xyz(LabToXyzWhiteRef(l, a, b, wref))
}

// DistanceLab is a good measure of visual similarity between two colors!
// A result of 0 would mean identical colors, while a result of 1 or higher
// means the colors differ a lot.
func (c1 Color) DistanceLab(c2 Color) float64 {
    l1, a1, b1 := c1.Lab()
    l2, a2, b2 := c2.Lab()
    return math.Sqrt(sq(l1-l2) + sq(a1-a2) + sq(b1-b2))
}

// That's actually the same, but I don't want to break code.
func (c1 Color) DistanceCIE76(c2 Color) float64 {
    return c1.DistanceLab(c2)
}

// Uses the CIE94 formula to calculate color distance. More accurate than
// DistanceLab, but also more work.
func (cl Color) DistanceCIE94(cr Color) float64 {
    l1, a1, b1 := cl.Lab()
    l2, a2, b2 := cr.Lab()

    kl := 1.0
    k1 := 0.045
    k2 := 0.015

    deltaL := l1 - l2
    c1 := math.Sqrt(sq(a1) + sq(b1))
    c2 := math.Sqrt(sq(a2) + sq(b2))
    deltaCab := c1 - c2
    deltaHab := math.Sqrt(sq(a1-a2) + sq(b1-b2) - sq(deltaCab))
    sl := 1.0
    sc := 1.0 + k1*c1
    sh := 1.0 + k2*c1

    return math.Sqrt(sq(deltaL/(kl*sl)) + sq(deltaCab/sc) + sq(deltaHab/sh))
}

// BlendLab blends two colors in the L*a*b* color-space, which should result in a smoother blend.
// t == 0 results in c1, t == 1 results in c2
func (c1 Color) BlendLab(c2 Color, t float64) Color {
    l1, a1, b1 := c1.Lab()
    l2, a2, b2 := c2.Lab()
    return Lab(l1 + t*(l2 - l1),
               a1 + t*(a2 - a1),
               b1 + t*(b2 - b1))
}

/// L*u*v* ///
//////////////
// http://en.wikipedia.org/wiki/CIELUV#XYZ_.E2.86.92_CIELUV_and_CIELUV_.E2.86.92_XYZ_conversions
// For L*u*v*, we need to L*u*v*<->XYZ<->RGB and the first one is device dependent.

func XyzToLuv(x, y, z float64) (l, a, b float64) {
    // Use D65 white as reference point by default.
    // http://www.fredmiranda.com/forum/topic/1035332
    // http://en.wikipedia.org/wiki/Standard_illuminant
    return XyzToLuvWhiteRef(x, y, z, D65)
}

func XyzToLuvWhiteRef(x, y, z float64, wref [3]float64) (l, u, v float64) {
    if y/wref[1] <= 6.0/29.0 * 6.0/29.0 * 6.0/29.0 {
        l = y/wref[1] * 29.0/3.0 * 29.0/3.0 * 29.0/3.0
    } else {
        l = 1.16 * math.Cbrt(y/wref[1]) - 0.16
    }
    ubis, vbis := xyz_to_uv(x, y, z)
    un, vn := xyz_to_uv(wref[0], wref[1], wref[2])
    u = 13.0*l * (ubis - un)
    v = 13.0*l * (vbis - vn)
    return
}

// For this part, we do as R's graphics.hcl does, not as wikipedia does.
// Or is it the same?
func xyz_to_uv(x, y, z float64) (u, v float64) {
    denom := x + 15.0*y + 3.0*z
    if denom == 0.0 {
        u, v = 0.0, 0.0
    } else {
        u = 4.0*x/denom
        v = 9.0*y/denom
    }
    return
}

func LuvToXyz(l, u, v float64) (x, y, z float64) {
    // D65 white (see above).
    return LuvToXyzWhiteRef(l, u, v, D65)
}

func LuvToXyzWhiteRef(l, u, v float64, wref [3]float64) (x, y, z float64) {
    //y = wref[1] * lab_finv((l + 0.16) / 1.16)
    if l <= 0.08 {
        y = wref[1] * l * 100.0 * 3.0/29.0 * 3.0/29.0 * 3.0/29.0
    } else {
        y = wref[1] * cub((l+0.16)/1.16)
    }
    un, vn := xyz_to_uv(wref[0], wref[1], wref[2])
    if l != 0.0 {
        ubis := u/(13.0*l) + un
        vbis := v/(13.0*l) + vn
        x = y*9.0*ubis/(4.0*vbis)
        z = y*(12.0-3.0*ubis-20.0*vbis)/(4.0*vbis)
    } else {
        x, y = 0.0, 0.0
    }
    return
}

// Converts the given color to CIE L*u*v* space using D65 as reference white.
// L* is in [0..1] and both u* and v* are in about [-1..1]
func (col Color) Luv() (l, u, v float64) {
    return XyzToLuv(col.Xyz())
}

// Converts the given color to CIE L*u*v* space, taking into account
// a given reference white. (i.e. the monitor's white)
// L* is in [0..1] and both u* and v* are in about [-1..1]
func (col Color) LuvWhiteRef(wref [3]float64) (l, u, v float64) {
    x, y, z := col.Xyz()
    return XyzToLuvWhiteRef(x, y, z, wref)
}

// Generates a color by using data given in CIE L*u*v* space using D65 as reference white.
// L* is in [0..1] and both u* and v* are in about [-1..1]
func Luv(l, u, v float64) Color {
    return Xyz(LuvToXyz(l, u, v))
}

// Generates a color by using data given in CIE L*u*v* space, taking
// into account a given reference white. (i.e. the monitor's white)
// L* is in [0..1] and both u* and v* are in about [-1..1]
func LuvWhiteRef(l, u, v float64, wref [3]float64) Color {
    return Xyz(LuvToXyzWhiteRef(l, u, v, wref))
}

// DistanceLuv is a good measure of visual similarity between two colors!
// A result of 0 would mean identical colors, while a result of 1 or higher
// means the colors differ a lot.
func (c1 Color) DistanceLuv(c2 Color) float64 {
    l1, u1, v1 := c1.Luv()
    l2, u2, v2 := c2.Luv()
    return math.Sqrt(sq(l1-l2) + sq(u1-u2) + sq(v1-v2))
}

// BlendLuv blends two colors in the CIE-L*u*v* color-space, which should result in a smoother blend.
// t == 0 results in c1, t == 1 results in c2
func (c1 Color) BlendLuv(c2 Color, t float64) Color {
    l1, u1, v1 := c1.Luv()
    l2, u2, v2 := c2.Luv()
    return Luv(l1 + t*(l2 - l1),
               u1 + t*(u2 - u1),
               v1 + t*(v2 - v1))
}

/// HCL ///
///////////
// HCL is nothing else than L*a*b* in cylindrical coordinates!
// (this was wrong on English wikipedia, I fixed it, let's hope the fix stays.)
// But it is widely popular since it is a "correct HSV"
// http://www.hunterlab.com/appnotes/an09_96a.pdf

// Converts the given color to HCL space using D65 as reference white.
// H values are in [0..360], C and L values are in [0..1] although C can overshoot 1.0
func (col Color) Hcl() (h, c, l float64) {
    return col.HclWhiteRef(D65)
}

func LabToHcl(L, a, b float64) (h, c, l float64) {
    // Oops, floating point workaround necessary if a ~= b and both are very small (i.e. almost zero).
    if math.Abs(b - a) > 1e-4 && math.Abs(a) > 1e-4 {
        h = math.Mod(57.29577951308232087721*math.Atan2(b, a) + 360.0, 360.0) // Rad2Deg
    } else {
        h = 0.0
    }
    c = math.Sqrt(sq(a) + sq(b))
    l = L
    return
}

// Converts the given color to HCL space, taking into account
// a given reference white. (i.e. the monitor's white)
// H values are in [0..360], C and L values are in [0..1]
func (col Color) HclWhiteRef(wref [3]float64) (h, c, l float64) {
    L, a, b := col.LabWhiteRef(wref)
    return LabToHcl(L, a, b)
}

// Generates a color by using data given in HCL space using D65 as reference white.
// H values are in [0..360], C and L values are in [0..1]
func Hcl(h, c, l float64) Color {
    return HclWhiteRef(h, c, l, D65)
}

func HclToLab(h, c, l float64) (L, a, b float64) {
    H := 0.01745329251994329576*h // Deg2Rad
    a = c*math.Cos(H)
    b = c*math.Sin(H)
    L = l
    return
}

// Generates a color by using data given in HCL space, taking
// into account a given reference white. (i.e. the monitor's white)
// H values are in [0..360], C and L values are in [0..1]
func HclWhiteRef(h, c, l float64, wref [3]float64) Color {
    L, a, b := HclToLab(h, c, l)
    return LabWhiteRef(L, a, b, wref)
}

// BlendHcl blends two colors in the CIE-L*C*h° color-space, which should result in a smoother blend.
// t == 0 results in c1, t == 1 results in c2
func (col1 Color) BlendHcl(col2 Color, t float64) Color {
    h1, c1, l1 := col1.Hcl()
    h2, c2, l2 := col2.Hcl()

    // We know that h are both in [0..360]
    return Hcl(interp_angle(h1, h2, t), c1 + t*(c2 - c1), l1 + t*(l2 - l1))
}
