/**
 * LIMBO 3D/4D Engine - Advanced GTA 7 Style
 * Professional atmospheric 3D engine with LIMBO aesthetics
 */

class Limbo3DEngine {
    constructor(canvas) {
        this.canvas = canvas;
        this.gl = canvas.getContext('webgl2') || canvas.getContext('webgl');
        if (!this.gl) throw new Error('WebGL not supported');
        
        // Advanced 3D/4D rendering pipeline
        this.programs = {};
        this.meshes = {};
        this.textures = {};
        
        // LIMBO-style atmosphere
        this.fogDensity = 0.15;
        this.ambientLight = [0.02, 0.02, 0.03]; // Very dark LIMBO ambient
        this.shadowIntensity = 0.95;
        
        // GTA 7-style post-processing
        this.bloomIntensity = 1.2;
        this.vignetteStrength = 0.8;
        this.chromaticAberration = 0.003;
        
        // 4D time/dimension effects
        this.timeWarp = 0.0;
        this.dimensionShift = 0.0;
        
        // Camera system
        this.camera = {
            position: [0, 2, 5],
            target: [0, 0, 0],
            up: [0, 1, 0],
            fov: 75,
            near: 0.1,
            far: 100
        };
        
        // Game entities
        this.player = {
            position: [0, 0, 0],
            velocity: [0, 0, 0],
            rotation: 0,
            scale: [1, 2, 0.3], // LIMBO boy proportions
            boundingBox: { width: 0.6, height: 2.0, depth: 0.3 }
        };
        
        this.obstacles = [];
        this.particles = [];
        this.environment = [];
        
        // Leaderboard and scoring
        this.score = 0;
        this.multiplier = 1.0;
        this.leaderboard = this.loadLeaderboard();
        
        this.init();
    }
    
    async init() {
        this.setupShaders();
        this.createMeshes();
        this.setupFramebuffers();
        this.createEnvironment();
        this.startRenderLoop();
    }
    
