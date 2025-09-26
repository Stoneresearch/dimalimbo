/**
 * DIMBO AI Background API
 * Replicate + Black Forest Labs Integration
 * Mobile-optimized background generation
 */

const express = require('express');
const cors = require('cors');
const app = express();

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static('.'));

// Replicate configuration
const REPLICATE_API_TOKEN = process.env.REPLICATE_API_TOKEN;

// Black Forest Labs model for LIMBO-style backgrounds
const FLUX_MODEL = "black-forest-labs/flux-schnell";

// Background generation endpoint
app.post('/api/background', async (req, res) => {
    try {
        const { prompt, width = 1024, height = 576 } = req.body;
        
        console.log('🎨 Generating AI background:', prompt);
        
        if (!REPLICATE_API_TOKEN) {
            console.log('⚠️  No Replicate token - using fallback');
            return res.json({
                url: null,
                fallback: true,
                message: 'Using gradient fallback'
            });
        }
        
        // Enhanced LIMBO-style prompt
        const enhancedPrompt = `${prompt}, dark atmospheric game background, high contrast silhouettes, moody lighting, cinematic composition, 4k game art, professional game background`;
        
        // Call Replicate API
        const response = await fetch('https://api.replicate.com/v1/predictions', {
            method: 'POST',
            headers: {
                'Authorization': `Token ${REPLICATE_API_TOKEN}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                version: await getLatestFluxVersion(),
                input: {
                    prompt: enhancedPrompt,
                    width: Math.min(width, 1024),
                    height: Math.min(height, 576),
                    num_inference_steps: 4, // Fast generation for mobile
                    guidance_scale: 7.5,
                    seed: Math.floor(Math.random() * 1000000)
                }
            })
        });
        
        if (!response.ok) {
            throw new Error(`Replicate API error: ${response.status}`);
        }
        
        const prediction = await response.json();
        
        // Poll for completion (with timeout for mobile)
        const imageUrl = await pollForCompletion(prediction.id, 30000); // 30s timeout
        
        res.json({
            url: imageUrl,
            fallback: false,
            prompt: enhancedPrompt
        });
        
    } catch (error) {
        console.error('❌ Background generation failed:', error);
        res.json({
            url: null,
            fallback: true,
            error: error.message
        });
    }
});

// Get latest FLUX model version
async function getLatestFluxVersion() {
    try {
        const response = await fetch(`https://api.replicate.com/v1/models/${FLUX_MODEL}`, {
            headers: {
                'Authorization': `Token ${REPLICATE_API_TOKEN}`
            }
        });
        
        if (response.ok) {
            const model = await response.json();
            return model.latest_version.id;
        }
    } catch (error) {
        console.log('Using fallback model version');
    }
    
    // Fallback version if API call fails
    return "f2ab8a5569070ad0648132b7e88c73db9fd9e1b81e87a12b1d13fecfdb3cceee";
}

// Poll Replicate for completion
async function pollForCompletion(predictionId, timeoutMs = 30000) {
    const startTime = Date.now();
    const pollInterval = 2000; // 2 seconds
    
    while (Date.now() - startTime < timeoutMs) {
        try {
            const response = await fetch(`https://api.replicate.com/v1/predictions/${predictionId}`, {
                headers: {
                    'Authorization': `Token ${REPLICATE_API_TOKEN}`
                }
            });
            
            if (!response.ok) continue;
            
            const prediction = await response.json();
            
            if (prediction.status === 'succeeded' && prediction.output) {
                console.log('✅ Background generated successfully');
                return Array.isArray(prediction.output) ? prediction.output[0] : prediction.output;
            }
            
            if (prediction.status === 'failed') {
                throw new Error('Generation failed');
            }
            
            // Still processing...
            await new Promise(resolve => setTimeout(resolve, pollInterval));
            
        } catch (error) {
            console.log('Polling error:', error.message);
            await new Promise(resolve => setTimeout(resolve, pollInterval));
        }
    }
    
    throw new Error('Generation timeout');
}

// Health check
app.get('/api/health', (req, res) => {
    res.json({
        status: 'ok',
        hasReplicateToken: !!REPLICATE_API_TOKEN,
        timestamp: new Date().toISOString()
    });
});

// Serve game files
app.get('/', (req, res) => {
    res.sendFile(__dirname + '/index.html');
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
    console.log(`
🕷️  DIMBO Server Running on port ${PORT}
🎮 Game: http://localhost:${PORT}
🎨 API: http://localhost:${PORT}/api/health
${REPLICATE_API_TOKEN ? '✅' : '❌'} Replicate API: ${REPLICATE_API_TOKEN ? 'Connected' : 'Missing token'}
    `);
});
