let cameras = [];
let refreshInterval = null;

// Load cameras on page load
window.addEventListener('DOMContentLoaded', () => {
    loadCameras();
    // Refresh every 2 seconds
    refreshInterval = setInterval(loadCameras, 2000);
});

// Clean up on page unload
window.addEventListener('beforeunload', () => {
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
});

// Load all cameras and their status
async function loadCameras() {
    try {
        const response = await fetch('/api/status');
        cameras = await response.json();
        updateStats();
        renderCameras();
    } catch (error) {
        console.error('Failed to load cameras:', error);
    }
}

// Update statistics
function updateStats() {
    const totalCameras = cameras.length;
    const streamingCameras = cameras.filter(c => c.isStreaming).length;
    const totalViewers = cameras.reduce((sum, c) => sum + (c.viewerCount || 0), 0);
    const enabledCameras = cameras.filter(c => c.enabled).length;

    document.getElementById('totalCameras').textContent = totalCameras;
    document.getElementById('streamingCameras').textContent = streamingCameras;
    document.getElementById('totalViewers').textContent = totalViewers;
    document.getElementById('enabledCameras').textContent = enabledCameras;
}

// Render all cameras
function renderCameras() {
    const grid = document.getElementById('cameraGrid');
    
    // Keep existing elements and update them to avoid flickering
    const existingCards = {};
    grid.querySelectorAll('.camera-card').forEach(card => {
        const id = card.dataset.cameraId;
        if (id) {
            existingCards[id] = card;
        }
    });

    cameras.forEach(camera => {
        let card = existingCards[camera.id];
        if (card) {
            // Update existing card
            updateCameraCard(card, camera);
            delete existingCards[camera.id];
        } else {
            // Create new card
            card = createCameraCard(camera);
            grid.appendChild(card);
        }
    });

    // Remove cards for cameras that no longer exist
    Object.values(existingCards).forEach(card => card.remove());
}

// Create a camera card
function createCameraCard(camera) {
    const card = document.createElement('div');
    card.className = `camera-card ${camera.isStreaming ? 'streaming' : ''}`;
    card.dataset.cameraId = camera.id;
    
    card.innerHTML = `
        <div class="camera-header">
            <div>
                <div class="camera-name">${camera.name}</div>
                <div class="camera-id">ID: ${camera.id}</div>
            </div>
            <span class="status-badge ${camera.isStreaming ? 'streaming' : 'offline'}">
                ${camera.isStreaming ? '🟢 推流中' : '⚫ 离线'}
            </span>
        </div>
        <div class="video-preview">
            ${camera.isStreaming ? 
                `<img class="video-stream" src="/stream/${camera.id}" alt="${camera.name}">` :
                `<div class="no-stream">暂无视频流</div>`
            }
        </div>
        <div class="camera-info">
            <div class="info-item">
                <div class="info-label">观看人数</div>
                <div class="info-value viewers">${camera.viewerCount || 0}</div>
            </div>
            <div class="info-item">
                <div class="info-label">状态</div>
                <div class="info-value">${camera.enabled ? '已启用' : '已禁用'}</div>
            </div>
            <div class="info-item">
                <div class="info-label">最后观看</div>
                <div class="info-value">${formatTime(camera.lastViewed)}</div>
            </div>
            <div class="info-item">
                <div class="info-label">ROI区域</div>
                <div class="info-value">${camera.roi ? camera.roi.length : 0} 个</div>
            </div>
        </div>
        <div class="controls">
            ${camera.isStreaming ? 
                `<button class="btn-danger" onclick="stopStream('${camera.id}')">停止推流</button>` :
                `<button class="btn-primary" onclick="startStream('${camera.id}')">开始推流</button>`
            }
            <button class="btn-primary" onclick="window.open('/stream/${camera.id}', '_blank')">查看流</button>
        </div>
    `;
    
    return card;
}

// Update an existing camera card
function updateCameraCard(card, camera) {
    card.className = `camera-card ${camera.isStreaming ? 'streaming' : ''}`;
    
    // Update status badge
    const statusBadge = card.querySelector('.status-badge');
    statusBadge.className = `status-badge ${camera.isStreaming ? 'streaming' : 'offline'}`;
    statusBadge.textContent = camera.isStreaming ? '🟢 推流中' : '⚫ 离线';
    
    // Update video preview
    const videoPreview = card.querySelector('.video-preview');
    if (camera.isStreaming) {
        videoPreview.innerHTML = `<img class="video-stream" src="/stream/${camera.id}" alt="${camera.name}">`;
    } else {
        videoPreview.innerHTML = `<div class="no-stream">暂无视频流</div>`;
    }
    
    // Update info values
    const infoValues = card.querySelectorAll('.info-value');
    infoValues[0].textContent = camera.viewerCount || 0;
    infoValues[1].textContent = camera.enabled ? '已启用' : '已禁用';
    infoValues[2].textContent = formatTime(camera.lastViewed);
    infoValues[3].textContent = camera.roi ? camera.roi.length : 0;

    // Update controls
    const controls = card.querySelector('.controls');
    controls.innerHTML = `
        ${camera.isStreaming ?
            `<button class="btn-danger" onclick="stopStream('${camera.id}')">停止推流</button>` :
            `<button class="btn-primary" onclick="startStream('${camera.id}')">开始推流</button>`
        }
        <button class="btn-primary" onclick="window.open('/stream/${camera.id}', '_blank')">查看流</button>
    `;
}

// Format time
function formatTime(timeStr) {
    if (!timeStr || timeStr === '0001-01-01T00:00:00Z') {
        return '从未';
    }

    const date = new Date(timeStr);
    const now = new Date();
    const diff = Math.floor((now - date) / 1000); // seconds

    if (diff < 60) {
        return `${diff}秒前`;
    } else if (diff < 3600) {
        return `${Math.floor(diff / 60)}分钟前`;
    } else if (diff < 86400) {
        return `${Math.floor(diff / 3600)}小时前`;
    } else {
        return date.toLocaleString('zh-CN');
    }
}

// Start stream
async function startStream(cameraId) {
    try {
        const response = await fetch(`/api/cameras/${cameraId}/start`, {
            method: 'POST'
        });

        if (response.ok) {
            console.log(`Started stream for camera: ${cameraId}`);
            // Refresh immediately
            loadCameras();
        } else {
            const error = await response.text();
            alert(`启动失败: ${error}`);
        }
    } catch (error) {
        console.error('Failed to start stream:', error);
        alert('启动失败: ' + error.message);
    }
}

// Stop stream
async function stopStream(cameraId) {
    try {
        const response = await fetch(`/api/cameras/${cameraId}/stop`, {
            method: 'POST'
        });

        if (response.ok) {
            console.log(`Stopped stream for camera: ${cameraId}`);
            // Refresh immediately
            loadCameras();
        } else {
            const error = await response.text();
            alert(`停止失败: ${error}`);
        }
    } catch (error) {
        console.error('Failed to stop stream:', error);
        alert('停止失败: ' + error.message);
    }
}