    setupShaders() {
        // LIMBO silhouette vertex shader
        const limboVertexSource = `#version 300 es
        precision highp float;
        
        in vec3 position;
        in vec3 normal;
        in vec2 texCoord;
        
        uniform mat4 modelMatrix;
        uniform mat4 viewMatrix;
        uniform mat4 projectionMatrix;
        uniform mat4 lightSpaceMatrix;
        uniform float time;
        uniform float dimensionShift;
        
        out vec3 worldPos;
        out vec3 worldNormal;
        out vec2 vTexCoord;
        out vec4 shadowCoord;
        out float depth;
        
        void main() {
            // 4D dimension shifting effect
            vec4 pos4d = vec4(position, 1.0);
            pos4d.w += sin(time * 0.5 + position.x) * dimensionShift * 0.1;
            
            // Transform to world space
            vec4 worldPosition = modelMatrix * pos4d;
            worldPos = worldPosition.xyz;
            worldNormal = normalize(mat3(modelMatrix) * normal);
            
            // LIMBO-style depth warping
            float limboWarp = sin(worldPos.z * 0.1 + time * 0.3) * 0.05;
            worldPosition.y += limboWarp;
            
            // Camera transformation
            vec4 viewPos = viewMatrix * worldPosition;
            gl_Position = projectionMatrix * viewPos;
            
            // Shadow mapping
            shadowCoord = lightSpaceMatrix * worldPosition;
            
            depth = gl_Position.z / gl_Position.w;
            vTexCoord = texCoord;
        }`;
        
        // Advanced LIMBO fragment shader with GTA 7 effects
        const limboFragmentSource = `#version 300 es
        precision highp float;
        
        in vec3 worldPos;
        in vec3 worldNormal;
        in vec2 vTexCoord;
        in vec4 shadowCoord;
        in float depth;
        
        uniform sampler2D shadowMap;
        uniform samplerCube envMap;
        uniform float time;
        uniform float fogDensity;
        uniform vec3 ambientLight;
        uniform vec3 cameraPos;
        uniform float shadowIntensity;
        uniform bool isPlayer;
        uniform float bloomIntensity;
        
        out vec4 fragColor;
        
        float calculateShadow() {
            vec3 projCoords = shadowCoord.xyz / shadowCoord.w;
            projCoords = projCoords * 0.5 + 0.5;
            
            if (projCoords.z > 1.0) return 0.0;
            
            float closestDepth = texture(shadowMap, projCoords.xy).r;
            float currentDepth = projCoords.z;
            
            // Soft shadows for LIMBO atmosphere
            float bias = 0.005;
            float shadow = 0.0;
            vec2 texelSize = 1.0 / textureSize(shadowMap, 0);
            
            for(int x = -1; x <= 1; ++x) {
                for(int y = -1; y <= 1; ++y) {
                    float pcfDepth = texture(shadowMap, projCoords.xy + vec2(x, y) * texelSize).r;
                    shadow += currentDepth - bias > pcfDepth ? shadowIntensity : 0.0;
                }
            }
            
            return shadow / 9.0;
        }
        
        void main() {
            vec3 normal = normalize(worldNormal);
            vec3 viewDir = normalize(cameraPos - worldPos);
            
            // LIMBO silhouette rendering
            if (isPlayer) {
                // Pure black silhouette with subtle rim lighting
                float rim = 1.0 - max(dot(viewDir, normal), 0.0);
                rim = pow(rim, 3.0) * 0.1;
                fragColor = vec4(rim, rim, rim * 1.2, 1.0);
            } else {
                // Environmental objects - dark and atmospheric
                float shadow = calculateShadow();
                
                // Base dark color
                vec3 baseColor = vec3(0.1, 0.08, 0.06);
                
                // Atmospheric lighting
                float ndl = max(dot(normal, normalize(vec3(-0.3, -1.0, -0.5))), 0.0);
                vec3 lighting = ambientLight + baseColor * ndl * (1.0 - shadow);
                
                // LIMBO fog effect
                float distance = length(cameraPos - worldPos);
                float fogFactor = exp(-fogDensity * distance);
                fogFactor = clamp(fogFactor, 0.0, 1.0);
                
                vec3 fogColor = vec3(0.05, 0.04, 0.03);
                vec3 finalColor = mix(fogColor, lighting, fogFactor);
                
                // Bloom for atmospheric glow
                float luminance = dot(finalColor, vec3(0.299, 0.587, 0.114));
                vec3 bloom = finalColor * smoothstep(0.1, 0.3, luminance) * bloomIntensity;
                finalColor += bloom;
                
                fragColor = vec4(finalColor, 1.0);
            }
            
            // 4D dimensional effects
            float dimensionalNoise = sin(worldPos.x * 10.0 + time) * 
                                   cos(worldPos.z * 8.0 + time * 0.7) * 0.02;
            fragColor.rgb += vec3(dimensionalNoise);
        }`;
        
        this.programs.limbo = this.createProgram(limboVertexSource, limboFragmentSource);
        
        // Post-processing shader for GTA 7 effects
        const postProcessFragment = `#version 300 es
        precision highp float;
        
        in vec2 texCoord;
        uniform sampler2D scene;
        uniform float time;
        uniform float vignetteStrength;
        uniform float chromaticAberration;
        
        out vec4 fragColor;
        
        void main() {
            vec2 uv = texCoord;
            vec2 center = vec2(0.5);
            
            // Chromatic aberration
            float aberration = chromaticAberration;
            vec2 distortion = (uv - center) * aberration;
            
            float r = texture(scene, uv - distortion).r;
            float g = texture(scene, uv).g;
            float b = texture(scene, uv + distortion).b;
            
            vec3 color = vec3(r, g, b);
            
            // Vignette effect
            float dist = distance(uv, center);
            float vignette = 1.0 - smoothstep(0.5, 1.0, dist * vignetteStrength);
            color *= vignette;
            
            // Film grain for LIMBO atmosphere
            float grain = fract(sin(dot(uv * time, vec2(12.9898, 78.233))) * 43758.5453) * 0.1;
            color += vec3(grain) * 0.05;
            
            fragColor = vec4(color, 1.0);
        }`;
        
        this.programs.postProcess = this.createProgram(this.getQuadVertexShader(), postProcessFragment);
    }
    
