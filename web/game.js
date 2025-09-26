/**
 * DIMBO - Modern Mobile Game Engine
 * GTA-style sophisticated game with AI backgrounds
 * Mobile-first, touch-responsive design
 */

class DimboGame {
    constructor() {
        this.canvas = null;
        this.ctx = null;
        this.width = 0;
        this.height = 0;
        
        // Game state
        this.gameState = 'splash'; // splash, playing, paused, gameOver, leaderboard
        this.score = 0;
        this.lives = 3;
        this.level = 1;
        
        // Player (LIMBO-style silhouette)
        this.player = {
            x: 60,
            y: 0,
            width: 30,
            height: 30,
            vx: 0,
            vy: 0,
            speed: 4,
            color: '#000000' // Pure black silhouette
        };
        
        // Obstacles
        this.obstacles = [];
        this.obstacleSpeed = 2;
        this.spawnRate = 60;
        this.frameCount = 0;
        
        // Mobile controls
        this.touch = {
            active: false,
            x: 0,
            y: 0,
            startX: 0,
            startY: 0
        };
        
        // AI Background system
        this.backgroundImage = null;
        this.backgroundUrl = null;
        this.isLoadingBackground = false;
        
        // UI elements (GTA-style)
        this.ui = {
            miniMap: { x: 20, y: 20, size: 120 },
            healthBar: { x: 20, y: 160, width: 200, height: 8 },
            scoreDisplay: { x: 20, y: 180 },
            menuButton: { x: 0, y: 0, size: 60 } // Positioned dynamically
        };
        
        // Particle system for effects
        this.particles = [];
        
        // Leaderboard (localStorage for mobile)
        this.leaderboard = this.loadLeaderboard();
        
        this.init();
    }
    
    init() {
        this.setupCanvas();
        this.setupEventListeners();
        this.resize();
        this.loadAIBackground();
        this.gameLoop();
    }
    
    setupCanvas() {
        console.log('🎨 Setting up canvas...');
        
        this.canvas = document.getElementById('gameCanvas');
        if (!this.canvas) {
            // Create canvas if it doesn't exist
            console.log('📐 Creating new canvas...');
            this.canvas = document.createElement('canvas');
            this.canvas.id = 'gameCanvas';
            this.canvas.style.cssText = `
                position: fixed;
                top: 0;
                left: 0;
                width: 100vw;
                height: 100vh;
                background: #0a0a0a;
                image-rendering: pixelated;
                touch-action: none;
                user-select: none;
                z-index: 10;
                display: none;
            `;
            document.body.appendChild(this.canvas);
            console.log('✅ Canvas created and added to DOM');
        }
        
        this.ctx = this.canvas.getContext('2d');
        if (!this.ctx) {
            console.error('❌ Failed to get 2D context!');
            return;
        }
        
        this.ctx.imageSmoothingEnabled = false; // Pixel-perfect rendering
        console.log('✅ Canvas context ready');
    }
    
    setupEventListeners() {
        // Resize handler
        window.addEventListener('resize', () => this.resize());
        
        // Mobile touch controls
        this.canvas.addEventListener('touchstart', (e) => this.handleTouchStart(e), { passive: false });
        this.canvas.addEventListener('touchmove', (e) => this.handleTouchMove(e), { passive: false });
        this.canvas.addEventListener('touchend', (e) => this.handleTouchEnd(e), { passive: false });
        
        // Desktop controls (fallback)
        window.addEventListener('keydown', (e) => this.handleKeyDown(e));
        this.canvas.addEventListener('click', (e) => this.handleClick(e));
        
        // Prevent context menu on mobile
        this.canvas.addEventListener('contextmenu', (e) => e.preventDefault());
    }
    
    resize() {
        const dpr = window.devicePixelRatio || 1;
        this.width = window.innerWidth;
        this.height = window.innerHeight;
        
        // Set canvas size for high-DPI displays
        this.canvas.width = this.width * dpr;
        this.canvas.height = this.height * dpr;
        this.canvas.style.width = this.width + 'px';
        this.canvas.style.height = this.height + 'px';
        
        this.ctx.scale(dpr, dpr);
        
        // Update player position to center
        this.player.y = this.height / 2 - this.player.height / 2;
        
        // Update UI positions for mobile
        this.updateUIPositions();
    }
    
    updateUIPositions() {
        // GTA-style UI positioning that works on all screen sizes
        const margin = Math.min(20, this.width * 0.03);
        
        this.ui.miniMap.x = margin;
        this.ui.miniMap.y = margin;
        this.ui.miniMap.size = Math.min(120, this.width * 0.25);
        
        this.ui.menuButton.x = this.width - 60 - margin;
        this.ui.menuButton.y = margin;
        
        this.ui.healthBar.x = margin;
        this.ui.healthBar.y = this.ui.miniMap.y + this.ui.miniMap.size + margin;
        this.ui.healthBar.width = Math.min(200, this.width * 0.4);
    }
    
