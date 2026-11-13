package color

// The 256-color palette: entries 0-15 are the ANSI colors, 16-231 a 6x6x6 color
// cube, and 232-255 a 24-step gray ramp.
const (
	cubeStart  = 16
	cubeSide   = 6
	grayStart  = 232
	grayCount  = 24
	grayFirst  = 8  // brightness of the darkest gray
	grayStep   = 10 // brightness added per step
	channelMax = 255
)

// ansiRGB holds the xterm default colors of the 16 ANSI palette entries.
// Terminals remap these to their theme, so they are only a reference for
// picking the nearest ANSI color.
var ansiRGB = [ansiColors]rgb{
	{0, 0, 0},
	{205, 0, 0},
	{0, 205, 0},
	{205, 205, 0},
	{0, 0, 238},
	{205, 0, 205},
	{0, 205, 205},
	{229, 229, 229},
	{127, 127, 127},
	{255, 0, 0},
	{0, 255, 0},
	{255, 255, 0},
	{92, 92, 255},
	{255, 0, 255},
	{0, 255, 255},
	{255, 255, 255},
}

// cubeLevels holds the brightness of each of the six steps along one axis of
// the color cube.
var cubeLevels = [cubeSide]int{0, 95, 135, 175, 215, 255}

// paletteRGB returns the color of a palette entry.
func paletteRGB(index int) rgb {
	switch {
	case index < cubeStart:
		return ansiRGB[index]
	case index < grayStart:
		i := index - cubeStart
		return rgb{
			r: cubeLevels[i/(cubeSide*cubeSide)],
			g: cubeLevels[i/cubeSide%cubeSide],
			b: cubeLevels[i%cubeSide],
		}
	default:
		level := grayFirst + grayStep*(index-grayStart)
		return rgb{level, level, level}
	}
}

// nearestPalette returns the cube or gray-ramp entry closest to c. It leaves
// entries 0-15 alone: a terminal theme decides what those look like.
func nearestPalette(c rgb) int {
	r, g, b := nearestCubeStep(c.r), nearestCubeStep(c.g), nearestCubeStep(c.b)
	cube := cubeStart + r*cubeSide*cubeSide + g*cubeSide + b

	step := min(max((c.r+c.g+c.b)/3-grayFirst+grayStep/2, 0)/grayStep, grayCount-1)
	gray := grayStart + step

	if distance(c, paletteRGB(gray)) < distance(c, paletteRGB(cube)) {
		return gray
	}
	return cube
}

// nearestCubeStep returns the cube step whose brightness is closest to channel.
func nearestCubeStep(channel int) int {
	best := 0
	for i, level := range cubeLevels {
		if abs(level-channel) < abs(cubeLevels[best]-channel) {
			best = i
		}
	}
	return best
}

// nearestANSI returns the ANSI palette entry, 0-15, closest to c.
func nearestANSI(c rgb) int {
	best := 0
	for i := range ansiRGB {
		if distance(c, ansiRGB[i]) < distance(c, ansiRGB[best]) {
			best = i
		}
	}
	return best
}

// distance is the squared Euclidean distance between two colors.
func distance(a, b rgb) int {
	dr, dg, db := a.r-b.r, a.g-b.g, a.b-b.b
	return dr*dr + dg*dg + db*db
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
