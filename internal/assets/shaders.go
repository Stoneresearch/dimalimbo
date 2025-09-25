package assets

// Kage (Ebiten) fragment shader for neon/scanline/CRT-like post effect.
// Draw the scene to an offscreen image, then pass it into this shader via DrawRectShader.
// The shader expects uniforms: time (float), intensity (float), and resolution (vec2).
const NeonCRTShader = `
package main

// Uniforms
var time float
var intensity float
var resolution vec2

// Input image is at image0

func hash(p vec2) float {
    // Simple hash for noise
    return fract(sin(dot(p, vec2(12.9898,78.233))) * 43758.5453)
}

func vignette(uv vec2) float {
    // center-based falloff
    c := uv - 0.5
    d := dot(c, c)
    return clamp(1.0 - d*1.5, 0.0, 1.0)
}

func scanline(y float) float {
    return 0.85 + 0.15 * sin(y * resolution.y * 3.14159)
}

func aberrationSample(uv vec2, offset float) vec3 {
    // simple chromatic aberration by sampling R,G,B at slight offsets
    r := imageSrc0At(uv + vec2(offset, 0)).r
    g := imageSrc0At(uv).g
    b := imageSrc0At(uv - vec2(offset, 0)).b
    return vec3(r,g,b)
}

func glitchOffset(uv vec2) vec2 {
    // horizontal glitch lines based on noise
    n := hash(vec2(floor(uv.y * resolution.y*0.5), floor(time*20.0)))
    g := step(0.98, n) * (hash(vec2(time, uv.y)) - 0.5) * 0.02 * intensity
    return vec2(g, 0)
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
    uv := texCoord
    // subtle barrel distortion
    centered := (uv - 0.5) * 2.0
    r2 := dot(centered, centered)
    k := 0.06 * intensity
    distorted := 0.5 + centered * (1.0 + k*r2)

    // glitch
    distorted += glitchOffset(distorted)

    // aberration
    abOff := (0.003 + 0.002*sin(time*1.7)) * intensity
    col := aberrationSample(distorted, abOff)

    // neon curve: push mids and add soft bloom-ish effect
    col = pow(col, vec3(0.9)) * 1.05

    // scanlines & vignette
    v := vignette(uv)
    s := scanline(uv.y)
    col *= v * s

    return vec4(col, 1.0)
}
`