    // AI Background loading with Replicate API (with graceful fallback)
    async loadAIBackground() {
        if (this.isLoadingBackground) return;
        this.isLoadingBackground = true;
        
        console.log('🎨 Loading background...');
        
        // Always use fallback for now (simple server testing)
        // TODO: Enable AI backgrounds when full server is running
        this.createFallbackBackground();
        this.isLoadingBackground = false;
        
        /* AI Background code (disabled for simple server testing)
        try {
            const response = await fetch('/api/background', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    prompt: "dark atmospheric forest silhouette, LIMBO game style",
                    width: Math.min(1600, this.width * 2),
                    height: Math.min(900, this.height * 2)
                })
            });
            
            if (response.ok) {
                const data = await response.json();
                this.backgroundUrl = data.url;
                await this.loadBackgroundImage();
            } else {
                throw new Error('API not available');
            }
        } catch (error) {
            console.log('AI background failed, using gradient fallback');
            this.createFallbackBackground();
        } finally {
            this.isLoadingBackground = false;
        }
        */
    }
    
    async loadBackgroundImage() {
        if (!this.backgroundUrl) return;
        
        return new Promise((resolve) => {
            const img = new Image();
            img.crossOrigin = 'anonymous';
            img.onload = () => {
                this.backgroundImage = img;
                resolve();
            };
            img.onerror = () => {
                this.createFallbackBackground();
                resolve();
            };
            img.src = this.backgroundUrl;
        });
    }
    
    createFallbackBackground() {
        // Create LIMBO-style gradient background
        const canvas = document.createElement('canvas');
        canvas.width = this.width;
        canvas.height = this.height;
        const ctx = canvas.getContext('2d');
        
        // Dark gradient
        const gradient = ctx.createLinearGradient(0, 0, 0, this.height);
        gradient.addColorStop(0, '#1a1a1a');
        gradient.addColorStop(0.7, '#0a0a0a');
        gradient.addColorStop(1, '#000000');
        
        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, this.width, this.height);
        
        // Add some atmospheric elements
        ctx.fillStyle = 'rgba(20, 20, 20, 0.8)';
        for (let i = 0; i < 5; i++) {
            const x = (i / 4) * this.width;
            const height = this.height * (0.6 + i * 0.08);
            const width = this.width / 4 + Math.random() * 100;
            ctx.fillRect(x, height, width, this.height - height);
        }
        
