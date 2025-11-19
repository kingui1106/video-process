// Camera configuration page script
let cameraId = null;
let camera = null;
let drawingState = {
    isDrawing: false,
    drawingEnabled: false,
    currentTool: 'rectangle',
    currentColor: '#FF0000',
    currentThickness: 2,
    currentText: '',
    fontSize: 13,
    tempPoints: [],
    elements: [],
    startX: 0,
    startY: 0
};

// Tool types
const TOOL_TYPES = {
    RECTANGLE: 'rectangle',
    POLYLINE: 'polyline',
    TEXT: 'text'
};

// Load camera on page load
window.addEventListener('DOMContentLoaded', () => {
    // Get camera ID from URL
    const urlParams = new URLSearchParams(window.location.search);
    cameraId = urlParams.get('id');
    
    if (!cameraId) {
        alert('未指定摄像头ID');
        window.location.href = '/config';
        return;
    }
    
    loadCamera();
    setupToolbar();
});

// Load camera data
async function loadCamera() {
    try {
        const response = await fetch('/api/cameras');
        const cameras = await response.json();
        camera = cameras.find(c => c.id === cameraId);
        
        if (!camera) {
            alert('摄像头不存在');
            window.location.href = '/config';
            return;
        }
        
        // Load existing elements
        drawingState.elements = camera.drawElements || [];
        
        renderCameraInfo();
        setupVideo();
        updateElementList();
    } catch (error) {
        console.error('Failed to load camera:', error);
        alert('加载摄像头失败: ' + error.message);
    }
}

// Render camera info
function renderCameraInfo() {
    const infoDiv = document.getElementById('cameraInfo');
    infoDiv.innerHTML = `
        <div class="camera-header">
            <div>
                <div class="camera-name">${camera.name}</div>
                <div class="camera-id">ID: ${camera.id} | RTSP: ${camera.rtspUrl}</div>
            </div>
            <span class="status ${camera.enabled ? 'online' : 'offline'}">
                ${camera.enabled ? '启用' : '禁用'}
            </span>
        </div>
    `;
}

// Setup video stream
function setupVideo() {
    const img = document.getElementById('videoStream');
    const roiCanvas = document.getElementById('roiCanvas');
    const drawCanvas = document.getElementById('drawCanvas');
    
    img.src = `/stream/${cameraId}`;
    
    // Set canvas size to match image
    const updateCanvasSize = () => {
        const rect = img.getBoundingClientRect();
        roiCanvas.width = rect.width;
        roiCanvas.height = rect.height;
        drawCanvas.width = rect.width;
        drawCanvas.height = rect.height;
        renderElements();
    };
    
    img.addEventListener('load', updateCanvasSize);
    window.addEventListener('resize', updateCanvasSize);
    
    // Initial size update
    setTimeout(updateCanvasSize, 100);
    
    // Setup drawing events
    setupDrawingEvents();
}

// Setup toolbar
function setupToolbar() {
    const toolSelect = document.getElementById('toolSelect');
    const colorPicker = document.getElementById('colorPicker');
    const thicknessInput = document.getElementById('thicknessInput');
    const textInput = document.getElementById('textInput');
    const fontSizeSelect = document.getElementById('fontSizeSelect');
    
    toolSelect.addEventListener('change', () => {
        drawingState.currentTool = toolSelect.value;
        updateToolVisibility();
    });
    
    colorPicker.addEventListener('change', () => {
        drawingState.currentColor = colorPicker.value;
    });
    
    thicknessInput.addEventListener('change', () => {
        drawingState.currentThickness = parseInt(thicknessInput.value);
    });
    
    textInput.addEventListener('change', () => {
        drawingState.currentText = textInput.value;
    });
    
    fontSizeSelect.addEventListener('change', () => {
        drawingState.fontSize = parseInt(fontSizeSelect.value);
    });
    
    updateToolVisibility();
}

// Update tool visibility based on selected tool
function updateToolVisibility() {
    const textInputGroup = document.getElementById('textInputGroup');
    const fontSizeGroup = document.getElementById('fontSizeGroup');

    if (drawingState.currentTool === TOOL_TYPES.TEXT) {
        textInputGroup.style.display = 'flex';
        fontSizeGroup.style.display = 'flex';
    } else {
        textInputGroup.style.display = 'none';
        fontSizeGroup.style.display = 'none';
    }
}

