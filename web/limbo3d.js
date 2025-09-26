/**
 * LIMBO 3D - Advanced 3D/4D Engine 
 * GTA 7-style graphics with LIMBO atmosphere
 * Physics-based gameplay with leaderboard
 */

class Limbo3D {
    constructor() {
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.world = null; // Physics world
        
        // Game state
        this.gameState = 'menu'; // menu, playing, gameOver, leaderboard
        this.score = 0;
        this.lives = 3;
        this.level = 1;
        
        // 3D Objects
        this.player = null;
        this.obstacles = [];
        this.platforms = [];
        this.particles = [];
        
        // Physics bodies
        this.playerBody = null;
        this.obstaclesBodies = [];
        
        // GTA 7-style effects
        this.composer = null;
        this.bloomPass = null;
        this.filmPass = null;
        
        // LIMBO atmospheric elements  
        this.fog = null;
        this.ambientLight = null;
        this.directionalLight = null;
        
        // Leaderboard system
        this.leaderboard = this.loadLeaderboard();
        this.playerName = '';
        
        // Controls
        this.keys = {};
        this.touch = { active: false, x: 0, y: 0 };
        
        this.init();
    }
    
    async init() {
        console.log('🎮 Initializing LIMBO 3D...');
        
        // Load Three.js and physics engine
        await this.loadDependencies();
        
        // Setup 3D scene
        this.setupScene();
        this.setupRenderer();
        this.setupCamera();
        this.setupLights();
        this.setupPhysics();
        this.setupPostProcessing();
        
        // Setup game elements
        this.createPlayer();
        this.createWorld();
        this.setupControls();
        this.setupUI();
        
        // Start game loop
        this.gameLoop();
        
        console.log('✅ LIMBO 3D initialized successfully');
    }
    
    async loadDependencies() {
        // Load Three.js
        const script1 = document.createElement('script');
        script1.src = 'https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js';
        document.head.appendChild(script1);
        
        // Load Cannon.js for physics
        const script2 = document.createElement('script');
        script2.src = 'https://cdnjs.cloudflare.com/ajax/libs/cannon.js/0.20.0/cannon.min.js';
        document.head.appendChild(script2);
        
        // Load postprocessing
        const script3 = document.createElement('script');
        script3.src = 'https://cdnjs.cloudflare.com/ajax/libs/postprocessing/6.21.3/postprocessing.umd.min.js';
        document.head.appendChild(script3);
        
        return new Promise(resolve => {
            let loaded = 0;
            const onLoad = () => {
                loaded++;
                if (loaded === 3) resolve();
            };
            script1.onload = onLoad;
            script2.onload = onLoad; 
            script3.onload = onLoad;
        });
    }
    
    setupScene() {
        // Create 3D scene with LIMBO atmosphere
        this.scene = new THREE.Scene();
        
        // LIMBO-style fog for atmospheric depth
        this.scene.fog = new THREE.FogExp2(0x000000, 0.002);
        this.scene.background = new THREE.Color(0x0a0a0a);
        
        console.log('🌫️ LIMBO atmospheric scene created');
    }
    