    createMeshes() {
        // LIMBO boy mesh - simple but iconic silhouette
        this.meshes.player = this.createBox(0.3, 1.0, 0.15);
        
        // Obstacle meshes - various LIMBO-style shapes
        this.meshes.obstacle = this.createBox(1, 2, 1);
        this.meshes.tree = this.createLimboTree();
        this.meshes.machinery = this.createMachinery();
        
        // Environment meshes
        this.meshes.ground = this.createGround(100, 100);
        this.meshes.skybox = this.createSkybox();
    }
    
    createLimboTree() {
        // Procedural LIMBO-style tree generation
        const vertices = [];
        const indices = [];
        
        // Twisted trunk
        for (let i = 0; i < 10; i++) {
            const y = i * 0.5;
            const twist = Math.sin(i * 0.3) * 0.2;
            const radius = 0.2 - i * 0.01;
            
            // Create trunk segments
            for (let j = 0; j < 8; j++) {
                const angle = (j / 8) * Math.PI * 2;
                const x = Math.cos(angle) * radius + twist;
                const z = Math.sin(angle) * radius;
                
                vertices.push(x, y, z, 0, 1, 0, 0, 0); // pos, normal, uv
            }
        }
        
        // Generate indices for trunk
        for (let i = 0; i < 9; i++) {
            for (let j = 0; j < 8; j++) {
                const curr = i * 8 + j;
                const next = i * 8 + ((j + 1) % 8);
                const currNext = (i + 1) * 8 + j;
                const nextNext = (i + 1) * 8 + ((j + 1) % 8);
                
                // Two triangles per quad
                indices.push(curr, next, currNext);
                indices.push(next, nextNext, currNext);
            }
        }
        
        // Add twisted branches
        const branchCount = 5 + Math.random() * 5;
        for (let b = 0; b < branchCount; b++) {
            const branchY = 2 + Math.random() * 3;
            const branchAngle = (b / branchCount) * Math.PI * 2;
            
            // Branch geometry...
            // (Additional procedural branch generation code here)
        }
        
        return this.createMeshFromData(vertices, indices);
    }
    
    createMachinery() {
        // Industrial LIMBO-style machinery
        const parts = [];
        
        // Base structure
        parts.push({
            type: 'box',
            position: [0, 1, 0],
            scale: [2, 2, 1],
            rotation: [0, Math.random() * 0.5, 0]
        });
        
        // Pipes and mechanical elements
        for (let i = 0; i < 3; i++) {
            parts.push({
                type: 'cylinder',
                position: [Math.random() * 2 - 1, Math.random() * 3, Math.random() * 2 - 1],
                scale: [0.1, 1 + Math.random(), 0.1],
                rotation: [Math.random() * Math.PI, 0, Math.random() * Math.PI]
            });
        }
        
        return this.createComplexMesh(parts);
    }
    
    // Advanced 3D rendering methods
    render() {
        // Multi-pass rendering for advanced effects
        this.renderShadowPass();
        this.renderScenePass();
        this.renderPostProcess();
        
        // Update animations
        this.updateAnimations();
        
        // Update scoring and leaderboard
        this.updateGameplay();
    }
    
    renderShadowPass() {
        // Render shadow map for LIMBO atmospheric shadows
        this.gl.bindFramebuffer(this.gl.FRAMEBUFFER, this.shadowFramebuffer);
        this.gl.viewport(0, 0, 2048, 2048);
        this.gl.clear(this.gl.DEPTH_BUFFER_BIT);
        
        // Render all shadow casters
        this.renderPlayer(true);
        this.renderObstacles(true);
        this.renderEnvironment(true);
    }
    
    renderScenePass() {
        // Main scene rendering
        this.gl.bindFramebuffer(this.gl.FRAMEBUFFER, this.sceneFramebuffer);
        this.gl.viewport(0, 0, this.canvas.width, this.canvas.height);
        this.gl.clear(this.gl.COLOR_BUFFER_BIT | this.gl.DEPTH_BUFFER_BIT);
        
        // Render scene with full lighting and effects
        this.renderSkybox();
        this.renderEnvironment();
        this.renderObstacles();
        this.renderPlayer();
        this.renderParticles();
    }
    
