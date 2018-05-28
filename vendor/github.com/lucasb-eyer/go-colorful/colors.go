// The colorful package provides all kinds of functions for working with colors.
package colorful

import (
	"math"
)

// A color is stored internally using sRGB (standard RGB) values in the range 0-1
type Color struct {
	R, G, B float64
}

// This is the default reference white point.
var D65 = [3]float64{0.95047, 1.00000, 1.08883}

func sq(v float64) float64 {
	return v * v
}

/// Linear ///
//////////////
// http://www.sjbrown.co.uk/2004/05/14/gamma-correct-rendering/
// http://www.brucelindbloom.com/Eqn_RGB_to_XYZ.html

func linearize(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// LinearRgb converts the color into the linear RGB space (see http://www.sjbrown.co.uk/2004/05/14/gamma-correct-rendering/).
func (col Color) LinearRgb() (r, g, b float64) {
	r = linearize(col.R)
	g = linearize(col.G)
	b = linearize(col.B)
	return
}

func delinearize(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}

// LinearRgb creates an sRGB color out of the given linear RGB color (see http://www.sjbrown.co.uk/2004/05/14/gamma-correct-rendering/).
func LinearRgb(r, g, b float64) Color {
	return Color{delinearize(r), delinearize(g), delinearize(b)}
}

// XyzToLinearRgb converts from CIE XYZ-space to Linear RGB space.
func XyzToLinearRgb(x, y, z float64) (r, g, b float64) {
	r = 3.2404542*x - 1.5371385*y - 0.4985314*z
	g = -0.9692660*x + 1.8760108*y + 0.0415560*z
	b = 0.0556434*x - 0.2040259*y + 1.0572252*z
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

/// L*a*b* ///
//////////////
// http://en.wikipedia.org/wiki/Lab_color_space#CIELAB-CIEXYZ_conversions
// For L*a*b*, we need to L*a*b*<->XYZ->RGB and the first one is device dependent.

func lab_f(t float64) float64 {
	if t > 6.0/29.0*6.0/29.0*6.0/29.0 {
		return math.Cbrt(t)
	}
	return t/3.0*29.0/6.0*29.0/6.0 + 4.0/29.0
}

func XyzToLab(x, y, z float64) (l, a, b float64) {
	// Use D65 white as reference point by default.
	// http://www.fredmiranda.com/forum/topic/1035332
	// http://en.wikipedia.org/wiki/Standard_illuminant
	return XyzToLabWhiteRef(x, y, z, D65)
}

func XyzToLabWhiteRef(x, y, z float64, wref [3]float64) (l, a, b float64) {
	fy := lab_f(y / wref[1])
	l = 1.16*fy - 0.16
	a = 5.0 * (lab_f(x/wref[0]) - fy)
	b = 2.0 * (fy - lab_f(z/wref[2]))
	return
}

func lab_finv(t float64) float64 {
	if t > 6.0/29.0 {
		return t * t * t
	}
	return 3.0 * 6.0 / 29.0 * 6.0 / 29.0 * (t - 4.0/29.0)
}

func LabToXyz(l, a, b float64) (x, y, z float64) {
	// D65 white (see above).
	return LabToXyzWhiteRef(l, a, b, D65)
}

func LabToXyzWhiteRef(l, a, b float64, wref [3]float64) (x, y, z float64) {
	l2 := (l + 0.16) / 1.16
	x = wref[0] * lab_finv(l2+a/5.0)
	y = wref[1] * lab_finv(l2)
	z = wref[2] * lab_finv(l2-b/2.0)
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
// WARNING: many combinations of `l`, `a`, and `b` values do not have corresponding
//          valid RGB values, check the FAQ in the README if you're unsure.
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
	if math.Abs(b-a) > 1e-4 && math.Abs(a) > 1e-4 {
		h = math.Mod(57.29577951308232087721*math.Atan2(b, a)+360.0, 360.0) // Rad2Deg
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
// WARNING: many combinations of `l`, `a`, and `b` values do not have corresponding
//          valid RGB values, check the FAQ in the README if you're unsure.
func Hcl(h, c, l float64) Color {
	return HclWhiteRef(h, c, l, D65)
}

func HclToLab(h, c, l float64) (L, a, b float64) {
	H := 0.01745329251994329576 * h // Deg2Rad
	a = c * math.Cos(H)
	b = c * math.Sin(H)
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
