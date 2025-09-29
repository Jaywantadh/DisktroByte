package main

func getCSS() string {
	return `
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        :root {
            --primary-color: #667eea;
            --secondary-color: #764ba2;
            --accent-color: #f093fb;
            --success-color: #10B981;
            --warning-color: #F59E0B;
            --danger-color: #EF4444;
            --info-color: #3B82F6;
            --dark-color: #1F2937;
            --light-color: #F9FAFB;
            --text-color: #374151;
            --border-radius: 12px;
            --box-shadow: 0 10px 25px rgba(0, 0, 0, 0.1);
            --glassmorphism: rgba(255, 255, 255, 0.25);
            --backdrop-filter: blur(10px);
        }

        body {
            font-family: 'Inter', 'Segoe UI', system-ui, -apple-system, sans-serif;
            background: linear-gradient(135deg, var(--primary-color) 0%, var(--accent-color) 50%, var(--secondary-color) 100%);
            background-size: 400% 400%;
            animation: gradientShift 15s ease infinite;
            min-height: 100vh;
            margin: 0;
            color: var(--text-color);
        }
        
        @keyframes gradientShift {
            0% { background-position: 0% 50%; }
            50% { background-position: 100% 50%; }
            100% { background-position: 0% 50%; }
        }

        /* Login Screen Styles */
        .login-container {
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            padding: 20px;
        }

        .login-card {
            background: var(--glassmorphism);
            backdrop-filter: var(--backdrop-filter);
            border: 1px solid rgba(255, 255, 255, 0.3);
            border-radius: 20px;
            box-shadow: 0 25px 50px rgba(0,0,0,0.15);
            padding: 50px;
            width: 100%;
            max-width: 450px;
            text-align: center;
            transform: translateY(0);
            transition: all 0.6s cubic-bezier(0.165, 0.84, 0.44, 1);
            position: relative;
            overflow: hidden;
        }
        
        .login-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: linear-gradient(90deg, transparent, rgba(255,255,255,0.2), transparent);
            transition: left 0.5s;
        }
        
        .login-card:hover::before {
            left: 100%;
        }
        
        .login-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 35px 70px rgba(0,0,0,0.2);
        }

        .login-header {
            margin-bottom: 30px;
        }

        .login-header h1 {
            color: var(--dark-color);
            font-size: 2.5em;
            margin-bottom: 10px;
        }

        .login-header p {
            color: #6c757d;
            font-size: 1.1em;
        }

        .form-group {
            margin-bottom: 20px;
            text-align: left;
        }

        .form-group label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #495057;
        }

        .form-control {
            width: 100%;
            padding: 16px 20px;
            border: 2px solid rgba(255, 255, 255, 0.3);
            border-radius: var(--border-radius);
            font-size: 16px;
            background: rgba(255, 255, 255, 0.9);
            backdrop-filter: blur(5px);
            transition: all 0.4s cubic-bezier(0.165, 0.84, 0.44, 1);
            color: var(--dark-color);
        }

        .form-control:focus {
            outline: none;
            border-color: var(--primary-color);
            background: rgba(255, 255, 255, 0.95);
            box-shadow: 0 0 0 4px rgba(102, 126, 234, 0.15), 0 10px 20px rgba(0,0,0,0.1);
            transform: translateY(-2px);
        }
        
        .form-control::placeholder {
            color: rgba(55, 65, 81, 0.6);
            transition: color 0.3s;
        }

        .form-select {
            width: 100%;
            padding: 12px 15px;
            border: 2px solid #e9ecef;
            border-radius: var(--border-radius);
            font-size: 16px;
            background: white;
        }

        .btn {
            background: linear-gradient(135deg, var(--primary-color) 0%, var(--accent-color) 50%, var(--secondary-color) 100%);
            background-size: 200% 200%;
            color: white;
            border: none;
            padding: 16px 32px;
            border-radius: var(--border-radius);
            cursor: pointer;
            font-size: 16px;
            font-weight: 600;
            transition: all 0.4s cubic-bezier(0.165, 0.84, 0.44, 1);
            text-decoration: none;
            display: inline-block;
            position: relative;
            overflow: hidden;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        
        .btn::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: linear-gradient(135deg, transparent, rgba(255,255,255,0.2), transparent);
            transform: translateX(-100%);
            transition: transform 0.6s;
        }
        
        .btn:hover::before {
            transform: translateX(100%);
        }

        .btn:hover {
            transform: translateY(-3px) scale(1.02);
            background-position: 100% 0;
            box-shadow: 0 15px 30px rgba(102, 126, 234, 0.4), 0 5px 15px rgba(0,0,0,0.1);
        }
        
        .btn:active {
            transform: translateY(-1px) scale(0.98);
        }

        .btn:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none;
        }

        .btn-primary {
            width: 100%;
            margin-bottom: 15px;
        }

        .btn-link {
            background: none;
            color: var(--primary-color);
            text-decoration: underline;
            font-weight: normal;
        }

        .btn-link:hover {
            transform: none;
            box-shadow: none;
        }

        /* Main App Styles */
        .app-container {
            display: flex;
            min-height: 100vh;
        }

        .sidebar {
            width: 280px;
            background: rgba(31, 41, 55, 0.95);
            backdrop-filter: var(--backdrop-filter);
            border-right: 1px solid rgba(255, 255, 255, 0.1);
            color: white;
            padding: 0;
            overflow-y: auto;
            position: relative;
        }
        
        .sidebar::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: linear-gradient(180deg, rgba(102, 126, 234, 0.1), rgba(118, 75, 162, 0.1));
            z-index: -1;
        }

        .sidebar-header {
            padding: 20px;
            border-bottom: 1px solid #34495e;
            text-align: center;
        }

        .sidebar-header h2 {
            margin-bottom: 5px;
        }

        .sidebar-header .user-info {
            font-size: 0.9em;
            opacity: 0.8;
        }

        .nav-menu {
            list-style: none;
            padding: 20px 0;
        }

        .nav-item {
            margin: 0;
        }

        .nav-link {
            display: block;
            padding: 18px 25px;
            color: rgba(255, 255, 255, 0.9);
            text-decoration: none;
            transition: all 0.4s cubic-bezier(0.165, 0.84, 0.44, 1);
            border-left: 4px solid transparent;
            position: relative;
            overflow: hidden;
        }
        
        .nav-link::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 4px;
            height: 100%;
            background: linear-gradient(135deg, var(--primary-color), var(--accent-color));
            transform: scaleY(0);
            transition: transform 0.3s;
        }

        .nav-link:hover,
        .nav-link.active {
            background: rgba(255, 255, 255, 0.15);
            color: white;
            padding-left: 35px;
            border-left-color: var(--primary-color);
        }
        
        .nav-link:hover::before,
        .nav-link.active::before {
            transform: scaleY(1);
        }

        .nav-link i {
            margin-right: 10px;
            width: 20px;
        }

        .main-content {
            flex: 1;
            background: #f8f9fa;
            padding: 0;
            overflow-y: auto;
        }

        .content-header {
            background: white;
            padding: 20px 30px;
            border-bottom: 1px solid #e9ecef;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .content-header h1 {
            color: var(--dark-color);
            margin: 0;
        }

        .logout-btn {
            background: var(--danger-color);
            padding: 8px 16px;
            font-size: 14px;
        }

        .content-body {
            padding: 30px;
        }

        .card {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(20px);
            border: 1px solid rgba(255, 255, 255, 0.3);
            border-radius: var(--border-radius);
            box-shadow: var(--box-shadow);
            margin-bottom: 30px;
            transition: all 0.4s cubic-bezier(0.165, 0.84, 0.44, 1);
            position: relative;
            overflow: hidden;
        }
        
        .card::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 2px;
            background: linear-gradient(90deg, var(--primary-color), var(--accent-color), var(--secondary-color));
            transition: left 0.5s;
        }
        
        .card:hover::before {
            left: 100%;
        }
        
        .card:hover {
            transform: translateY(-5px);
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
        }

        .card-header {
            background: #f8f9fa;
            border-bottom: 1px solid #e9ecef;
            padding: 20px;
            border-radius: var(--border-radius) var(--border-radius) 0 0;
        }

        .card-body {
            padding: 20px;
        }

        .card-title {
            margin: 0;
            color: var(--dark-color);
        }

        /* Role-specific content areas */
        .content-area {
            display: none;
        }

        .content-area.active {
            display: block;
        }

        /* File management styles */
        .file-list {
            margin-top: 20px;
        }

        .file-item {
            background: white;
            border: 1px solid #e9ecef;
            border-radius: var(--border-radius);
            padding: 15px;
            margin-bottom: 15px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .file-info h4 {
            margin: 0 0 5px 0;
            color: var(--dark-color);
        }

        .file-details {
            font-size: 0.9em;
            color: #6c757d;
        }

        .file-actions {
            display: flex;
            gap: 10px;
        }

        .btn-sm {
            padding: 6px 12px;
            font-size: 14px;
        }

        /* Progress bars */
        .progress {
            width: 100%;
            height: 20px;
            background: #e9ecef;
            border-radius: 10px;
            overflow: hidden;
            margin: 15px 0;
        }

        .progress-bar {
            height: 100%;
            background: linear-gradient(135deg, var(--primary-color) 0%, var(--accent-color) 50%, var(--secondary-color) 100%);
            background-size: 200% 100%;
            animation: progressShimmer 2s ease-in-out infinite;
            width: 0%;
            transition: width 0.6s cubic-bezier(0.165, 0.84, 0.44, 1);
            text-align: center;
            line-height: 20px;
            color: white;
            font-size: 12px;
            position: relative;
            overflow: hidden;
        }
        
        .progress-bar::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: linear-gradient(90deg, transparent, rgba(255,255,255,0.3), transparent);
            animation: progressGlow 1.5s ease-in-out infinite;
        }
        
        @keyframes progressShimmer {
            0% { background-position: 200% 0; }
            100% { background-position: -200% 0; }
        }
        
        @keyframes progressGlow {
            0%, 100% { transform: translateX(-100%); }
            50% { transform: translateX(100%); }
        }

        /* Status indicators */
        .status {
            padding: 15px;
            border-radius: var(--border-radius);
            margin: 15px 0;
        }

        .status.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }

        .status.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }

        .status.warning {
            background: #fff3cd;
            color: #856404;
            border: 1px solid #ffeaa7;
        }

        .status.info {
            background: #d1ecf1;
            color: #0c5460;
            border: 1px solid #bee5eb;
        }

        /* Tables */
        .table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }

        .table th,
        .table td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #e9ecef;
        }

        .table th {
            background: #f8f9fa;
            font-weight: 600;
            color: var(--dark-color);
        }

        .table tbody tr:hover {
            background: #f8f9fa;
        }

        /* Admin specific styles */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }

        .stat-card {
            background: white;
            border-radius: var(--border-radius);
            padding: 20px;
            text-align: center;
            box-shadow: var(--box-shadow);
        }

        .stat-number {
            font-size: 2.5em;
            font-weight: bold;
            color: var(--primary-color);
            margin-bottom: 10px;
        }

        .stat-label {
            color: #6c757d;
            font-size: 0.9em;
        }

        /* Log area */
        .log-area {
            background: #2c3e50;
            color: #ecf0f1;
            border-radius: var(--border-radius);
            padding: 20px;
            height: 300px;
            overflow-y: auto;
            font-family: 'Courier New', monospace;
            font-size: 14px;
            margin-top: 20px;
        }

        .log-entry {
            margin-bottom: 5px;
            padding: 5px 10px;
            border-radius: 4px;
            white-space: pre-wrap;
        }

        .log-entry.info {
            background: rgba(23, 162, 184, 0.2);
            border-left: 3px solid var(--info-color);
        }

        .log-entry.success {
            background: rgba(40, 167, 69, 0.2);
            border-left: 3px solid var(--success-color);
        }

        .log-entry.error {
            background: rgba(220, 53, 69, 0.2);
            border-left: 3px solid var(--danger-color);
        }

        .log-entry.warning {
            background: rgba(255, 193, 7, 0.2);
            border-left: 3px solid var(--warning-color);
        }

        /* Responsive design */
        @media (max-width: 768px) {
            .app-container {
                flex-direction: column;
            }

            .sidebar {
                width: 100%;
                height: auto;
            }

            .nav-menu {
                display: flex;
                overflow-x: auto;
                padding: 10px 0;
            }

            .nav-item {
                min-width: max-content;
            }

            .content-body {
                padding: 15px;
            }

            .stats-grid {
                grid-template-columns: 1fr;
            }
        }

        /* Animations */
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .fade-in {
            animation: fadeIn 0.3s ease-out;
        }

        /* Modal styles */
        .modal {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.5);
        }

        .modal-content {
            background-color: white;
            margin: 10% auto;
            padding: 0;
            border-radius: var(--border-radius);
            width: 90%;
            max-width: 500px;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
        }

        .modal-header {
            background: var(--dark-color);
            color: white;
            padding: 20px;
            border-radius: var(--border-radius) var(--border-radius) 0 0;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .modal-body {
            padding: 20px;
        }

        .modal-footer {
            padding: 15px 20px;
            border-top: 1px solid #e9ecef;
            display: flex;
            justify-content: flex-end;
            gap: 10px;
        }

        .close {
            background: none;
            border: none;
            color: white;
            font-size: 24px;
            cursor: pointer;
        }

        /* Network status indicators */
        .status-indicator {
            display: inline-block;
            width: 14px;
            height: 14px;
            border-radius: 50%;
            margin-right: 10px;
            position: relative;
            transition: all 0.3s;
        }
        
        .status-indicator::before {
            content: '';
            position: absolute;
            top: -2px;
            left: -2px;
            right: -2px;
            bottom: -2px;
            border-radius: 50%;
            opacity: 0;
            transition: all 0.3s;
        }

        .status-online {
            background-color: var(--success-color);
            box-shadow: 0 0 10px rgba(16, 185, 129, 0.3);
            animation: pulseGreen 2s infinite;
        }
        
        .status-online::before {
            background-color: var(--success-color);
            animation: rippleGreen 1.5s infinite;
        }

        .status-offline {
            background-color: var(--danger-color);
            box-shadow: 0 0 10px rgba(239, 68, 68, 0.3);
        }

        .status-connecting {
            background-color: var(--warning-color);
            box-shadow: 0 0 10px rgba(245, 158, 11, 0.3);
            animation: pulseYellow 1s infinite;
        }
        
        @keyframes pulseGreen {
            0%, 100% { box-shadow: 0 0 10px rgba(16, 185, 129, 0.3); }
            50% { box-shadow: 0 0 20px rgba(16, 185, 129, 0.6), 0 0 30px rgba(16, 185, 129, 0.3); }
        }
        
        @keyframes pulseYellow {
            0%, 100% { box-shadow: 0 0 10px rgba(245, 158, 11, 0.3); }
            50% { box-shadow: 0 0 20px rgba(245, 158, 11, 0.6), 0 0 30px rgba(245, 158, 11, 0.3); }
        }
        
        @keyframes rippleGreen {
            0% { transform: scale(1); opacity: 1; }
            100% { transform: scale(2); opacity: 0; }
        }

        /* File upload drag and drop */
        .upload-area {
            border: 2px dashed #e9ecef;
            border-radius: var(--border-radius);
            padding: 40px;
            text-align: center;
            margin: 20px 0;
            transition: border-color 0.3s, background-color 0.3s;
        }

        .upload-area.dragover {
            border-color: var(--primary-color);
            background: rgba(102, 126, 234, 0.05);
        }

        .upload-area input[type="file"] {
            display: none;
        }

        .upload-icon {
            font-size: 48px;
            color: #6c757d;
            margin-bottom: 15px;
        }
    `
}