        this.backgroundImage = canvas;
    }
    
    // Mobile touch controls
    handleTouchStart(e) {
        e.preventDefault();
        const touch = e.touches[0];
        const rect = this.canvas.getBoundingClientRect();
        
        this.touch.active = true;
        this.touch.startX = this.touch.x = touch.clientX - rect.left;
        this.touch.startY = this.touch.y = touch.clientY - rect.top;
        
        if (this.gameState === 'splash') {
            this.startGame();
        }
    }
    
    handleTouchMove(e) {
        e.preventDefault();
        if (!this.touch.active || this.gameState !== 'playing') return;
        
        const touch = e.touches[0];
        const rect = this.canvas.getBoundingClientRect();
        
        this.touch.x = touch.clientX - rect.left;
        this.touch.y = touch.clientY - rect.top;
        
        // Move player toward touch position (intuitive mobile control)
        const targetX = this.touch.x - this.player.width / 2;
        const targetY = this.touch.y - this.player.height / 2;
        
        const dx = targetX - this.player.x;
        const dy = targetY - this.player.y;
        
        this.player.vx = dx * 0.1; // Smooth movement
        this.player.vy = dy * 0.1;
    }
    
    handleTouchEnd(e) {
        e.preventDefault();
        this.touch.active = false;
        this.player.vx *= 0.8; // Gradual slowdown
        this.player.vy *= 0.8;
    }
    
    // Desktop controls
    handleKeyDown(e) {
        if (this.gameState === 'splash' && e.code === 'Space') {
            this.startGame();
        }
        
        if (this.gameState === 'playing') {
            switch (e.code) {
                case 'ArrowUp':
                case 'KeyW':
                    this.player.vy = -this.player.speed;
                    break;
                case 'ArrowDown':
                case 'KeyS':
                    this.player.vy = this.player.speed;
                    break;
                case 'ArrowLeft':
                case 'KeyA':
                    this.player.vx = -this.player.speed;
                    break;
                case 'ArrowRight':
                case 'KeyD':
                    this.player.vx = this.player.speed;
                    break;
            }
        }
    }
    
    handleClick(e) {
        if (this.gameState === 'splash') {
            this.startGame();
        }
    }
    
    startGame() {
        console.log('🎮 Starting DIMBO game...');
        
        this.gameState = 'playing';
        this.score = 0;
        this.lives = 3;
        this.obstacles = [];
        this.frameCount = 0;
        
        // Hide splash screen
        const splash = document.getElementById('splash');
        if (splash) {
            splash.style.display = 'none';
            console.log('✅ Splash hidden');
        }
        
        const root = document.getElementById('root');
        if (root) root.style.display = 'none';
        
        // Make sure canvas is visible and working
        if (this.canvas) {
            this.canvas.style.display = 'block';
            console.log('✅ Canvas activated');
            console.log('📐 Canvas size:', this.canvas.width, 'x', this.canvas.height);
        } else {
            console.error('❌ Canvas not found!');
        }
    }
    
    update() {
        if (this.gameState !== 'playing') return;
        
        this.frameCount++;
        
        // Update player
        this.player.x += this.player.vx;
        this.player.y += this.player.vy;
        
        // Keep player in bounds
        this.player.x = Math.max(0, Math.min(this.width - this.player.width, this.player.x));
        this.player.y = Math.max(0, Math.min(this.height - this.player.height, this.player.y));
        
        // Spawn obstacles
        if (this.frameCount % this.spawnRate === 0) {
            this.obstacles.push({
                x: this.width,
                y: Math.random() * (this.height - 40),
                width: 20 + Math.random() * 30,
                height: 20 + Math.random() * 30,
                vx: -this.obstacleSpeed - Math.random() * 2
            });
        }
        
        // Update obstacles
        this.obstacles = this.obstacles.filter(obstacle => {
            obstacle.x += obstacle.vx;
            
            // Check collision
            if (this.checkCollision(this.player, obstacle)) {
                this.lives--;
                this.addExplosionParticles(this.player.x, this.player.y);
                
                if (this.lives <= 0) {
                    this.gameOver();
                }
                return false; // Remove obstacle
            }
            
            // Remove off-screen obstacles and award points
            if (obstacle.x + obstacle.width < 0) {
                this.score += 10;
                return false;
            }
            
            return true;
        });
        
        // Update particles
        this.updateParticles();
        
        // Increase difficulty
        if (this.frameCount % 600 === 0) {
            this.obstacleSpeed += 0.2;
            this.spawnRate = Math.max(30, this.spawnRate - 2);
        }
    }
    
    checkCollision(a, b) {
        return a.x < b.x + b.width &&
               a.x + a.width > b.x &&
               a.y < b.y + b.height &&
               a.y + a.height > b.y;
    }
    
    addExplosionParticles(x, y) {
        for (let i = 0; i < 10; i++) {
            this.particles.push({
                x: x + Math.random() * 30,
                y: y + Math.random() * 30,
                vx: (Math.random() - 0.5) * 10,
                vy: (Math.random() - 0.5) * 10,
                life: 1.0,
                decay: 0.02,
                size: 2 + Math.random() * 3
            });
        }
    }
    
    updateParticles() {
        this.particles = this.particles.filter(particle => {
            particle.x += particle.vx;
            particle.y += particle.vy;
            particle.life -= particle.decay;
            particle.vx *= 0.98;
            particle.vy *= 0.98;
            
            return particle.life > 0;
        });
    }
    
    gameOver() {
        this.gameState = 'gameOver';
        this.saveScore();
    }
    
    saveScore() {
        const name = prompt('Enter your name:') || 'Player';
        this.leaderboard.push({ name, score: this.score, date: new Date().toLocaleDateString() });
        this.leaderboard.sort((a, b) => b.score - a.score);
        this.leaderboard = this.leaderboard.slice(0, 10);
        
        localStorage.setItem('dimbo_leaderboard', JSON.stringify(this.leaderboard));
    }
    
    loadLeaderboard() {
        try {
            return JSON.parse(localStorage.getItem('dimbo_leaderboard')) || [];
        } catch {
            return [];
        }
    }
    
    draw() {
        // Clear canvas
        this.ctx.fillStyle = '#0a0a0a';
        this.ctx.fillRect(0, 0, this.width, this.height);
        
        // Draw AI background
        if (this.backgroundImage) {
            this.ctx.save();
            this.ctx.globalAlpha = 0.6;
            this.ctx.drawImage(this.backgroundImage, 0, 0, this.width, this.height);
            this.ctx.restore();
        }
        
        if (this.gameState === 'playing') {
            this.drawGame();
            this.drawGTAUI();
        } else if (this.gameState === 'gameOver') {
            this.drawGameOver();
        }
    }
    
    drawGame() {
        // Draw player (LIMBO-style silhouette)
        this.ctx.fillStyle = this.player.color;
        this.ctx.fillRect(this.player.x, this.player.y, this.player.width, this.player.height);
        
        // Player glow for visibility
        this.ctx.shadowColor = 'rgba(255,255,255,0.3)';
        this.ctx.shadowBlur = 10;
        this.ctx.fillRect(this.player.x, this.player.y, this.player.width, this.player.height);
        this.ctx.shadowBlur = 0;
        
        // Draw obstacles
        this.ctx.fillStyle = '#2a1515';
        this.obstacles.forEach(obstacle => {
            this.ctx.fillRect(obstacle.x, obstacle.y, obstacle.width, obstacle.height);
            
            // Danger glow
            this.ctx.shadowColor = 'rgba(255,0,0,0.2)';
            this.ctx.shadowBlur = 5;
            this.ctx.fillRect(obstacle.x, obstacle.y, obstacle.width, obstacle.height);
            this.ctx.shadowBlur = 0;
        });
        
        // Draw particles
        this.particles.forEach(particle => {
            this.ctx.save();
            this.ctx.globalAlpha = particle.life;
            this.ctx.fillStyle = '#ffaaaa';
            this.ctx.fillRect(particle.x, particle.y, particle.size, particle.size);
            this.ctx.restore();
        });
    }
    
    drawGTAUI() {
        // GTA-style UI with mobile-friendly sizing
        const fontSize = Math.max(16, this.width * 0.03);
        this.ctx.font = `${fontSize}px Arial, sans-serif`;
        this.ctx.fillStyle = 'rgba(0,0,0,0.7)';
        
        // Mini-map background
        const miniMap = this.ui.miniMap;
        this.ctx.fillRect(miniMap.x - 5, miniMap.y - 5, miniMap.size + 10, miniMap.size + 10);
        
        // Mini-map
        this.ctx.fillStyle = 'rgba(20,40,20,0.8)';
        this.ctx.fillRect(miniMap.x, miniMap.y, miniMap.size, miniMap.size);
        
        // Player dot on mini-map
        const playerDotX = miniMap.x + (this.player.x / this.width) * miniMap.size;
        const playerDotY = miniMap.y + (this.player.y / this.height) * miniMap.size;
        this.ctx.fillStyle = '#00ff00';
        this.ctx.fillRect(playerDotX - 2, playerDotY - 2, 4, 4);
        
        // Health bar
        const healthBar = this.ui.healthBar;
        this.ctx.fillStyle = 'rgba(0,0,0,0.7)';
        this.ctx.fillRect(healthBar.x - 2, healthBar.y - 2, healthBar.width + 4, healthBar.height + 4);
        
        this.ctx.fillStyle = '#333';
        this.ctx.fillRect(healthBar.x, healthBar.y, healthBar.width, healthBar.height);
        
        const healthWidth = (this.lives / 3) * healthBar.width;
        this.ctx.fillStyle = this.lives > 1 ? '#00ff00' : '#ff3333';
        this.ctx.fillRect(healthBar.x, healthBar.y, healthWidth, healthBar.height);
        
        // Score
        this.ctx.fillStyle = '#ffffff';
        this.ctx.fillText(`Score: ${this.score}`, this.ui.scoreDisplay.x, healthBar.y + healthBar.height + 25);
        
        // Touch hint for mobile
        if ('ontouchstart' in window && this.touch.active) {
            this.ctx.save();
            this.ctx.globalAlpha = 0.3;
            this.ctx.fillStyle = '#ffffff';
            this.ctx.beginPath();
            this.ctx.arc(this.touch.x, this.touch.y, 30, 0, Math.PI * 2);
            this.ctx.fill();
            this.ctx.restore();
        }
    }
    
    drawGameOver() {
        // Dark overlay
        this.ctx.fillStyle = 'rgba(0,0,0,0.8)';
        this.ctx.fillRect(0, 0, this.width, this.height);
        
        // Game over text
        const fontSize = Math.max(24, this.width * 0.06);
        this.ctx.font = `${fontSize}px Arial, sans-serif`;
        this.ctx.fillStyle = '#ffffff';
        this.ctx.textAlign = 'center';
        
        this.ctx.fillText('Game Over', this.width / 2, this.height / 2 - 50);
        this.ctx.fillText(`Final Score: ${this.score}`, this.width / 2, this.height / 2);
        
        // Restart hint
        const hintFontSize = Math.max(16, this.width * 0.03);
        this.ctx.font = `${hintFontSize}px Arial, sans-serif`;
        this.ctx.fillText('Tap to restart', this.width / 2, this.height / 2 + 50);
        
        this.ctx.textAlign = 'start';
        
        // Handle restart
        if (this.touch.active || this.frameCount % 60 === 0) {
            // Allow restart after a brief delay
            setTimeout(() => {
                this.gameState = 'splash';
            }, 1000);
        }
    }
    
    gameLoop() {
        this.update();
        this.draw();
        requestAnimationFrame(() => this.gameLoop());
    }
}

// Initialize game when page loads
document.addEventListener('DOMContentLoaded', () => {
    new DimboGame();
});