    setupRenderer() {
        // High-quality renderer for GTA 7-style graphics
        this.renderer = new THREE.WebGLRenderer({ 
            antialias: true,
            powerPreference: "high-performance"
        });
        
        this.renderer.setSize(window.innerWidth, window.innerHeight);
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        
        // Enable advanced rendering features
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        this.renderer.outputEncoding = THREE.sRGBEncoding;
        this.renderer.toneMapping = THREE.ACESFilmicToneMapping;
        this.renderer.toneMappingExposure = 0.8;
        
        // Add to DOM
        this.renderer.domElement.id = 'limbo3d-canvas';
        this.renderer.domElement.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            width: 100vw;
            height: 100vh;
            z-index: 1;
            touch-action: none;
        `;
        
        document.body.appendChild(this.renderer.domElement);
        console.log('🎨 High-quality renderer initialized');
    }
    
    setupCamera() {
        // Cinematic camera setup like GTA 7
        this.camera = new THREE.PerspectiveCamera(
            75, 
            window.innerWidth / window.innerHeight, 
            0.1, 
            1000
        );
        
        // Position camera for side-scrolling LIMBO-style view
        this.camera.position.set(0, 5, 10);
        this.camera.lookAt(0, 0, 0);
        
        console.log('📷 Cinematic camera positioned');
    }
    
    setupLights() {
        // LIMBO-style moody lighting
        this.ambientLight = new THREE.AmbientLight(0x404040, 0.1);
        this.scene.add(this.ambientLight);
        
        // Dramatic directional light
        this.directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
        this.directionalLight.position.set(-10, 10, 5);
        this.directionalLight.castShadow = true;
        
        // High-quality shadows
        this.directionalLight.shadow.mapSize.width = 2048;
        this.directionalLight.shadow.mapSize.height = 2048;
        this.directionalLight.shadow.camera.near = 0.5;
        this.directionalLight.shadow.camera.far = 50;
        
        this.scene.add(this.directionalLight);
        
        console.log('💡 Atmospheric LIMBO lighting setup');
    }
    
    setupPhysics() {
        // Realistic physics world
        this.world = new CANNON.World();
        this.world.gravity.set(0, -15, 0);
        this.world.broadphase = new CANNON.NaiveBroadphase();
        this.world.solver.iterations = 10;
        
        console.log('🌍 Physics world created');
    }
    
    setupPostProcessing() {
        // GTA 7-style post-processing effects
        if (typeof EffectComposer !== 'undefined') {
            this.composer = new EffectComposer(this.renderer);
            
            // Base render pass
            const renderPass = new RenderPass(this.scene, this.camera);
            this.composer.addPass(renderPass);
            
            // Bloom effect for atmospheric glow
            this.bloomPass = new BloomPass(1.5, 25, 5, 256);
            this.composer.addPass(this.bloomPass);
            
            // Film grain for LIMBO atmosphere
            this.filmPass = new FilmPass(0.35, 0.025, 648, false);
            this.composer.addPass(this.filmPass);
            
            console.log('✨ GTA 7-style post-processing enabled');
        }
    }
    
    createPlayer() {
        // LIMBO-style player silhouette
        const playerGeometry = new THREE.BoxGeometry(1, 2, 0.5);
        const playerMaterial = new THREE.MeshLambertMaterial({ 
            color: 0x000000,
            transparent: true,
            opacity: 0.9
        });
        
        this.player = new THREE.Mesh(playerGeometry, playerMaterial);
        this.player.position.set(-8, 1, 0);
        this.player.castShadow = true;
        this.scene.add(this.player);
        
        // Physics body for player
        const playerShape = new CANNON.Box(new CANNON.Vec3(0.5, 1, 0.25));
        this.playerBody = new CANNON.Body({ mass: 1 });
        this.playerBody.addShape(playerShape);
        this.playerBody.position.set(-8, 1, 0);
        this.world.add(this.playerBody);
        
        console.log('👤 LIMBO-style player created');
    }
    
    createWorld() {
        // Create LIMBO-style world with platforms and obstacles
        this.createPlatforms();
        this.createObstacles();
        this.createAtmosphere();
        
        console.log('🌍 LIMBO world created');
    }
    
    createPlatforms() {
        // Ground and platforms
        const groundGeometry = new THREE.PlaneGeometry(50, 1);
        const groundMaterial = new THREE.MeshLambertMaterial({ color: 0x1a1a1a });
        
        const ground = new THREE.Mesh(groundGeometry, groundMaterial);
        ground.rotation.x = -Math.PI / 2;
        ground.position.y = -1;
        ground.receiveShadow = true;
        this.scene.add(ground);
        
        // Physics ground
        const groundShape = new CANNON.Plane();
        const groundBody = new CANNON.Body({ mass: 0 });
        groundBody.addShape(groundShape);
        groundBody.quaternion.setFromAxisAngle(new CANNON.Vec3(1, 0, 0), -Math.PI / 2);
        groundBody.position.set(0, -1, 0);
        this.world.add(groundBody);
        
        // Create platforms
        for (let i = 0; i < 10; i++) {
            const platformGeometry = new THREE.BoxGeometry(3, 0.5, 1);
            const platformMaterial = new THREE.MeshLambertMaterial({ color: 0x2a2a2a });
            
            const platform = new THREE.Mesh(platformGeometry, platformMaterial);
            platform.position.set(i * 8 - 10, Math.random() * 5 + 2, 0);
            platform.castShadow = true;
            platform.receiveShadow = true;
            this.scene.add(platform);
            this.platforms.push(platform);
            
            // Physics platform
            const platformShape = new CANNON.Box(new CANNON.Vec3(1.5, 0.25, 0.5));
            const platformBody = new CANNON.Body({ mass: 0 });
            platformBody.addShape(platformShape);
            platformBody.position.copy(platform.position);
            this.world.add(platformBody);
        }
    }
    
    createObstacles() {
        // Dynamic obstacles with physics
        for (let i = 0; i < 15; i++) {
            const obstacleGeometry = new THREE.CylinderGeometry(0.3, 0.3, 2, 8);
            const obstacleMaterial = new THREE.MeshLambertMaterial({ 
                color: 0x400000,
                transparent: true,
                opacity: 0.8
            });
            
            const obstacle = new THREE.Mesh(obstacleGeometry, obstacleMaterial);
            obstacle.position.set(
                Math.random() * 40 - 10,
                Math.random() * 8 + 3,
                Math.random() * 2 - 1
            );
            obstacle.castShadow = true;
            this.scene.add(obstacle);
            this.obstacles.push(obstacle);
            
            // Physics obstacle
            const obstacleShape = new CANNON.Cylinder(0.3, 0.3, 2, 8);
            const obstacleBody = new CANNON.Body({ mass: 1 });
            obstacleBody.addShape(obstacleShape);
            obstacleBody.position.copy(obstacle.position);
            this.world.add(obstacleBody);
            this.obstaclesBodies.push(obstacleBody);
        }
    }
    
    createAtmosphere() {
        // Particle system for atmospheric effects
        const particleCount = 100;
        const particleGeometry = new THREE.BufferGeometry();
        const particlePositions = new Float32Array(particleCount * 3);
        
        for (let i = 0; i < particleCount * 3; i += 3) {
            particlePositions[i] = Math.random() * 40 - 20;     // x
            particlePositions[i + 1] = Math.random() * 20;      // y  
            particlePositions[i + 2] = Math.random() * 10 - 5;  // z
        }
        
        particleGeometry.setAttribute('position', new THREE.BufferAttribute(particlePositions, 3));
        
        const particleMaterial = new THREE.PointsMaterial({
            color: 0x666666,
            size: 0.1,
            transparent: true,
            opacity: 0.3
        });
        
        const particles = new THREE.Points(particleGeometry, particleMaterial);
        this.scene.add(particles);
        this.particles.push(particles);
        
        console.log('✨ Atmospheric particles added');
    }
    
    setupControls() {
        // Keyboard controls
        window.addEventListener('keydown', (e) => {
            this.keys[e.code] = true;
            if (e.code === 'Space' && this.gameState === 'menu') {
                this.startGame();
            }
        });
        
        window.addEventListener('keyup', (e) => {
            this.keys[e.code] = false;
        });
        
        // Touch controls for mobile
        this.renderer.domElement.addEventListener('touchstart', (e) => {
            e.preventDefault();
            const touch = e.touches[0];
            const rect = this.renderer.domElement.getBoundingClientRect();
            
            this.touch.active = true;
            this.touch.x = ((touch.clientX - rect.left) / rect.width) * 2 - 1;
            this.touch.y = -((touch.clientY - rect.top) / rect.height) * 2 + 1;
            
            if (this.gameState === 'menu') {
                this.startGame();
            }
        }, { passive: false });
        
        this.renderer.domElement.addEventListener('touchmove', (e) => {
            e.preventDefault();
            if (!this.touch.active) return;
            
            const touch = e.touches[0];
            const rect = this.renderer.domElement.getBoundingClientRect();
            
            this.touch.x = ((touch.clientX - rect.left) / rect.width) * 2 - 1;
            this.touch.y = -((touch.clientY - rect.top) / rect.height) * 2 + 1;
        }, { passive: false });
        
        this.renderer.domElement.addEventListener('touchend', (e) => {
            e.preventDefault();
            this.touch.active = false;
        }, { passive: false });
        
        // Window resize
        window.addEventListener('resize', () => {
            this.camera.aspect = window.innerWidth / window.innerHeight;
            this.camera.updateProjectionMatrix();
            this.renderer.setSize(window.innerWidth, window.innerHeight);
            if (this.composer) {
                this.composer.setSize(window.innerWidth, window.innerHeight);
            }
        });
        
        console.log('🎮 Advanced controls setup');
    }
    
    setupUI() {
        // Create UI overlay
        const uiDiv = document.createElement('div');
        uiDiv.id = 'limbo3d-ui';
        uiDiv.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            width: 100vw;
            height: 100vh;
            pointer-events: none;
            font-family: 'SF Pro Display', Arial, sans-serif;
            color: #ffffff;
            z-index: 10;
        `;
        document.body.appendChild(uiDiv);
        
        console.log('🎯 UI overlay created');
    }
    
