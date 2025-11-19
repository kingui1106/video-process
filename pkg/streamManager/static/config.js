let cameras = [];
let drawingStates = {}; // Track drawing state for each camera

// Load cameras on page load
window.addEventListener('DOMContentLoaded', () => {
    loadCameras();
});

// Load all cameras from API
async function loadCameras() {
    try {
        const response = await fetch('/api/cameras');
        cameras = await response.json();
        renderCameras();
    } catch (error) {
        console.error('Failed to load cameras:', error);
    }
}

// Render all cameras
function renderCameras() {
    const grid = document.getElementById('cameraGrid');
    grid.innerHTML = '';

    cameras.forEach(camera => {
        const card = createCameraCard(camera);
        grid.appendChild(card);
    });
}

// Create a camera card
function createCameraCard(camera) {
    const card = document.createElement('div');
    card.className = 'camera-card';
    card.innerHTML = `
        <div class="camera-header">
            <div>
                <div class="camera-name">${camera.name}</div>
                <div class="camera-id">ID: ${camera.id}</div>
            </div>
            <span class="status ${camera.enabled ? 'online' : 'offline'}">
                ${camera.enabled ? '在线' : '离线'}
            </span>
        </div>
        <div class="video-container" id="container-${camera.id}">
            <img class="video-stream" id="stream-${camera.id}" 
                 src="/stream/${camera.id}" 
                 alt="${camera.name}"
                 crossorigin="anonymous">
            <canvas class="roi-canvas" id="roi-${camera.id}"></canvas>
            <canvas class="draw-canvas" id="draw-${camera.id}"></canvas>
        </div>
        <div class="controls">
            <button class="btn-primary" onclick="startDrawing('${camera.id}')">绘制区域</button>
            <button class="btn-danger" onclick="clearROI('${camera.id}')">清除区域</button>
            <button class="btn-secondary" onclick="saveROI('${camera.id}')">保存ROI</button>
            <button class="btn-secondary" onclick="showEditCameraModal('${camera.id}')">编辑</button>
            <button class="btn-danger" onclick="deleteCamera('${camera.id}')">删除</button>
        </div>
        <div class="roi-list" id="roi-list-${camera.id}"></div>
        <div class="stream-url">
            流地址: <a href="/stream/${camera.id}" target="_blank">/stream/${camera.id}</a>
        </div>
    `;

    // Initialize drawing state
    drawingStates[camera.id] = {
        isDrawing: false,
        startX: 0,
        startY: 0,
        rois: camera.roi || []
    };

    // Setup canvas after card is added to DOM
    setTimeout(() => setupCanvas(camera.id), 100);

    return card;
}

// Setup canvas for drawing
function setupCanvas(cameraId) {
    const img = document.getElementById(`stream-${cameraId}`);
    const roiCanvas = document.getElementById(`roi-${cameraId}`);
    const drawCanvas = document.getElementById(`draw-${cameraId}`);

    if (!img || !roiCanvas || !drawCanvas) return;

    // Set canvas size to match image
    const updateCanvasSize = () => {
        const rect = img.getBoundingClientRect();
        roiCanvas.width = rect.width;
        roiCanvas.height = rect.height;
        drawCanvas.width = rect.width;
        drawCanvas.height = rect.height;
        drawExistingROIs(cameraId);
    };

    img.addEventListener('load', updateCanvasSize);
    window.addEventListener('resize', updateCanvasSize);
    updateCanvasSize();

    // Setup drawing events
    setupDrawingEvents(cameraId);
}