// Setup drawing events
function setupDrawingEvents() {
    const drawCanvas = document.getElementById('drawCanvas');

    drawCanvas.addEventListener('mousedown', (e) => {
        if (!drawingState.drawingEnabled) return;

        const rect = drawCanvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;

        if (drawingState.currentTool === TOOL_TYPES.RECTANGLE) {
            drawingState.isDrawing = true;
            drawingState.startX = x;
            drawingState.startY = y;
        } else if (drawingState.currentTool === TOOL_TYPES.POLYLINE) {
            // Add point to polyline
            drawingState.tempPoints.push({x, y});
            renderTempDrawing();
        } else if (drawingState.currentTool === TOOL_TYPES.TEXT) {
            // Place text at click position
            const text = drawingState.currentText.trim();
            if (text) {
                addTextElement(x, y, text);
                document.getElementById('textInput').value = '';
                drawingState.currentText = '';
            } else {
                alert('请先输入文字内容');
            }
        }
    });

    drawCanvas.addEventListener('mousemove', (e) => {
        const rect = drawCanvas.getBoundingClientRect();
        const currentX = e.clientX - rect.left;
        const currentY = e.clientY - rect.top;

        if (drawingState.currentTool === TOOL_TYPES.RECTANGLE && drawingState.isDrawing) {
            renderTempDrawing(currentX, currentY);
        } else if (drawingState.currentTool === TOOL_TYPES.POLYLINE && drawingState.tempPoints.length > 0) {
            renderTempDrawing(currentX, currentY);
        }
    });

    drawCanvas.addEventListener('mouseup', (e) => {
        if (!drawingState.isDrawing) return;

        const rect = drawCanvas.getBoundingClientRect();
        const endX = e.clientX - rect.left;
        const endY = e.clientY - rect.top;

        if (drawingState.currentTool === TOOL_TYPES.RECTANGLE) {
            addRectangleElement(drawingState.startX, drawingState.startY, endX, endY);
            drawingState.isDrawing = false;
        }
    });

    // Double click to finish polyline
    drawCanvas.addEventListener('dblclick', (e) => {
        if (drawingState.currentTool === TOOL_TYPES.POLYLINE && drawingState.tempPoints.length > 1) {
            addPolylineElement();
        }
    });
}

// Add rectangle element
function addRectangleElement(x1, y1, x2, y2) {
    const img = document.getElementById('videoStream');
    const canvas = document.getElementById('drawCanvas');

    // Convert canvas coordinates to image coordinates
    const scaleX = img.naturalWidth / canvas.width;
    const scaleY = img.naturalHeight / canvas.height;

    const element = {
        type: 'rectangle',
        points: [
            {x: Math.round(x1 * scaleX), y: Math.round(y1 * scaleY)},
            {x: Math.round(x2 * scaleX), y: Math.round(y2 * scaleY)}
        ],
        color: drawingState.currentColor,
        thickness: drawingState.currentThickness
    };

    drawingState.elements.push(element);
    renderElements();
    updateElementList();
}

// Add polyline element
function addPolylineElement() {
    const img = document.getElementById('videoStream');
    const canvas = document.getElementById('drawCanvas');

    // Convert canvas coordinates to image coordinates
    const scaleX = img.naturalWidth / canvas.width;
    const scaleY = img.naturalHeight / canvas.height;

    const points = drawingState.tempPoints.map(p => ({
        x: Math.round(p.x * scaleX),
        y: Math.round(p.y * scaleY)
    }));

    const element = {
        type: 'polyline',
        points: points,
        color: drawingState.currentColor,
        thickness: drawingState.currentThickness
    };

    drawingState.elements.push(element);
    drawingState.tempPoints = [];
    renderElements();
    updateElementList();
}

// Add text element
function addTextElement(x, y, text) {
    const img = document.getElementById('videoStream');
    const canvas = document.getElementById('drawCanvas');

    // Convert canvas coordinates to image coordinates
    const scaleX = img.naturalWidth / canvas.width;
    const scaleY = img.naturalHeight / canvas.height;

    const element = {
        type: 'text',
        points: [{x: Math.round(x * scaleX), y: Math.round(y * scaleY)}],
        text: text,
        color: drawingState.currentColor,
        thickness: drawingState.currentThickness,
        fontSize: drawingState.fontSize
    };

    drawingState.elements.push(element);
    renderElements();
    updateElementList();
}

