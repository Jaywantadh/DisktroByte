package main

func getJavaScript() string {
	return `
        // Global variables
        let currentUser = null;
        let sessionToken = null;

        // Initialize application
        document.addEventListener('DOMContentLoaded', function() {
            // Check if user is already logged in
            checkSession();
            
            // Setup navigation
            setupNavigation();
            
            // Setup drag and drop for file upload
            setupFileUpload();
            
            // Start periodic updates
            startPeriodicUpdates();
        });

        // Session management
        async function checkSession() {
            try {
                const response = await fetch('/api/auth/validate', {
                    method: 'GET',
                    credentials: 'include'
                });

                if (response.ok) {
                    const result = await response.json();
                    if (result.success) {
                        currentUser = result.data.user;
                        sessionToken = currentUser.session_token;
                        showMainApp();
                        return;
                    }
                }
            } catch (error) {
                console.error('Session validation error:', error);
            }
            
            showLoginScreen();
        }

        async function login() {
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const nodeId = document.getElementById('nodeId').value;

            if (!username || !password) {
                showLoginStatus('Please enter username and password', 'error');
                return;
            }

            try {
                const response = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        username: username,
                        password: password,
                        node_id: nodeId
                    }),
                    credentials: 'include'
                });

                const result = await response.json();

                if (result.success) {
                    currentUser = result.user;
                    sessionToken = result.token;
                    showMainApp();
                    showLoginStatus('Login successful!', 'success');
                } else {
                    showLoginStatus(result.message, 'error');
                }
            } catch (error) {
                showLoginStatus('Login failed: ' + error.message, 'error');
            }
        }

        async function register() {
            const username = document.getElementById('regUsername').value;
            const password = document.getElementById('regPassword').value;
            const displayName = document.getElementById('regDisplayName').value;
            const email = document.getElementById('regEmail').value;
            const role = document.getElementById('regRole').value;

            if (!username || !password) {
                showLoginStatus('Please enter username and password', 'error');
                return;
            }

            try {
                const response = await fetch('/api/auth/register', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        username: username,
                        password: password,
                        role: role,
                        profile: {
                            display_name: displayName,
                            email: email
                        }
                    })
                });

                const result = await response.json();

                if (result.success) {
                    showLoginStatus('Registration successful! Please login.', 'success');
                    showLoginForm();
                } else {
                    showLoginStatus(result.message, 'error');
                }
            } catch (error) {
                showLoginStatus('Registration failed: ' + error.message, 'error');
            }
        }

        async function logout() {
            try {
                await fetch('/api/auth/logout', {
                    method: 'POST',
                    credentials: 'include'
                });
            } catch (error) {
                console.error('Logout error:', error);
            }

		// Close SSE connection
		closeLogSSE();
		
		currentUser = null;
		sessionToken = null;
		showLoginScreen();
        }

        // UI management
        function showLoginScreen() {
            document.getElementById('loginScreen').style.display = 'flex';
            document.getElementById('mainApp').style.display = 'none';
        }

        function showMainApp() {
            document.getElementById('loginScreen').style.display = 'none';
            document.getElementById('mainApp').style.display = 'flex';
            
            if (currentUser) {
                updateUserInfo();
                setupRoleBasedUI();
                loadDashboard();
            }
        }

        function updateUserInfo() {
            document.getElementById('userDisplayName').textContent = 
                currentUser.profile.display_name || currentUser.username;
            document.getElementById('userRole').textContent = 
                currentUser.role.charAt(0).toUpperCase() + currentUser.role.slice(1);
        }

        function setupRoleBasedUI() {
            // Show/hide navigation items based on role
            const senderNav = document.getElementById('senderNav');
            const receiverNav = document.getElementById('receiverNav');
            const adminNav = document.getElementById('adminNav');

            // Hide all role-specific navs first
            senderNav.style.display = 'none';
            receiverNav.style.display = 'none';
            adminNav.style.display = 'none';

            // Show appropriate navs based on role
            if (currentUser.role === 'sender' || currentUser.role === 'user' || 
                currentUser.role === 'admin' || currentUser.role === 'superadmin') {
                senderNav.style.display = 'block';
            }

            if (currentUser.role === 'receiver' || currentUser.role === 'user' || 
                currentUser.role === 'admin' || currentUser.role === 'superadmin') {
                receiverNav.style.display = 'block';
            }

            if (currentUser.role === 'admin' || currentUser.role === 'superadmin') {
                adminNav.style.display = 'block';
            }
        }

        function showLoginForm() {
            document.getElementById('loginForm').style.display = 'block';
            document.getElementById('registerForm').style.display = 'none';
        }

        function showRegisterForm() {
            document.getElementById('loginForm').style.display = 'none';
            document.getElementById('registerForm').style.display = 'block';
        }

        function showLoginStatus(message, type) {
            const status = document.getElementById('loginStatus');
            status.textContent = message;
            status.className = 'status ' + type;
            status.style.display = 'block';

            setTimeout(() => {
                status.style.display = 'none';
            }, 5000);
        }

        // Navigation
        function setupNavigation() {
            document.querySelectorAll('.nav-link').forEach(link => {
                link.addEventListener('click', function(e) {
                    e.preventDefault();
                    
                    // Remove active class from all links
                    document.querySelectorAll('.nav-link').forEach(l => l.classList.remove('active'));
                    
                    // Add active class to clicked link
                    this.classList.add('active');
                    
                    // Show corresponding content
                    const section = this.getAttribute('data-section');
                    showSection(section);
                });
            });
        }

        function showSection(sectionName) {
            // Hide all content areas
            document.querySelectorAll('.content-area').forEach(area => {
                area.classList.remove('active');
            });

            // Show selected section
            document.getElementById(sectionName).classList.add('active');

            // Update page title
            const titles = {
                'dashboard': 'Dashboard',
                'send-files': 'Send Files',
                'receive-files': 'Receive Files',
                'file-logs': 'File Logs',
                'network': 'Network Status',
                'streaming': 'Streaming',
                'admin': 'Administration'
            };

            document.getElementById('contentTitle').textContent = titles[sectionName] || sectionName;

            // Load section-specific data
            loadSectionData(sectionName);
        }

        async function loadSectionData(section) {
            switch (section) {
                case 'dashboard':
                    loadDashboard();
                    break;
                case 'send-files':
                    loadSentFiles();
                    break;
                case 'receive-files':
                    loadReceivedFiles();
                    break;
                case 'file-logs':
                    loadFileLogs();
                    break;
                case 'network':
                    loadNetworkStatus();
                    break;
                case 'streaming':
                    loadStreamingStatus();
                    break;
                case 'admin':
                    if (currentUser.role === 'admin' || currentUser.role === 'superadmin') {
                        loadAdminData();
                    }
                    break;
            }
        }

        // Dashboard
        async function loadDashboard() {
            try {
                // Load dashboard statistics
                const response = await apiCall('/api/system/stats');
                if (response && response.success) {
                    updateDashboardStats(response.data);
                }
            } catch (error) {
                console.error('Failed to load dashboard:', error);
            }
        }

        function updateDashboardStats(data) {
            document.getElementById('totalFiles').textContent = data.total_files || '0';
            document.getElementById('activePeers').textContent = data.active_peers || '0';
            document.getElementById('streamingSessions').textContent = data.streaming_sessions || '0';
            document.getElementById('totalStorage').textContent = formatFileSize(data.total_storage || 0);
        }

        // File operations and queue management
        let fileQueue = [];
        
        function setupFileUpload() {
            const uploadArea = document.querySelector('.upload-area');
            const fileInput = document.getElementById('fileInput');

            if (uploadArea) {
                uploadArea.addEventListener('dragover', function(e) {
                    e.preventDefault();
                    this.classList.add('dragover');
                });

                uploadArea.addEventListener('dragleave', function() {
                    this.classList.remove('dragover');
                });

                uploadArea.addEventListener('drop', function(e) {
                    e.preventDefault();
                    this.classList.remove('dragover');
                    
                    const files = e.dataTransfer.files;
                    if (files.length > 0) {
                        addFilesToQueue(files);
                    }
                });
            }
            
            // Handle file input change
            if (fileInput) {
                fileInput.addEventListener('change', function(e) {
                    if (e.target.files.length > 0) {
                        addFilesToQueue(e.target.files);
                        e.target.value = ''; // Clear input for next selection
                    }
                });
            }
        }
        
        function addFilesToQueue(files) {
            for (let file of files) {
                // Check if file is already in queue
                if (!fileQueue.find(f => f.name === file.name && f.size === file.size)) {
                    fileQueue.push({
                        file: file,
                        name: file.name,
                        size: file.size,
                        type: file.type,
                        id: Date.now() + Math.random()
                    });
                }
            }
            updateFileQueueDisplay();
        }
        
        function removeFileFromQueue(fileId) {
            fileQueue = fileQueue.filter(f => f.id !== fileId);
            updateFileQueueDisplay();
        }
        
        function clearFileQueue() {
            fileQueue = [];
            updateFileQueueDisplay();
        }
        
        function updateFileQueueDisplay() {
            const queueContainer = document.getElementById('fileQueue');
            const queueList = document.getElementById('fileQueueList');
            const totalFiles = document.getElementById('queueTotalFiles');
            const totalSize = document.getElementById('queueTotalSize');
            const sendBtn = document.getElementById('sendFilesBtn');
            
            if (fileQueue.length === 0) {
                queueContainer.style.display = 'none';
                sendBtn.disabled = true;
                return;
            }
            
            queueContainer.style.display = 'block';
            sendBtn.disabled = false;
            
            let html = '';
            let totalSizeBytes = 0;
            
            fileQueue.forEach(fileItem => {
                totalSizeBytes += fileItem.size;
                const fileIcon = getFileIcon(fileItem.type);
                
                html += '<div class="file-item" style="display: flex; justify-content: space-between; align-items: center; padding: 10px; margin-bottom: 10px; border: 1px solid #e9ecef; border-radius: 8px;">' +
                    '<div style="display: flex; align-items: center;">' +
                        '<span style="font-size: 24px; margin-right: 10px;">' + fileIcon + '</span>' +
                        '<div>' +
                            '<div style="font-weight: 600;">' + fileItem.name + '</div>' +
                            '<div style="font-size: 12px; color: #6c757d;">' + formatFileSize(fileItem.size) + ' • ' + (fileItem.type || 'Unknown type') + '</div>' +
                        '</div>' +
                    '</div>' +
                    '<button onclick="removeFileFromQueue(' + fileItem.id + ')" style="background: #dc3545; color: white; border: none; padding: 5px 10px; border-radius: 4px; cursor: pointer;">✕</button>' +
                '</div>';
            });
            
            queueList.innerHTML = html;
            totalFiles.textContent = fileQueue.length;
            totalSize.textContent = formatFileSize(totalSizeBytes);
        }
        
        function getFileIcon(mimeType) {
            if (!mimeType) return '📄';
            
            if (mimeType.startsWith('image/')) return '🖼️';
            if (mimeType.startsWith('video/')) return '🎥';
            if (mimeType.startsWith('audio/')) return '🎵';
            if (mimeType.includes('pdf')) return '📕';
            if (mimeType.includes('word')) return '📘';
            if (mimeType.includes('excel') || mimeType.includes('sheet')) return '📊';
            if (mimeType.includes('powerpoint') || mimeType.includes('presentation')) return '📹';
            if (mimeType.includes('zip') || mimeType.includes('rar') || mimeType.includes('7z')) return '🗜️';
            if (mimeType.includes('text/')) return '📝';
            
            return '📄';
        }

        async function sendFiles() {
            const password = document.getElementById('filePassword').value;
            
            if (fileQueue.length === 0) {
                showSendStatus('Please select files to send', 'error');
                return;
            }

            if (!password) {
                showSendStatus('Please enter encryption password', 'error');
                return;
            }

            const formData = new FormData();
            for (let fileItem of fileQueue) {
                formData.append('file', fileItem.file);
            }
            formData.append('password', password);

            try {
                showSendProgress(true);
                updateSendProgress(0);

                const response = await fetch('/api/files/chunk', {
                    method: 'POST',
                    body: formData,
                    credentials: 'include'
                });

                const result = await response.json();

                if (result.success) {
                    updateSendProgress(100);
                    showSendStatus('Files sent successfully!', 'success');
                    clearFileQueue(); // Clear the queue after successful sending
                    loadSentFiles(); // Refresh sent files list
                } else {
                    showSendStatus('Failed to send files: ' + result.message, 'error');
                }
            } catch (error) {
                showSendStatus('Error sending files: ' + error.message, 'error');
            } finally {
                setTimeout(() => showSendProgress(false), 2000);
            }
        }

        function showSendProgress(show) {
            document.getElementById('sendProgress').style.display = show ? 'block' : 'none';
        }

        function updateSendProgress(percent) {
            const bar = document.getElementById('sendProgressBar');
            bar.style.width = percent + '%';
            bar.textContent = Math.round(percent) + '%';
        }

        function showSendStatus(message, type) {
            const status = document.getElementById('sendStatus');
            status.textContent = message;
            status.className = 'status ' + type;
            status.style.display = 'block';

            setTimeout(() => {
                status.style.display = 'none';
            }, 5000);
        }

	async function loadSentFiles() {
		try {
			// First try to get actual files from the files list API
			const filesResponse = await apiCall('/api/files/list');
			if (filesResponse && filesResponse.success && filesResponse.data.files && filesResponse.data.files.length > 0) {
				console.log('📜 Found actual files in system:', filesResponse.data.files.length);
				displaySentFilesFromAPI(filesResponse.data.files);
				return;
			}
			
			// Fallback to logs-based approach
			const response = await apiCall('/api/files/logs');
			if (response && response.success) {
				displaySentFiles(response.data.logs || response.data);
			}
		} catch (error) {
			console.error('Failed to load sent files:', error);
		}
	}
	
	function displaySentFilesFromAPI(files) {
		const container = document.getElementById('sentFilesList');
		if (!files || files.length === 0) {
			container.innerHTML = '<p class="text-muted">No files uploaded yet.</p>';
			return;
		}
		
		let html = '';
		files.forEach(file => {
			html += '<div class="file-item" style="border-left: 4px solid #10B981; margin-bottom: 15px; padding: 15px; background: rgba(16, 185, 129, 0.05);">' +
				'<div class="file-info">' +
					'<h4>📁 ' + (file.file_name || file.name) + '</h4>' +
					'<div class="file-details">' +
						'📊 Size: ' + formatFileSize(file.file_size || file.size) + ' | ' +
						(file.chunk_count ? '🧩 Chunks: ' + file.chunk_count + ' | ' : '') +
						(file.replica_count ? '🔄 Replicas: ' + file.replica_count + ' | ' : '') +
						'📅 Date: ' + new Date(file.created_at || Date.now()).toLocaleString() + ' | ' +
						(file.owner_id ? '👤 Owner: ' + file.owner_id + ' | ' : '') +
						(file.health_status ? '❤️ Status: ' + file.health_status : '') +
					'</div>' +
					(file.description ? '<p style="color: #6c757d; font-size: 0.9em; margin-top: 8px;">' + file.description + '</p>' : '') +
				'</div>' +
				'<div class="file-actions">' +
					'<span class="status-indicator status-online"></span>' +
					'Uploaded' +
				'</div>' +
			'</div>';
		});
		
		container.innerHTML = html;
	}

        function displaySentFiles(logs) {
            const container = document.getElementById('sentFilesList');
            if (!logs || logs.length === 0) {
                container.innerHTML = '<p class="text-muted">No files sent yet.</p>';
                return;
            }

            const sentFiles = logs.filter(log => log.operation === 'chunk' || log.operation === 'upload');
            let html = '';

            sentFiles.forEach(log => {
                html += '<div class="file-item" style="border-left: 4px solid #10B981; margin-bottom: 15px; padding: 15px; background: rgba(16, 185, 129, 0.05);">' +
                    '<div class="file-info">' +
                        '<h4>📤 ' + log.file_name + '</h4>' +
                        '<div class="file-details">' +
                            '📊 Size: ' + formatFileSize(log.file_size) + ' | ' +
                            '🧩 Chunks: ' + log.chunk_count + ' | ' +
                            '🔄 Replicas: ' + (log.replica_info ? log.replica_info.length : 0) + ' | ' +
                            '📅 Date: ' + new Date(log.timestamp).toLocaleString() +
                        '</div>' +
                    '</div>' +
                    '<div class="file-actions">' +
                        '<span class="status-indicator status-' + (log.status === 'completed' ? 'online' : 'offline') + '"></span>' +
                        log.status.charAt(0).toUpperCase() + log.status.slice(1) +
                    '</div>' +
                '</div>';
            });

            container.innerHTML = html;
        }

        async function loadReceivedFiles() {
            const container = document.getElementById('receivedFilesList');
            
            // For superadmin, show all received file details
            if (currentUser.role === 'superadmin') {
                try {
                    const response = await apiCall('/api/files/received');
                    if (response && response.success) {
                        displayReceivedFilesSuperadmin(response.data);
                    }
                } catch (error) {
                    console.error('Failed to load received files:', error);
                    container.innerHTML = '<div class="card"><div class="card-body"><p class="text-danger">Failed to load received files: ' + error.message + '</p></div></div>';
                }
            } else if (currentUser.role === 'receiver') {
                // For receiver role, show limited information for privacy
                container.innerHTML = '<div class="card">' +
                    '<div class="card-body text-center">' +
                        '<h5>🔒 Privacy Protected</h5>' +
                        '<p class="text-muted">File details are hidden for privacy. Files are stored securely and automatically.</p>' +
                        '<p class="text-muted">Only general statistics are available.</p>' +
                        '<div style="margin-top: 20px; padding: 15px; background: rgba(16, 185, 129, 0.1); border-radius: 8px;">' +
                            '<p><strong>ℹ️ For full file visibility, contact your administrator for superadmin access.</strong></p>' +
                        '</div>' +
                    '</div>' +
                '</div>';
            } else {
                // For other roles, show basic received files info
                try {
                    const response = await apiCall('/api/files/logs');
                    if (response && response.success) {
                        const receivedLogs = (response.data.logs || response.data).filter(log => log.operation === 'receive');
                        displayReceivedFiles(receivedLogs);
                    }
                } catch (error) {
                    console.error('Failed to load received files:', error);
                }
            }
        }

        function displayReceivedFiles(files) {
            const container = document.getElementById('receivedFilesList');
            if (!files || files.length === 0) {
                container.innerHTML = '<p class="text-muted">No files received yet.</p>';
                return;
            }

            let html = '';
            files.forEach(file => {
                html += '<div class="file-item" style="border-left: 4px solid #3B82F6; margin-bottom: 15px; padding: 15px; background: rgba(59, 130, 246, 0.05);">' +
                    '<div class="file-info">' +
                        '<h4>📥 ' + (file.file_name || '[Received File]') + '</h4>' +
                        '<div class="file-details">' +
                            '📅 Received: ' + new Date(file.timestamp || file.received_at).toLocaleString() + ' | ' +
                            '🔒 Status: Stored Securely' +
                        '</div>' +
                    '</div>' +
                    '<div class="file-actions">' +
                        '<span class="status-indicator status-online"></span>' +
                        'Stored' +
                    '</div>' +
                '</div>';
            });

            container.innerHTML = html;
        }
        
        function displayReceivedFilesSuperadmin(data) {
            const container = document.getElementById('receivedFilesList');
            const files = data.files || [];
            
            if (!files || files.length === 0) {
                container.innerHTML = '<div class="card"><div class="card-body text-center"><p class="text-muted">📥 No files received yet.</p></div></div>';
                return;
            }
            
            // Add summary statistics
            let html = '<div class="card" style="margin-bottom: 20px; border: 2px solid #F59E0B;">' +
                '<div class="card-header" style="background: linear-gradient(135deg, #F59E0B, #D97706); color: white;">' +
                    '<h4 class="card-title" style="margin: 0;">🔑 SUPERADMIN: Received Files Overview</h4>' +
                '</div>' +
                '<div class="card-body">' +
                    '<div class="stats-grid" style="grid-template-columns: repeat(4, 1fr);">' +
                        '<div class="stat-card" style="border: 1px solid #F59E0B;">' +
                            '<div class="stat-number" style="color: #D97706;">' + data.total_count + '</div>' +
                            '<div class="stat-label">Total Files</div>' +
                        '</div>' +
                        '<div class="stat-card" style="border: 1px solid #F59E0B;">' +
                            '<div class="stat-number" style="color: #D97706;">' + formatFileSize(data.total_size) + '</div>' +
                            '<div class="stat-label">Total Size</div>' +
                        '</div>' +
                        '<div class="stat-card" style="border: 1px solid #F59E0B;">' +
                            '<div class="stat-number" style="color: #D97706;">' + data.node_id.substring(0, 8) + '...</div>' +
                            '<div class="stat-label">Node ID</div>' +
                        '</div>' +
                        '<div class="stat-card" style="border: 1px solid #F59E0B;">' +
                            '<div class="stat-number" style="color: #D97706;">' + data.access_level.toUpperCase() + '</div>' +
                            '<div class="stat-label">Access Level</div>' +
                        '</div>' +
                    '</div>' +
                '</div>' +
            '</div>';
            
            // Add detailed file information
            files.forEach(file => {
                const tags = file.tags ? file.tags.map(tag => '<span class="badge" style="background: #10B981; color: white; padding: 2px 8px; border-radius: 12px; margin-right: 5px;">' + tag + '</span>').join('') : '';
                
                html += '<div class="file-item" style="border: 2px solid #EF4444; margin-bottom: 20px; padding: 20px; background: linear-gradient(135deg, rgba(239, 68, 68, 0.05), rgba(220, 38, 38, 0.05)); border-radius: 12px;">' +
                    '<div class="file-info">' +
                        '<h4 style="color: #DC2626;">🔍 ' + file.filename + '</h4>' +
                        '<p style="font-style: italic; color: #6B7280; margin: 5px 0;">Original: ' + file.original_name + '</p>' +
                        '<div class="file-details" style="margin-top: 10px;">' +
                            '<div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 15px; margin-bottom: 15px;">' +
                                '<div><strong>📊 Size:</strong> ' + formatFileSize(file.file_size) + '</div>' +
                                '<div><strong>🧩 Chunks:</strong> ' + file.chunk_count + '</div>' +
                                '<div><strong>👤 Sender:</strong> ' + file.sender_user + '</div>' +
                                '<div><strong>🌐 Node:</strong> ' + file.sender_node.substring(0, 12) + '...</div>' +
                                '<div><strong>📅 Received:</strong> ' + new Date(file.received_at).toLocaleString() + '</div>' +
                                '<div><strong>🔒 Encryption:</strong> ' + file.encryption + '</div>' +
                                '<div><strong>📁 Type:</strong> ' + file.file_type + '</div>' +
                                '<div><strong>📡 Access Count:</strong> ' + file.access_count + '</div>' +
                            '</div>' +
                            '<div style="margin-bottom: 10px;"><strong>📏 Storage Locations:</strong></div>' +
                            '<div style="margin-bottom: 15px;">' + file.stored_at.map(loc => '<span class="badge" style="background: #6366F1; color: white; padding: 4px 10px; border-radius: 15px; margin-right: 8px;">' + loc + '</span>').join('') + '</div>' +
                            (file.tags && file.tags.length > 0 ? '<div><strong>🏷️ Tags:</strong> ' + tags + '</div>' : '') +
                        '</div>' +
                    '</div>' +
                    '<div class="file-actions">' +
                        '<span class="status-indicator status-online" style="animation: pulseGreen 2s infinite;"></span>' +
                        '<strong style="color: #10B981;">' + file.status.toUpperCase() + '</strong>' +
                    '</div>' +
                '</div>';
            });
            
            container.innerHTML = html;
        }

        async function loadFileLogs() {
            try {
                const response = await apiCall('/api/files/logs');
                if (response && response.success) {
                    displayFileLogs(response.data);
                }
            } catch (error) {
                console.error('Failed to load file logs:', error);
            }
        }

        function displayFileLogs(data) {
            const logArea = document.getElementById('fileLogArea');
            const logs = data.logs || data;
            const userRole = data.user_role;
            const hasReceivedFiles = data.has_received_files;
            
            if (!logs || logs.length === 0) {
                logArea.innerHTML = '<div class="log-entry info">📋 No file operations logged yet. File activities will appear here in real-time.</div>';
                return;
            }
            
            // Add header for superadmin showing additional access info
            let html = '';
            if (userRole === 'superadmin' && hasReceivedFiles) {
                html += '<div class="log-entry warning" style="background: linear-gradient(135deg, rgba(245, 158, 11, 0.2), rgba(217, 119, 6, 0.2)); border: 2px solid #F59E0B; margin-bottom: 15px; padding: 15px; border-radius: 8px;">' +
                    '<strong>🔑 SUPERADMIN ACCESS: Showing all file operations including received files</strong><br>' +
                    'Total logs: ' + logs.length + ' | Role: ' + userRole.toUpperCase() + ' | Access Level: FULL' +
                '</div>';
            }
            
            logs.slice(-50).reverse().forEach(log => { // Show last 50 logs, most recent first
                const timestamp = new Date(log.timestamp).toLocaleString();
                const logClass = log.status === 'completed' ? 'success' : 
                                log.status === 'failed' ? 'error' : 
                                log.status === 'in_progress' ? 'warning' : 'info';
                
                // Add icons based on operation type
                const operationIcons = {
                    'chunk': '📦',
                    'upload': '⬆️',
                    'download': '⬇️',
                    'receive': '📥',
                    'broadcast': '📡',
                    'reassemble': '🔧'
                };
                
                const icon = operationIcons[log.operation] || '📄';
                const statusIcon = log.status === 'completed' ? '✅' : 
                                  log.status === 'failed' ? '❌' : 
                                  log.status === 'in_progress' ? '🔄' : 'ℹ️';
                
                html += '<div class="log-entry ' + logClass + ' fade-in" style="animation-delay: 0.1s">' +
                    '<strong>' + icon + ' [' + timestamp + ']</strong><br>' +
                    '🎯 <strong>' + log.operation.toUpperCase() + ':</strong> ' + log.file_name + '<br>' +
                    '📊 Status: ' + statusIcon + ' ' + log.status.toUpperCase() + ' | ' +
                    '📐 Size: ' + formatFileSize(log.file_size);
                
                if (log.chunk_count && log.chunk_count > 0) {
                    html += ' | 🧩 Chunks: ' + log.chunk_count;
                }
                if (log.replica_info && log.replica_info.length) {
                    html += ' | 🔄 Replicas: ' + log.replica_info.length;
                }
                if (log.progress && log.progress > 0 && log.progress < 100) {
                    html += '<br>📈 Progress: ' + Math.round(log.progress) + '%';
                }
                if (log.error) {
                    html += '<br>⚠️ <span style="color: #ef4444;">Error: ' + log.error + '</span>';
                }
                
                html += '</div>';
            });

            logArea.innerHTML = html;
            logArea.scrollTop = logArea.scrollHeight;
            
            // Add a visual indicator that logs are being updated
            const updateIndicator = document.createElement('div');
            updateIndicator.className = 'log-entry info';
            updateIndicator.style.opacity = '0.6';
            updateIndicator.innerHTML = '🔄 <em>Real-time updates active...</em>';
            logArea.appendChild(updateIndicator);
            
            setTimeout(() => {
                if (updateIndicator.parentNode) {
                    updateIndicator.remove();
                }
            }, 2000);
        }

        // Network management
        async function loadNetworkStatus() {
            try {
                const response = await apiCall('/api/network/status');
                if (response && response.success) {
                    displayNetworkStatus(response.data);
                }
            } catch (error) {
                console.error('Failed to load network status:', error);
            }
        }

        function displayNetworkStatus(networkData) {
            const container = document.getElementById('networkStatus');
            
            let html = '<h4>Connected Peers</h4>';
            
            if (!networkData.peers || networkData.peers.length === 0) {
                html += '<p class="text-muted">No peers connected.</p>';
            } else {
                html += '<div class="table-responsive"><table class="table">' +
                    '<thead>' +
                        '<tr>' +
                            '<th>Status</th>' +
                            '<th>Node ID</th>' +
                            '<th>Address</th>' +
                            '<th>Last Seen</th>' +
                            '<th>Files</th>' +
                        '</tr>' +
                    '</thead>' +
                    '<tbody>';

                networkData.peers.forEach(peer => {
                    const statusClass = peer.status === 'online' ? 'online' : 'offline';
                    const lastSeen = new Date(peer.last_seen).toLocaleString();
                    
                    html += '<tr>' +
                        '<td><span class="status-indicator status-' + statusClass + '"></span>' + peer.status + '</td>' +
                        '<td>' + (peer.id || 'Unknown') + '</td>' +
                        '<td>' + peer.address + ':' + peer.port + '</td>' +
                        '<td>' + lastSeen + '</td>' +
                        '<td>' + (peer.files ? peer.files.length : 0) + '</td>' +
                    '</tr>';
                });

                html += '</tbody></table></div>';
            }

            container.innerHTML = html;
        }

        async function refreshNetwork() {
            loadNetworkStatus();
        }

        // Streaming management
        async function loadStreamingStatus() {
            try {
                const response = await apiCall('/api/stream/status');
                if (response && response.success) {
                    displayStreamingStatus(response.data);
                }
            } catch (error) {
                console.error('Failed to load streaming status:', error);
            }
        }

        function displayStreamingStatus(streamData) {
            const container = document.getElementById('activeStreams');
            
            if (!streamData.sessions || streamData.sessions.length === 0) {
                container.innerHTML = '<p class="text-muted">No active streaming sessions.</p>';
                return;
            }

            let html = '<h4>Active Streaming Sessions</h4>';
            html += '<div class="table-responsive"><table class="table">' +
                '<thead>' +
                    '<tr>' +
                        '<th>Session ID</th>' +
                        '<th>File</th>' +
                        '<th>Progress</th>' +
                        '<th>Speed</th>' +
                        '<th>ETA</th>' +
                        '<th>Actions</th>' +
                    '</tr>' +
                '</thead>' +
                '<tbody>';

            streamData.sessions.forEach(session => {
                html += '<tr>' +
                    '<td>' + session.id.substring(0, 8) + '...</td>' +
                    '<td>' + session.file_name + '</td>' +
                    '<td>' +
                        '<div class="progress" style="height: 15px;">' +
                            '<div class="progress-bar" style="width: ' + session.progress + '%">' +
                                Math.round(session.progress) + '%' +
                            '</div>' +
                        '</div>' +
                    '</td>' +
                    '<td>' + formatSpeed(session.speed) + '</td>' +
                    '<td>' + formatDuration(session.eta) + '</td>' +
                    '<td>' +
                        '<button class="btn btn-sm" onclick="pauseStream(\'' + session.id + '\')">' +
                            (session.paused ? '▶️' : '⏸️') +
                        '</button> ' +
                        '<button class="btn btn-sm" onclick="cancelStream(\'' + session.id + '\')">❌</button>' +
                    '</td>' +
                '</tr>';
            });

            html += '</tbody></table></div>';
            container.innerHTML = html;
        }

        async function pauseStream(sessionId) {
            try {
                await apiCall('/api/stream/control', 'POST', {
                    session_id: sessionId,
                    action: 'pause'
                });
                loadStreamingStatus(); // Refresh
            } catch (error) {
                console.error('Failed to pause stream:', error);
            }
        }

        async function cancelStream(sessionId) {
            try {
                await apiCall('/api/stream/control', 'POST', {
                    session_id: sessionId,
                    action: 'cancel'
                });
                loadStreamingStatus(); // Refresh
            } catch (error) {
                console.error('Failed to cancel stream:', error);
            }
        }

        // Admin functions
        async function loadAdminData() {
            try {
                const response = await apiCall('/api/users/stats');
                if (response && response.success) {
                    updateAdminStats(response.data);
                }

                const systemResponse = await apiCall('/api/system/stats');
                if (systemResponse && systemResponse.success) {
                    updateAdminSystemStats(systemResponse.data);
                }

                loadSystemLogs();
            } catch (error) {
                console.error('Failed to load admin data:', error);
            }
        }

        function updateAdminStats(stats) {
            document.getElementById('adminTotalUsers').textContent = stats.total_users || '0';
            document.getElementById('adminActiveSessions').textContent = stats.active_sessions || '0';
        }

        function updateAdminSystemStats(stats) {
            document.getElementById('adminTotalFiles').textContent = stats.total_files || '0';
            document.getElementById('adminNetworkNodes').textContent = stats.total_nodes || '0';
        }

        async function loadUsers() {
            try {
                const response = await apiCall('/api/users');
                if (response && response.success) {
                    displayUsers(response.data);
                }
            } catch (error) {
                console.error('Failed to load users:', error);
            }
        }

        function displayUsers(users) {
            const container = document.getElementById('usersList');
            
            if (!users || users.length === 0) {
                container.innerHTML = '<p class="text-muted">No users found.</p>';
                return;
            }

            let html = '<div class="table-responsive"><table class="table">' +
                '<thead>' +
                    '<tr>' +
                        '<th>Username</th>' +
                        '<th>Role</th>' +
                        '<th>Status</th>' +
                        '<th>Last Login</th>' +
                        '<th>Actions</th>' +
                    '</tr>' +
                '</thead>' +
                '<tbody>';

            users.forEach(user => {
                const lastLogin = user.last_login ? new Date(user.last_login).toLocaleString() : 'Never';
                const statusClass = user.is_active ? 'online' : 'offline';
                
                html += '<tr>' +
                    '<td>' + user.username + '</td>' +
                    '<td>' + user.role + '</td>' +
                    '<td><span class="status-indicator status-' + statusClass + '"></span>' + 
                        (user.is_active ? 'Active' : 'Inactive') + '</td>' +
                    '<td>' + lastLogin + '</td>' +
                    '<td>' +
                        '<button class="btn btn-sm" onclick="editUser(\'' + user.id + '\')">' +
                            'Edit' +
                        '</button>' +
                    '</td>' +
                '</tr>';
            });

            html += '</tbody></table></div>';
            container.innerHTML = html;
        }

        async function loadSystemLogs() {
            try {
                const response = await apiCall('/api/system/logs');
                if (response && response.success) {
                    displaySystemLogs(response.data);
                }
            } catch (error) {
                console.error('Failed to load system logs:', error);
            }
        }

        function displaySystemLogs(logs) {
            const logArea = document.getElementById('systemLogArea');
            if (!logs || logs.length === 0) {
                logArea.innerHTML = '<div class="log-entry info">No system logs available.</div>';
                return;
            }

            let html = '';
            logs.slice(-100).reverse().forEach(log => { // Show last 100 logs
                const timestamp = new Date(log.timestamp).toLocaleTimeString();
                html += '<div class="log-entry ' + log.level + '">' +
                    '[' + timestamp + '] ' + log.message +
                '</div>';
            });

            logArea.innerHTML = html;
            logArea.scrollTop = logArea.scrollHeight;
        }

        // Utility functions
        async function apiCall(url, method = 'GET', data = null) {
            const options = {
                method: method,
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json'
                }
            };

            if (data && method !== 'GET') {
                options.body = JSON.stringify(data);
            }

            try {
                const response = await fetch(url, options);
                
                if (response.status === 401) {
                    // Session expired
                    logout();
                    return null;
                }

                return await response.json();
            } catch (error) {
                console.error('API call failed:', error);
                throw error;
            }
        }

        function formatFileSize(bytes) {
            if (bytes === 0) return '0 B';
            
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function formatSpeed(bytesPerSecond) {
            return formatFileSize(bytesPerSecond) + '/s';
        }

        function formatDuration(seconds) {
            if (!seconds || seconds <= 0) return 'N/A';
            
            const hours = Math.floor(seconds / 3600);
            const minutes = Math.floor((seconds % 3600) / 60);
            const secs = Math.floor(seconds % 60);
            
            if (hours > 0) {
                return hours + 'h ' + minutes + 'm ' + secs + 's';
            } else if (minutes > 0) {
                return minutes + 'm ' + secs + 's';
            } else {
                return secs + 's';
            }
        }

        function startPeriodicUpdates() {
            // Update dashboard every 10 seconds
            setInterval(() => {
                if (currentUser && document.getElementById('dashboard').classList.contains('active')) {
                    loadDashboard();
                }
            }, 10000);

			// Setup Server-Sent Events for real-time log updates
			setupLogSSE();
			
			// Fallback: Update file logs every 10 seconds if SSE fails
			setInterval(() => {
				if (currentUser && document.getElementById('file-logs').classList.contains('active') && !window.logSSEActive) {
					loadFileLogs();
					showUpdateIndicator('fileLogArea');
				}
			}, 10000);

            // Update streaming status every 2 seconds
            setInterval(() => {
                if (currentUser && document.getElementById('streaming').classList.contains('active')) {
                    loadStreamingStatus();
                }
            }, 2000);

            // Update network status every 5 seconds
            setInterval(() => {
                if (currentUser && document.getElementById('network').classList.contains('active')) {
                    loadNetworkStatus();
                    showUpdateIndicator('networkPeersList');
                }
            }, 5000);
            
		// Update sent files list every 5 seconds
		setInterval(() => {
			if (currentUser && document.getElementById('send-files').classList.contains('active')) {
				loadSentFiles();
			}
		}, 5000);
		
		// Update available files for reassembly every 8 seconds
		setInterval(() => {
			if (currentUser && document.getElementById('receive-files').classList.contains('active')) {
				loadAvailableFiles();
			}
		}, 8000);
        }
        
        // Add visual feedback for updates
        function showUpdateIndicator(elementId) {
            const element = document.getElementById(elementId);
            if (element) {
                element.style.transition = 'opacity 0.3s ease';
                element.style.opacity = '0.7';
                setTimeout(() => {
                    element.style.opacity = '1';
                }, 300);
            }
        }

		// Server-Sent Events for real-time log updates
		let logEventSource = null;
		window.logSSEActive = false;
		
		function setupLogSSE() {
			if (!currentUser) return;
			
			// Close existing connection
			if (logEventSource) {
				logEventSource.close();
			}
			
			try {
				logEventSource = new EventSource('/api/files/logs/stream');
				window.logSSEActive = true;
				
				logEventSource.onopen = function(event) {
					console.log('✅ SSE connection established for logs');
					showLogSSEStatus('connected');
				};
				
				logEventSource.onmessage = function(event) {
					try {
						const logEvent = JSON.parse(event.data);
						
						if (logEvent.type === 'log') {
							// Update the logs display if the file-logs section is active
							if (document.getElementById('file-logs').classList.contains('active')) {
								displayFileLogs(logEvent.data);
								showUpdateIndicator('fileLogArea');
								console.log('📊 Updated logs via SSE:', logEvent.data.total_logs + ' logs');
							}
						} else if (logEvent.type === 'heartbeat') {
							// Handle heartbeat to keep connection alive
							showLogSSEStatus('active');
						}
					} catch (error) {
						console.error('Error parsing SSE log event:', error);
					}
				};
				
				logEventSource.onerror = function(event) {
					console.error('❌ SSE connection error:', event);
					window.logSSEActive = false;
					showLogSSEStatus('error');
					
					// Reconnect after 5 seconds
					setTimeout(() => {
						if (currentUser) {
							console.log('🔄 Attempting to reconnect SSE...');
							setupLogSSE();
						}
					}, 5000);
				};
				
			} catch (error) {
				console.error('Failed to setup SSE:', error);
				window.logSSEActive = false;
			}
		}
		
		function showLogSSEStatus(status) {
			const statusElement = document.getElementById('logSSEStatus');
			if (statusElement) {
				const statusMap = {
					'connected': '🟢 Real-time updates connected',
					'active': '🔵 Real-time updates active',
					'error': '🔴 Real-time updates disconnected',
					'reconnecting': '🟡 Reconnecting...',
				};
				statusElement.textContent = statusMap[status] || status;
				statusElement.className = 'sse-status ' + status;
			}
		}
		
		// Close SSE connection when user logs out
		function closeLogSSE() {
			if (logEventSource) {
				logEventSource.close();
				logEventSource = null;
			}
			window.logSSEActive = false;
		}
		
		// Additional UI helper functions
		function editUser(userId) {
			// TODO: Implement user editing modal
			console.log('Edit user:', userId);
		}

        // File Reassembly Functions
        async function loadAvailableFiles() {
            const refreshBtn = document.getElementById('refreshFilesBtn');
            const filesList = document.getElementById('availableFilesList');
            
            // Show loading state
            refreshBtn.disabled = true;
            refreshBtn.innerHTML = '<i>🔄</i> Loading...';
            
            try {
                const response = await fetch('/api/files/available', {
                    method: 'GET',
                    credentials: 'include'
                });

                const result = await response.json();
                
                if (result.success) {
                    displayAvailableFiles(result.data.files);
                } else {
                    filesList.innerHTML = '<div class="status error">Failed to load files: ' + result.message + '</div>';
                }
            } catch (error) {
                filesList.innerHTML = '<div class="status error">Error loading files: ' + error.message + '</div>';
            } finally {
                refreshBtn.disabled = false;
                refreshBtn.innerHTML = '<i>🔄</i> Refresh Available Files';
            }
        }

        function displayAvailableFiles(files) {
            const filesList = document.getElementById('availableFilesList');
            
            if (!files || files.length === 0) {
                filesList.innerHTML = '<div class="text-center" style="padding: 40px; color: #6c757d;"><div style="font-size: 48px; margin-bottom: 15px;">📁</div><p>No files available for reassembly</p></div>';
                return;
            }

            let html = '';
            files.forEach(file => {
                const completeness = ((file.chunks_available / file.chunk_count) * 100).toFixed(1);
                const statusColor = completeness === '100.0' ? '#10B981' : completeness > '80' ? '#F59E0B' : '#EF4444';
                const canReassemble = completeness === '100.0';
                const isDemoFile = file.file_id.startsWith('demo-file-');
                const description = file.description || '';
                
                // Different icons for different file types
                let fileIcon = '📄';
                if (file.file_name.includes('.pdf')) fileIcon = '📑';
                else if (file.file_name.includes('.zip')) fileIcon = '🗜️';
                else if (file.file_name.includes('.mp4')) fileIcon = '🎥';
                else if (file.file_name.includes('.pptx')) fileIcon = '📊';
                
                html += '<div class="file-item" data-file-id="' + file.file_id + '" style="' + (isDemoFile ? 'border-left: 4px solid #17a2b8;' : '') + '">' +
                    '<div class="file-info">' +
                    '<h4>' + fileIcon + ' ' + file.file_name + (isDemoFile ? ' <span style="font-size: 0.75em; color: #17a2b8; font-weight: normal;">(Demo)</span>' : '') + '</h4>' +
                    (description ? '<p style="color: #6c757d; font-size: 0.85em; margin: 5px 0;">' + description + '</p>' : '') +
                    '<div class="file-details">' +
                    '<span>Size: ' + formatFileSize(file.file_size) + '</span> | ' +
                    '<span>Chunks: ' + file.chunks_available + '/' + file.chunk_count + ' (' + completeness + '%)</span> | ' +
                    '<span>Status: ' + file.status + '</span>' +
                    (file.created_at ? ' | <span>Created: ' + new Date(file.created_at).toLocaleDateString() + '</span>' : '') +
                    '<div style="margin-top: 5px;">' +
                    '<div class="progress" style="height: 6px; background: #e9ecef; border-radius: 3px; overflow: hidden;">' +
                    '<div class="progress-bar" style="width: ' + completeness + '%; background: ' + statusColor + '; height: 100%;"></div>' +
                    '</div></div></div></div>' +
                    '<div class="file-actions">' +
                    '<button class="btn btn-sm" onclick="reassembleFile(\'' + file.file_id + '\', \'' + file.file_name + '\')" ' +
                    (canReassemble ? '' : 'disabled') + '>' +
                    '🔧 ' + (canReassemble ? (isDemoFile ? 'Prepare' : 'Reassemble') : 'Incomplete') + '</button>' +
                    (canReassemble ? '<button class="btn btn-sm" onclick="downloadFile(\'' + file.file_id + '\', \'' + file.file_name + '\'")>📥 Download</button>' : '') +
                    '</div></div>';
            });
            
            filesList.innerHTML = html;
        }

        async function reassembleFile(fileId, fileName) {
            const progressDiv = document.getElementById('reassemblyProgress');
            const progressBar = document.getElementById('reassemblyProgressBar');
            const statusDiv = document.getElementById('reassemblyStatus');
            const password = (document.getElementById('reassemblyPassword') || {}).value || '';
            
            // Show progress area
            progressDiv.style.display = 'block';
            progressBar.style.width = '0%';
            statusDiv.innerHTML = '<div class="status info">Starting file reassembly...</div>';
            
            // Simulate reassembly progress
            let progress = 0;
            const progressInterval = setInterval(() => {
                progress += Math.random() * 20;
                if (progress > 95) progress = 95;
                
                progressBar.style.width = progress + '%';
                progressBar.textContent = Math.round(progress) + '%';
                
                if (progress > 30) statusDiv.innerHTML = '<div class="status info">Collecting chunks from network...</div>';
                if (progress > 60) statusDiv.innerHTML = '<div class="status info">Reconstructing file...</div>';
                if (progress > 85) statusDiv.innerHTML = '<div class="status info">Verifying integrity...</div>';
            }, 500);
            
            try {
                // Check if this is a demo file
                if (fileId.startsWith('demo-file-')) {
                    setTimeout(() => {
                        statusDiv.innerHTML = '<div class="status info">📦 Preparing demo file (' + fileName + ')...</div>';
                    }, 800);
                    
                    setTimeout(() => {
                        clearInterval(progressInterval);
                        progressBar.style.width = '100%';
                        progressBar.textContent = '100%';
                        statusDiv.innerHTML = '<div class="status success">✅ Demo file ' + fileName + ' is ready! <button class="btn btn-sm" onclick="downloadFile(\'' + fileId + '\', \'' + fileName + '\')" style="margin-left: 10px;">📥 Download Now</button></div>';
                        addToReassemblyHistory(fileId, fileName, 'demo-prepared');
                    }, 2500);
                } else {
                    // For real files, use the actual reassembly API
                    const response = await fetch('/api/dfs/reassemble', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        credentials: 'include',
                        body: JSON.stringify({
                            file_id: fileId,
                            output_path: './reassembled/' + fileName,
                            password: password
                        })
                    });

                    const result = await response.json();
                    
                    clearInterval(progressInterval);
                    
                    if (result.success) {
                        progressBar.style.width = '100%';
                        progressBar.textContent = '100%';
                        statusDiv.innerHTML = '<div class="status success">File reassembled successfully! <button class="btn btn-sm" onclick="downloadFile(\'' + fileId + '\', \'' + fileName + '\')" style="margin-left: 10px;">📥 Download Now</button></div>';
                        addToReassemblyHistory(fileId, fileName, 'completed');
                    } else {
                        statusDiv.innerHTML = '<div class="status error">Reassembly failed: ' + result.message + '</div>';
                    }
                }
            } catch (error) {
                clearInterval(progressInterval);
                statusDiv.innerHTML = '<div class="status error">Error during reassembly: ' + error.message + '</div>';
            }
        }

        async function downloadFile(fileId, fileName) {
            try {
                const password = (document.getElementById('reassemblyPassword') || {}).value || '';
                const link = document.createElement('a');
                link.href = '/api/files/download?file_id=' + encodeURIComponent(fileId) + (password ? '&password=' + encodeURIComponent(password) : '');
                link.download = fileName;
                document.body.appendChild(link);
                link.click();
                document.body.removeChild(link);
                
                const statusDiv = document.getElementById('reassemblyStatus');
                statusDiv.innerHTML = '<div class="status success">Download started for ' + fileName + '</div>';
                addToReassemblyHistory(fileId, fileName, 'downloaded');
                
            } catch (error) {
                console.error('Download error:', error);
                const statusDiv = document.getElementById('reassemblyStatus');
                statusDiv.innerHTML = '<div class="status error">Download failed: ' + error.message + '</div>';
            }
        }

        function addToReassemblyHistory(fileId, fileName, action) {
            const historyDiv = document.getElementById('reassemblyHistory');
            const timestamp = new Date().toLocaleString();
            
            let existingHistory = historyDiv.innerHTML;
            if (existingHistory.includes('Your file reassembly history will appear here')) {
                existingHistory = '';
            }
            
            const actionIcon = action === 'completed' ? '✅' : '📥';
            const actionText = action === 'completed' ? 'Reassembled' : 'Downloaded';
            
            const historyEntry = '<div class="file-item" style="margin-bottom: 10px; background: #f8f9fa;">' +
                '<div class="file-info">' +
                '<h5>' + actionIcon + ' ' + fileName + '</h5>' +
                '<div class="file-details">' +
                '<span>Action: ' + actionText + '</span> | ' +
                '<span>Time: ' + timestamp + '</span> | ' +
                '<span>File ID: ' + fileId + '</span>' +
                '</div></div></div>';
            
            historyDiv.innerHTML = historyEntry + existingHistory;
        }

        function formatFileSize(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }
    `
}