// Setup drawing events
function setupDrawingEvents(cameraId) {
    const drawCanvas = document.getElementById(`draw-${cameraId}`);
    const state = drawingStates[cameraId];

    drawCanvas.addEventListener('mousedown', (e) => {
        if (!state.drawingEnabled) return;
        
        const rect = drawCanvas.getBoundingClientRect();
        state.isDrawing = true;
        state.startX = e.clientX - rect.left;
        state.startY = e.clientY - rect.top;
    });

    drawCanvas.addEventListener('mousemove', (e) => {
        if (!state.isDrawing) return;

        const rect = drawCanvas.getBoundingClientRect();
        const currentX = e.clientX - rect.left;
        const currentY = e.clientY - rect.top;

        const ctx = drawCanvas.getContext('2d');
        ctx.clearRect(0, 0, drawCanvas.width, drawCanvas.height);
        
        ctx.strokeStyle = '#00ff00';
        ctx.lineWidth = 2;
        ctx.strokeRect(
            state.startX,
            state.startY,
            currentX - state.startX,
            currentY - state.startY
        );
    });

    drawCanvas.addEventListener('mouseup', (e) => {
        if (!state.isDrawing) return;

        const rect = drawCanvas.getBoundingClientRect();
        const endX = e.clientX - rect.left;
        const endY = e.clientY - rect.top;

        // Calculate ROI in image coordinates
        const img = document.getElementById(`stream-${cameraId}`);
        const scaleX = img.naturalWidth / rect.width;
        const scaleY = img.naturalHeight / rect.height;

        const roi = {
            x: Math.round(Math.min(state.startX, endX) * scaleX),
            y: Math.round(Math.min(state.startY, endY) * scaleY),
            width: Math.round(Math.abs(endX - state.startX) * scaleX),
            height: Math.round(Math.abs(endY - state.startY) * scaleY)
        };

        if (roi.width > 10 && roi.height > 10) {
            state.rois.push(roi);
            drawExistingROIs(cameraId);
            updateROIList(cameraId);
        }

        state.isDrawing = false;
        const ctx = drawCanvas.getContext('2d');
        ctx.clearRect(0, 0, drawCanvas.width, drawCanvas.height);
    });
}

// Draw existing ROIs
function drawExistingROIs(cameraId) {
    const roiCanvas = document.getElementById(`roi-${cameraId}`);
    const img = document.getElementById(`stream-${cameraId}`);
    const state = drawingStates[cameraId];

    if (!roiCanvas || !img) return;

    const ctx = roiCanvas.getContext('2d');
    ctx.clearRect(0, 0, roiCanvas.width, roiCanvas.height);

    const rect = img.getBoundingClientRect();
    const scaleX = rect.width / img.naturalWidth;
    const scaleY = rect.height / img.naturalHeight;

    ctx.strokeStyle = '#ff0000';
    ctx.lineWidth = 2;

    state.rois.forEach((roi, index) => {
        const x = roi.x * scaleX;
        const y = roi.y * scaleY;
        const width = roi.width * scaleX;
        const height = roi.height * scaleY;

        ctx.strokeRect(x, y, width, height);

        // Draw label
        ctx.fillStyle = '#ff0000';
        ctx.font = '12px Arial';
        ctx.fillText(`ROI ${index + 1}`, x + 5, y + 15);
    });
}

// Update ROI list display
function updateROIList(cameraId) {
    const listDiv = document.getElementById(`roi-list-${cameraId}`);
    const state = drawingStates[cameraId];

    if (!listDiv) return;

    if (state.rois.length === 0) {
        listDiv.innerHTML = '<div style="color: #666;">暂无检测区域</div>';
        return;
    }

    listDiv.innerHTML = '<div style="margin-bottom: 5px; font-weight: bold;">检测区域:</div>';
    state.rois.forEach((roi, index) => {
        const item = document.createElement('div');
        item.className = 'roi-item';
        item.innerHTML = `
            <span>ROI ${index + 1}: (${roi.x}, ${roi.y}) ${roi.width}x${roi.height}</span>
            <button class="btn-danger" onclick="deleteROI('${cameraId}', ${index})">删除</button>
        `;
        listDiv.appendChild(item);
    });
}

// Start drawing mode
function startDrawing(cameraId) {
    const state = drawingStates[cameraId];
    state.drawingEnabled = !state.drawingEnabled;

    const drawCanvas = document.getElementById(`draw-${cameraId}`);
    if (state.drawingEnabled) {
        drawCanvas.style.cursor = 'crosshair';
        alert('请在视频画面上拖动鼠标绘制检测区域');
    } else {
        drawCanvas.style.cursor = 'default';
    }
}