func getLoginHTML() string {
	return `
        <div class="login-card fade-in">
            <div class="login-header">
                <h1>🚀 DisktroByte</h1>
                <p>P2P Distributed File System</p>
            </div>
            
            <div id="loginForm">
                <div class="form-group">
                    <label for="username">Username</label>
                    <input type="text" id="username" class="form-control" placeholder="Enter your username" required>
                </div>
                
                <div class="form-group">
                    <label for="password">Password</label>
                    <input type="password" id="password" class="form-control" placeholder="Enter your password" required>
                </div>
                
                <div class="form-group">
                    <label for="nodeId">Node ID (Optional)</label>
                    <input type="text" id="nodeId" class="form-control" placeholder="Auto-generated if empty">
                </div>
                
                <button type="button" class="btn btn-primary" onclick="login()">Sign In</button>
                <button type="button" class="btn btn-link" onclick="showRegisterForm()">Create New Account</button>
            </div>
            
            <div id="registerForm" style="display: none;">
                <div class="form-group">
                    <label for="regUsername">Username</label>
                    <input type="text" id="regUsername" class="form-control" placeholder="Choose a username" required>
                </div>
                
                <div class="form-group">
                    <label for="regPassword">Password</label>
                    <input type="password" id="regPassword" class="form-control" placeholder="Choose a password" required>
                </div>
                
                <div class="form-group">
                    <label for="regDisplayName">Display Name</label>
                    <input type="text" id="regDisplayName" class="form-control" placeholder="Your display name">
                </div>
                
                <div class="form-group">
                    <label for="regEmail">Email</label>
                    <input type="email" id="regEmail" class="form-control" placeholder="Your email address">
                </div>
                
                <div class="form-group">
                    <label for="regRole">Role</label>
                    <select id="regRole" class="form-select">
                        <option value="user">User</option>
                        <option value="sender">Sender</option>
                        <option value="receiver">Receiver</option>
                    </select>
                </div>
                
                <button type="button" class="btn btn-primary" onclick="register()">Create Account</button>
                <button type="button" class="btn btn-link" onclick="showLoginForm()">Back to Sign In</button>
            </div>
            
            <div id="loginStatus" class="status" style="display: none;"></div>
        </div>
    `
}