    startGame() {
        this.gameState = 'playing';
        this.score = 0;
        this.lives = 3;
        
        // Hide splash screen
        const splash = document.getElementById('splash');
        if (splash) splash.style.display = 'none';
        
        console.log('🎮 LIMBO 3D game started!');
    }
    
    update() {
        if (this.gameState !== 'playing') return;
        
        // Update physics
        this.world.step(1/60);
        
        // Player controls
        if (this.keys['ArrowUp'] || this.keys['KeyW'] || this.touch.active) {
            this.playerBody.velocity.y = Math.max(this.playerBody.velocity.y, 8);
        }
        if (this.keys['ArrowLeft'] || this.keys['KeyA']) {
            this.playerBody.velocity.x = -5;
        }
        if (this.keys['ArrowRight'] || this.keys['KeyD']) {
            this.playerBody.velocity.x = 5;
        }
        
        // Touch controls
        if (this.touch.active) {
            this.playerBody.velocity.x += this.touch.x * 3;
        }
        
        // Update player mesh position from physics
        this.player.position.copy(this.playerBody.position);
        this.player.quaternion.copy(this.playerBody.quaternion);
        
        // Update obstacles
        for (let i = 0; i < this.obstacles.length; i++) {
            this.obstacles[i].position.copy(this.obstaclesBodies[i].position);
            this.obstacles[i].quaternion.copy(this.obstaclesBodies[i].quaternion);
        }
        
        // Camera follow player
        this.camera.position.x = this.player.position.x + 5;
        this.camera.lookAt(this.player.position);
        
        // Collision detection
        this.checkCollisions();
        
        // Update score
        this.score += 0.1;
        
        // Animate particles
        this.updateParticles();
    }
    