    renderPostProcess() {
        // GTA 7-style post-processing
        this.gl.bindFramebuffer(this.gl.FRAMEBUFFER, null);
        this.gl.viewport(0, 0, this.canvas.width, this.canvas.height);
        
        this.gl.useProgram(this.programs.postProcess);
        this.gl.uniform1f(this.gl.getUniformLocation(this.programs.postProcess, 'time'), performance.now() / 1000);
        this.gl.uniform1f(this.gl.getUniformLocation(this.programs.postProcess, 'vignetteStrength'), this.vignetteStrength);
        this.gl.uniform1f(this.gl.getUniformLocation(this.programs.postProcess, 'chromaticAberration'), this.chromaticAberration);
        
        this.renderFullscreenQuad();
    }
    
    // Game mechanics with advanced scoring
    updateGameplay() {
        // Advanced scoring system
        const basePoints = 10;
        const distanceBonus = Math.floor(this.player.position[2] / 10) * 5;
        const styleBonus = this.calculateStyleBonus();
        
        this.score += (basePoints + distanceBonus) * this.multiplier + styleBonus;
        
        // Multiplier system
        if (this.isNearMissing()) {
            this.multiplier = Math.min(this.multiplier + 0.1, 5.0);
        } else {
            this.multiplier = Math.max(this.multiplier - 0.01, 1.0);
        }
        
        // Dynamic difficulty
        this.spawnObstacles();
        this.updateEnvironment();
    }
    
    calculateStyleBonus() {
        // Bonus points for stylish moves
        let bonus = 0;
        
        // Near-miss bonus
        if (this.isNearMissing()) bonus += 50;
        
        // Speed bonus
        const speed = Math.sqrt(
            this.player.velocity[0] ** 2 + 
            this.player.velocity[2] ** 2
        );
        if (speed > 5) bonus += Math.floor(speed) * 10;
        
        // Combo bonus
        if (this.comboCount > 5) bonus += this.comboCount * 20;
        
        return bonus;
    }
    
    // Advanced leaderboard system
    saveScore(playerName) {
        const entry = {
            name: playerName,
            score: this.score,
            date: new Date().toISOString(),
            maxMultiplier: Math.floor(this.maxMultiplier * 10) / 10,
            styleScore: this.totalStyleBonus
        };
        
        this.leaderboard.push(entry);
        this.leaderboard.sort((a, b) => b.score - a.score);
        this.leaderboard = this.leaderboard.slice(0, 10); // Top 10
        
        localStorage.setItem('limbo3d-leaderboard', JSON.stringify(this.leaderboard));
        return this.leaderboard.indexOf(entry) + 1; // Return ranking
    }
    
    loadLeaderboard() {
        const saved = localStorage.getItem('limbo3d-leaderboard');
        return saved ? JSON.parse(saved) : [];
    }
    
    // Utility methods
    createProgram(vertexSource, fragmentSource) {
        const program = this.gl.createProgram();
        
        const vertexShader = this.compileShader(vertexSource, this.gl.VERTEX_SHADER);
        const fragmentShader = this.compileShader(fragmentSource, this.gl.FRAGMENT_SHADER);
        
        this.gl.attachShader(program, vertexShader);
        this.gl.attachShader(program, fragmentShader);
        this.gl.linkProgram(program);
        
        if (!this.gl.getProgramParameter(program, this.gl.LINK_STATUS)) {
            throw new Error('Program linking failed: ' + this.gl.getProgramInfoLog(program));
        }
        
        return program;
    }
    
    compileShader(source, type) {
        const shader = this.gl.createShader(type);
        this.gl.shaderSource(shader, source);
        this.gl.compileShader(shader);
        
        if (!this.gl.getShaderParameter(shader, this.gl.COMPILE_STATUS)) {
            throw new Error('Shader compilation failed: ' + this.gl.getShaderInfoLog(shader));
        }
        
        return shader;
    }
    
    startRenderLoop() {
        const render = () => {
            this.render();
            requestAnimationFrame(render);
        };
        render();
    }
}

// Export for use
window.Limbo3DEngine = Limbo3DEngine;