// Clear all ROIs
function clearROI(cameraId) {
    if (!confirm('确定要清除所有检测区域吗？')) return;

    const state = drawingStates[cameraId];
    state.rois = [];
    drawExistingROIs(cameraId);
    updateROIList(cameraId);
}

// Delete a specific ROI
function deleteROI(cameraId, index) {
    const state = drawingStates[cameraId];
    state.rois.splice(index, 1);
    drawExistingROIs(cameraId);
    updateROIList(cameraId);
}

// Save ROI configuration
async function saveROI(cameraId) {
    const state = drawingStates[cameraId];

    try {
        const response = await fetch(`/api/cameras/${cameraId}/roi`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ roi: state.rois })
        });

        if (response.ok) {
            alert('配置保存成功！');
        } else {
            alert('保存失败: ' + await response.text());
        }
    } catch (error) {
        console.error('Failed to save ROI:', error);
        alert('保存失败: ' + error.message);
    }
}

// Camera management functions
let editingCameraId = null;

// Show add camera modal
function showAddCameraModal() {
    editingCameraId = null;
    document.getElementById('modalTitle').textContent = '添加摄像头';
    document.getElementById('cameraForm').reset();
    document.getElementById('cameraId').disabled = false;
    document.getElementById('cameraModal').style.display = 'block';
}

// Show edit camera modal
function showEditCameraModal(cameraId) {
    editingCameraId = cameraId;
    const camera = cameras.find(c => c.id === cameraId);
    if (!camera) return;

    document.getElementById('modalTitle').textContent = '编辑摄像头';
    document.getElementById('cameraId').value = camera.id;
    document.getElementById('cameraId').disabled = true;
    document.getElementById('cameraName').value = camera.name;
    document.getElementById('cameraRtspUrl').value = camera.rtspUrl;
    document.getElementById('cameraEnabled').value = camera.enabled.toString();
    document.getElementById('cameraModal').style.display = 'block';
}

// Close camera modal
function closeCameraModal() {
    document.getElementById('cameraModal').style.display = 'none';
    editingCameraId = null;
}

// Save camera form
async function saveCameraForm(event) {
    event.preventDefault();

    const cameraData = {
        id: document.getElementById('cameraId').value,
        name: document.getElementById('cameraName').value,
        rtspUrl: document.getElementById('cameraRtspUrl').value,
        enabled: document.getElementById('cameraEnabled').value === 'true',
        roi: []
    };

    try {
        let response;
        if (editingCameraId) {
            // Update existing camera
            response = await fetch(`/api/cameras/${editingCameraId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(cameraData)
            });
        } else {
            // Add new camera
            response = await fetch('/api/cameras', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(cameraData)
            });
        }

        if (response.ok) {
            alert(editingCameraId ? '摄像头更新成功！' : '摄像头添加成功！');
            closeCameraModal();
            loadCameras();
        } else {
            const error = await response.text();
            alert('保存失败: ' + error);
        }
    } catch (error) {
        console.error('Failed to save camera:', error);
        alert('保存失败: ' + error.message);
    }
}

// Delete camera
async function deleteCamera(cameraId) {
    const camera = cameras.find(c => c.id === cameraId);
    if (!camera) return;

    if (!confirm(`确定要删除摄像头 "${camera.name}" 吗？`)) {
        return;
    }

    try {
        const response = await fetch(`/api/cameras/${cameraId}`, {
            method: 'DELETE'
        });

        if (response.ok) {
            alert('摄像头删除成功！');
            loadCameras();
        } else {
            const error = await response.text();
            alert('删除失败: ' + error);
        }
    } catch (error) {
        console.error('Failed to delete camera:', error);
        alert('删除失败: ' + error.message);
    }
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modal = document.getElementById('cameraModal');
    if (event.target === modal) {
        closeCameraModal();
    }
}