    checkCollisions() {
        // Check player vs obstacles collision
        const playerPos = this.player.position;
        
        for (let obstacle of this.obstacles) {
            const distance = playerPos.distanceTo(obstacle.position);
            if (distance < 1.5) {
                this.gameOver();
                return;
            }
        }
        
        // Check if player fell
        if (playerPos.y < -10) {
            this.gameOver();
        }
    }
    
    updateParticles() {
        // Animate atmospheric particles
        for (let particle of this.particles) {
            particle.rotation.y += 0.001;
            
            const positions = particle.geometry.attributes.position.array;
            for (let i = 1; i < positions.length; i += 3) {
                positions[i] -= 0.01; // Drift down
                if (positions[i] < 0) {
                    positions[i] = 20; // Reset to top
                }
            }
            particle.geometry.attributes.position.needsUpdate = true;
        }
    }
    
    gameOver() {
        this.gameState = 'gameOver';
        
        // Save high score
        this.saveScore();
        
        // Show game over UI
        setTimeout(() => {
            this.showLeaderboard();
        }, 2000);
        
        console.log('💀 Game Over! Final Score:', Math.floor(this.score));
    }
    
    saveScore() {
        const finalScore = Math.floor(this.score);
        const playerName = this.playerName || 'Anonymous';
        
        // Add to leaderboard
        this.leaderboard.push({
            name: playerName,
            score: finalScore,
            date: new Date().toLocaleDateString()
        });
        
        // Sort and keep top 10
        this.leaderboard.sort((a, b) => b.score - a.score);
        this.leaderboard = this.leaderboard.slice(0, 10);
        
        // Save to localStorage
        localStorage.setItem('limbo3d-leaderboard', JSON.stringify(this.leaderboard));
        
        console.log('💾 Score saved:', finalScore);
    }
    
