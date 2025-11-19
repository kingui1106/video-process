let cameras = [];

// Load cameras on page load
window.addEventListener('DOMContentLoaded', () => {
    loadCameras();
});

// Load all cameras from API
async function loadCameras() {
    try {
        const response = await fetch('/api/cameras');
        cameras = await response.json();
        renderCameraList();
    } catch (error) {
        console.error('Failed to load cameras:', error);
    }
}

// Render camera list
function renderCameraList() {
    const list = document.getElementById('cameraList');
    if (!list) return;
    
    list.innerHTML = '';

    if (cameras.length === 0) {
        list.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">📹</div>
                <div class="empty-state-text">暂无摄像头</div>
                <div class="empty-state-text">点击上方"添加摄像头"按钮开始配置</div>
            </div>
        `;
        return;
    }

    cameras.forEach(camera => {
        const item = createCameraListItem(camera);
        list.appendChild(item);
    });
}

// Create a camera list item
function createCameraListItem(camera) {
    const item = document.createElement('div');
    item.className = 'camera-item';
    
    const elementCount = camera.drawElements ? camera.drawElements.length : 0;
    
    item.innerHTML = `
        <div class="camera-item-header">
            <div class="camera-info">
                <div class="camera-name">${camera.name}</div>
                <div class="camera-id">ID: ${camera.id}</div>
            </div>
            <span class="status ${camera.enabled ? 'online' : 'offline'}">
                ${camera.enabled ? '启用' : '禁用'}
            </span>
        </div>
        <div class="camera-url">${camera.rtspUrl}</div>
        <div class="camera-stats">
            <div class="stat-item">
                <span class="stat-label">绘制元素:</span>
                <span class="stat-value">${elementCount}</span>
            </div>
        </div>
        <div class="camera-actions">
            <button class="btn-primary" onclick="goToCameraConfig('${camera.id}')">
                🎨 配置绘制
            </button>
            <button class="btn-secondary" onclick="showEditCameraModal('${camera.id}')">
                ✏️ 编辑
            </button>
            <button class="btn-danger" onclick="deleteCamera('${camera.id}')">
                🗑️ 删除
            </button>
        </div>
    `;
    
    return item;
}

// Navigate to camera configuration page
function goToCameraConfig(cameraId) {
    window.location.href = `/camera-config?id=${cameraId}`;
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
            response = await fetch(`/api/cameras/${editingCameraId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(cameraData)
            });
        } else {
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