// Render all elements
function renderElements() {
    const canvas = document.getElementById('roiCanvas');
    const ctx = canvas.getContext('2d');
    const img = document.getElementById('videoStream');

    if (!img.naturalWidth || !img.naturalHeight) return;

    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // Calculate scale
    const scaleX = canvas.width / img.naturalWidth;
    const scaleY = canvas.height / img.naturalHeight;

    // Draw each element
    drawingState.elements.forEach(elem => {
        ctx.strokeStyle = elem.color || '#FF0000';
        ctx.lineWidth = elem.thickness || 2;
        ctx.fillStyle = elem.color || '#FF0000';

        if (elem.type === 'rectangle' && elem.points.length >= 2) {
            const x1 = elem.points[0].x * scaleX;
            const y1 = elem.points[0].y * scaleY;
            const x2 = elem.points[1].x * scaleX;
            const y2 = elem.points[1].y * scaleY;
            ctx.strokeRect(x1, y1, x2 - x1, y2 - y1);
        } else if (elem.type === 'polyline' && elem.points.length > 1) {
            ctx.beginPath();
            ctx.moveTo(elem.points[0].x * scaleX, elem.points[0].y * scaleY);
            for (let i = 1; i < elem.points.length; i++) {
                ctx.lineTo(elem.points[i].x * scaleX, elem.points[i].y * scaleY);
            }
            ctx.stroke();
        } else if (elem.type === 'text' && elem.points.length > 0) {
            ctx.font = `${elem.fontSize || 13}px Arial`;
            ctx.fillText(elem.text || '', elem.points[0].x * scaleX, elem.points[0].y * scaleY);
        }
    });
}

// Render temporary drawing
function renderTempDrawing(currentX, currentY) {
    const canvas = document.getElementById('drawCanvas');
    const ctx = canvas.getContext('2d');

    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    ctx.strokeStyle = drawingState.currentColor;
    ctx.lineWidth = drawingState.currentThickness;

    if (drawingState.currentTool === TOOL_TYPES.RECTANGLE && drawingState.isDrawing) {
        const width = currentX - drawingState.startX;
        const height = currentY - drawingState.startY;
        ctx.strokeRect(drawingState.startX, drawingState.startY, width, height);
    } else if (drawingState.currentTool === TOOL_TYPES.POLYLINE && drawingState.tempPoints.length > 0) {
        ctx.beginPath();
        ctx.moveTo(drawingState.tempPoints[0].x, drawingState.tempPoints[0].y);
        for (let i = 1; i < drawingState.tempPoints.length; i++) {
            ctx.lineTo(drawingState.tempPoints[i].x, drawingState.tempPoints[i].y);
        }
        if (currentX !== undefined && currentY !== undefined) {
            ctx.lineTo(currentX, currentY);
        }
        ctx.stroke();
    }
}

// Update element list
function updateElementList() {
    const listDiv = document.getElementById('elementList');

    if (drawingState.elements.length === 0) {
        listDiv.innerHTML = '<div style="color: #666; padding: 10px;">暂无绘制元素</div>';
        return;
    }

    listDiv.innerHTML = '';
    drawingState.elements.forEach((elem, index) => {
        const item = document.createElement('div');
        item.className = 'element-item';

        let info = '';
        if (elem.type === 'rectangle') {
            info = `矩形 (${elem.points[0].x},${elem.points[0].y}) → (${elem.points[1].x},${elem.points[1].y})`;
        } else if (elem.type === 'polyline') {
            info = `折线 (${elem.points.length}个点)`;
        } else if (elem.type === 'text') {
            info = `文字 "${elem.text}" (${elem.points[0].x},${elem.points[0].y})`;
        }

        item.innerHTML = `
            <div class="element-info">
                <span class="element-type">${elem.type}</span>
                ${info}
            </div>
            <button class="btn-danger" onclick="deleteElement(${index})">删除</button>
        `;

        listDiv.appendChild(item);
    });
}

// Start drawing
function startDrawing() {
    drawingState.drawingEnabled = !drawingState.drawingEnabled;
    const btn = event.target;

    if (drawingState.drawingEnabled) {
        btn.textContent = '停止绘制';
        btn.classList.remove('btn-primary');
        btn.classList.add('btn-danger');
    } else {
        btn.textContent = '开始绘制';
        btn.classList.remove('btn-danger');
        btn.classList.add('btn-primary');
        // Clear temp drawing
        const canvas = document.getElementById('drawCanvas');
        const ctx = canvas.getContext('2d');
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        drawingState.tempPoints = [];
    }
}

// Clear all elements
function clearElements() {
    if (!confirm('确定要清除所有绘制元素吗？')) {
        return;
    }

    drawingState.elements = [];
    renderElements();
    updateElementList();
}

// Delete single element
function deleteElement(index) {
    drawingState.elements.splice(index, 1);
    renderElements();
    updateElementList();
}

// Save elements
async function saveElements() {
    try {
        const response = await fetch(`/api/cameras/${cameraId}/roi`, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({drawElements: drawingState.elements})
        });

        if (response.ok) {
            alert('保存成功！');
        } else {
            const error = await response.text();
            alert('保存失败: ' + error);
        }
    } catch (error) {
        console.error('Failed to save elements:', error);
        alert('保存失败: ' + error.message);
    }
}