    loadLeaderboard() {
        try {
            const saved = localStorage.getItem('limbo3d-leaderboard');
            return saved ? JSON.parse(saved) : [];
        } catch (e) {
            return [];
        }
    }
    
    showLeaderboard() {
        // Create leaderboard UI
        const leaderboardUI = document.createElement('div');
        leaderboardUI.style.cssText = `
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: rgba(0, 0, 0, 0.9);
            border: 2px solid #444;
            border-radius: 15px;
            padding: 30px;
            color: #ffffff;
            font-family: 'SF Pro Display', Arial, sans-serif;
            text-align: center;
            z-index: 100;
            min-width: 300px;
        `;
        
        let html = `
            <h2 style="margin-bottom: 20px; color: #ff6b6b;">🏆 LIMBO Leaderboard</h2>
            <p style="margin-bottom: 20px;">Your Score: ${Math.floor(this.score)}</p>
            <div style="margin-bottom: 20px;">
        `;
        
        for (let i = 0; i < Math.min(5, this.leaderboard.length); i++) {
            const entry = this.leaderboard[i];
            html += `
                <div style="margin: 10px 0; padding: 10px; background: rgba(255,255,255,0.1); border-radius: 8px;">
                    ${i + 1}. ${entry.name} - ${entry.score} pts
                </div>
            `;
        }
        
        html += `
            </div>
            <button onclick="location.reload()" style="
                padding: 10px 20px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                border: none;
                border-radius: 25px;
                color: white;
                font-weight: bold;
                cursor: pointer;
                font-size: 16px;
            ">Play Again</button>
        `;
        
        leaderboardUI.innerHTML = html;
        document.body.appendChild(leaderboardUI);
    }
    
    render() {
        if (this.composer) {
            this.composer.render();
        } else {
            this.renderer.render(this.scene, this.camera);
        }
        
        // Render UI
        this.renderUI();
    }
    
    renderUI() {
        const ui = document.getElementById('limbo3d-ui');
        if (!ui) return;
        
        if (this.gameState === 'playing') {
            ui.innerHTML = `
                <div style="position: absolute; top: 20px; left: 20px; font-size: 24px;">
                    Score: ${Math.floor(this.score)}
                </div>
                <div style="position: absolute; top: 20px; right: 20px; font-size: 24px;">
                    Lives: ${this.lives}
                </div>
                <div style="position: absolute; bottom: 20px; left: 20px; font-size: 16px; opacity: 0.7;">
                    WASD/Arrow Keys or Touch to move
                </div>
            `;
        } else if (this.gameState === 'menu') {
            ui.innerHTML = `
                <div style="
                    position: absolute;
                    top: 50%;
                    left: 50%;
                    transform: translate(-50%, -50%);
                    text-align: center;
                    font-size: 32px;
                ">
                    <h1 style="margin-bottom: 30px; text-shadow: 0 0 20px rgba(255,255,255,0.5);">
                        LIMBO 3D
                    </h1>
                    <p style="font-size: 18px; opacity: 0.8;">
                        Press SPACE or Touch to Start
                    </p>
                </div>
            `;
        }
    }
    
    gameLoop() {
        this.update();
        this.render();
        requestAnimationFrame(() => this.gameLoop());
    }
}

// Initialize when page loads
document.addEventListener('DOMContentLoaded', () => {
    console.log('🚀 Starting LIMBO 3D initialization...');
    new Limbo3D();
});