func getMainAppHTML() string {
	return `
        <div class="sidebar">
            <div class="sidebar-header">
                <h2>DisktroByte</h2>
                <div class="user-info">
                    <div id="userDisplayName">Loading...</div>
                    <div id="userRole">...</div>
                </div>
            </div>
            
            <ul class="nav-menu">
                <li class="nav-item">
                    <a href="#" class="nav-link active" data-section="dashboard">
                        <i>🏠</i> Dashboard
                    </a>
                </li>
                <li class="nav-item" id="senderNav" style="display: none;">
                    <a href="#" class="nav-link" data-section="send-files">
                        <i>📤</i> Send Files
                    </a>
                </li>
                <li class="nav-item" id="receiverNav" style="display: none;">
                    <a href="#" class="nav-link" data-section="receive-files">
                        <i>📥</i> Receive Files
                    </a>
                </li>
                <li class="nav-item">
                    <a href="#" class="nav-link" data-section="file-logs">
                        <i>📋</i> File Logs
                    </a>
                </li>
                <li class="nav-item">
                    <a href="#" class="nav-link" data-section="file-reassembly">
                        <i>🔧</i> File Reassembly
                    </a>
                </li>
                <li class="nav-item">
                    <a href="#" class="nav-link" data-section="network">
                        <i>🌐</i> Network
                    </a>
                </li>
                <li class="nav-item">
                    <a href="#" class="nav-link" data-section="streaming">
                        <i>📡</i> Streaming
                    </a>
                </li>
                <li class="nav-item" id="adminNav" style="display: none;">
                    <a href="#" class="nav-link" data-section="admin">
                        <i>⚙️</i> Administration
                    </a>
                </li>
            </ul>
        </div>
        
        <div class="main-content">
            <div class="content-header">
                <h1 id="contentTitle">Dashboard</h1>
                <button class="btn logout-btn" onclick="logout()">Logout</button>
            </div>
            
            <div class="content-body">
                <div id="dashboard" class="content-area active">
                    <div class="stats-grid">
                        <div class="stat-card">
                            <div class="stat-number" id="totalFiles">0</div>
                            <div class="stat-label">Total Files</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-number" id="activePeers">0</div>
                            <div class="stat-label">Connected Peers</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-number" id="streamingSessions">0</div>
                            <div class="stat-label">Active Streams</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-number" id="totalStorage">0 MB</div>
                            <div class="stat-label">Storage Used</div>
                        </div>
                    </div>
                    
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">System Status</h3>
                        </div>
                        <div class="card-body">
                            <p><span class="status-indicator status-online"></span> HTTP P2P Network: Online</p>
                            <p><span class="status-indicator status-online"></span> TCP P2P Network: Online</p>
                            <p><span class="status-indicator status-online"></span> Broadcast System: Online</p>
                            <p><span class="status-indicator status-online"></span> Stream Processor: Online</p>
                        </div>
                    </div>
                </div>
                
                <div id="send-files" class="content-area">
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">Send Files</h3>
                        </div>
                        <div class="card-body">
                            <div class="upload-area" onclick="document.getElementById('fileInput').click()">
                                <div class="upload-icon">📁</div>
                                <p>Click here or drag and drop files to send</p>
                                <input type="file" id="fileInput" multiple>
                            </div>
                            
                            <div class="form-group">
                                <label for="filePassword">Encryption Password</label>
                                <input type="password" id="filePassword" class="form-control" placeholder="Enter encryption password">
                            </div>
                            
                            <!-- File Queue Display -->
                            <div id="fileQueue" class="card" style="display: none; margin: 20px 0;">
                                <div class="card-header">
                                    <h4 class="card-title">📋 Selected Files Queue</h4>
                                    <button class="btn btn-sm" onclick="clearFileQueue()" style="float: right; background: #ef4444; color: white;">Clear All</button>
                                </div>
                                <div class="card-body">
                                    <div id="fileQueueList"></div>
                                    <div style="margin-top: 15px; padding-top: 15px; border-top: 1px solid #e9ecef;">
                                        <strong>Total: <span id="queueTotalFiles">0</span> files (<span id="queueTotalSize">0 MB</span>)</strong>
                                    </div>
                                </div>
                            </div>
                            
                            <button class="btn btn-primary" onclick="sendFiles()" id="sendFilesBtn" disabled>Send Files</button>
                            
                            <div id="sendProgress" class="progress" style="display: none;">
                                <div id="sendProgressBar" class="progress-bar"></div>
                            </div>
                            
                            <div id="sendStatus" class="status" style="display: none;"></div>
                        </div>
                    </div>
                    
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">Sent Files Log</h3>
                        </div>
                        <div class="card-body">
                            <div id="sentFilesList"></div>
                        </div>
                    </div>
                </div>
                
                <div id="receive-files" class="content-area">
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">Received Files</h3>
                        </div>
                        <div class="card-body">
                            <p class="text-muted">Files are received automatically and stored securely. You cannot see details of received files for privacy protection.</p>
                            <div id="receivedFilesList"></div>
                        </div>
                    </div>
                </div>
                
                <div id="file-logs" class="content-area">
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">File Operation Logs</h3>
                        </div>
                        <div class="card-body">
                            <div class="log-area" id="fileLogArea"></div>
                        </div>
                    </div>
                </div>
                
                <div id="file-reassembly" class="content-area">
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">🔧 File Reassembly</h3>
                        </div>
                        <div class="card-body">
                            <p class="text-muted">Browse and reassemble distributed files from the network. Select files to download and reconstruct them from their chunks.</p>
                            
                            <div class="form-group" style="margin-bottom: 12px;">
                                <label for="reassemblyPassword">Decryption Password</label>
                                <input type="password" id="reassemblyPassword" class="form-control" placeholder="Enter the same password used during upload">
                            </div>
                            
                            <div class="form-group" style="margin-bottom: 20px;">
                                <button class="btn btn-primary" onclick="loadAvailableFiles()" id="refreshFilesBtn">
                                    <i>🔄</i> Refresh Available Files
                                </button>
                            </div>
                            
                            <div id="availableFilesList" class="file-list">
                                <div class="text-center" style="padding: 40px; color: #6c757d;">
                                    <div style="font-size: 48px; margin-bottom: 15px;">📁</div>
                                    <p>Click "Refresh Available Files" to load files that can be reassembled</p>
                                </div>
                            </div>
                            
                            <div id="reassemblyProgress" style="display: none;">
                                <h4>Reassembly Progress</h4>
                                <div class="progress">
                                    <div id="reassemblyProgressBar" class="progress-bar"></div>
                                </div>
                                <div id="reassemblyStatus" class="status"></div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">📋 Reassembly History</h3>
                        </div>
                        <div class="card-body">
                            <div id="reassemblyHistory">
                                <p class="text-muted">Your file reassembly history will appear here.</p>
                            </div>
                        </div>
                    </div>
                </div>
                
                <div id="network" class="content-area">
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">Network Status</h3>
                        </div>
                        <div class="card-body">
                            <button class="btn btn-primary" onclick="refreshNetwork()">Refresh Network</button>
                            <div id="networkStatus"></div>
                        </div>
                    </div>
                </div>
                
                <div id="streaming" class="content-area">
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">Active Streams</h3>
                        </div>
                        <div class="card-body">
                            <div id="activeStreams"></div>
                        </div>
                    </div>
                </div>
                
                <div id="admin" class="content-area">
                    <div class="card">
                        <div class="card-header">
                            <h3 class="card-title">System Administration</h3>
                        </div>
                        <div class="card-body">
                            <div class="stats-grid">
                                <div class="stat-card">
                                    <div class="stat-number" id="adminTotalUsers">0</div>
                                    <div class="stat-label">Total Users</div>
                                </div>
                                <div class="stat-card">
                                    <div class="stat-number" id="adminActiveSessions">0</div>
                                    <div class="stat-label">Active Sessions</div>
                                </div>
                                <div class="stat-card">
                                    <div class="stat-number" id="adminTotalFiles">0</div>
                                    <div class="stat-label">System Files</div>
                                </div>
                                <div class="stat-card">
                                    <div class="stat-number" id="adminNetworkNodes">0</div>
                                    <div class="stat-label">Network Nodes</div>
                                </div>
                            </div>
                            
                            <div class="card">
                                <div class="card-header">
                                    <h4>User Management</h4>
                                </div>
                                <div class="card-body">
                                    <button class="btn btn-primary" onclick="loadUsers()">Load Users</button>
                                    <div id="usersList"></div>
                                </div>
                            </div>
                            
                            <div class="card">
                                <div class="card-header">
                                    <h4>System Logs</h4>
                                </div>
                                <div class="card-body">
                                    <div class="log-area" id="systemLogArea"></div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `
}
