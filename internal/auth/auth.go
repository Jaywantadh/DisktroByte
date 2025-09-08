package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// UserRole defines user roles in the system
type UserRole string

const (
	RoleUser        UserRole = "user"
	RoleSender      UserRole = "sender"
	RoleReceiver    UserRole = "receiver"
	RoleAdmin       UserRole = "admin"
	RoleSuperAdmin  UserRole = "superadmin"
)

// User represents a system user/node
type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	NodeID        string    `json:"node_id"`
	PasswordHash  string    `json:"password_hash"`
	Salt          string    `json:"salt"`
	Role          UserRole  `json:"role"`
	CreatedAt     time.Time `json:"created_at"`
	LastLogin     time.Time `json:"last_login"`
	IsActive      bool      `json:"is_active"`
	Profile       UserProfile `json:"profile"`
	Permissions   []string  `json:"permissions"`
	SessionToken  string    `json:"session_token,omitempty"`
}

// UserProfile contains user profile information
type UserProfile struct {
	DisplayName   string                 `json:"display_name"`
	Email         string                 `json:"email"`
	Organization  string                 `json:"organization"`
	Location      string                 `json:"location"`
	Preferences   map[string]interface{} `json:"preferences"`
	Avatar        string                 `json:"avatar"`
}

// Session represents an active user session
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Token        string    `json:"token"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastActivity time.Time `json:"last_activity"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	IsActive     bool      `json:"is_active"`
}

// AuthManager manages user authentication and sessions
type AuthManager struct {
	users        map[string]*User
	sessions     map[string]*Session
	usersByName  map[string]*User
	mu           sync.RWMutex
	sessionTTL   time.Duration
	maxSessions  int
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	NodeID   string `json:"node_id"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Success      bool      `json:"success"`
	Message      string    `json:"message"`
	Token        string    `json:"token,omitempty"`
	User         *User     `json:"user,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Permissions  []string  `json:"permissions,omitempty"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username     string      `json:"username"`
	Password     string      `json:"password"`
	NodeID       string      `json:"node_id"`
	Role         UserRole    `json:"role"`
	Profile      UserProfile `json:"profile"`
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(sessionTTL time.Duration, maxSessions int) *AuthManager {
	if sessionTTL <= 0 {
		sessionTTL = 24 * time.Hour // Default 24 hours
	}
	if maxSessions <= 0 {
		maxSessions = 100 // Default max 100 concurrent sessions
	}

	am := &AuthManager{
		users:       make(map[string]*User),
		sessions:    make(map[string]*Session),
		usersByName: make(map[string]*User),
		sessionTTL:  sessionTTL,
		maxSessions: maxSessions,
	}

	// Create default admin user
	am.createDefaultAdmin()

	// Start session cleanup routine
	go am.sessionCleanup()

	return am
}

// createDefaultAdmin creates the default admin user
func (am *AuthManager) createDefaultAdmin() {
	adminUser := &User{
		ID:       uuid.New().String(),
		Username: "admin",
		NodeID:   uuid.New().String(),
		Role:     RoleSuperAdmin,
		CreatedAt: time.Now(),
		IsActive:  true,
		Profile: UserProfile{
			DisplayName: "System Administrator",
			Email:       "admin@disktrobyte.local",
		},
		Permissions: []string{
			"system.manage",
			"users.manage",
			"files.manage",
			"network.manage",
			"logs.view",
			"stats.view",
			"config.manage",
		},
	}

	// Set default admin password
	salt, hash := am.hashPassword("admin123")
	adminUser.Salt = salt
	adminUser.PasswordHash = hash

	am.users[adminUser.ID] = adminUser
	am.usersByName[adminUser.Username] = adminUser

	fmt.Printf("🔐 Default admin user created (username: admin, password: admin123)\n")
}

// Register creates a new user account
func (am *AuthManager) Register(req RegisterRequest) (*User, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Validate input
	if req.Username == "" || req.Password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	if len(req.Password) < 6 {
		return nil, fmt.Errorf("password must be at least 6 characters")
	}

	// Check if username already exists
	if _, exists := am.usersByName[req.Username]; exists {
		return nil, fmt.Errorf("username already exists")
	}

	// Set default role if not specified
	if req.Role == "" {
		req.Role = RoleUser
	}

	// Generate node ID if not provided
	nodeID := req.NodeID
	if nodeID == "" {
		nodeID = uuid.New().String()
	}

	// Create new user
	user := &User{
		ID:        uuid.New().String(),
		Username:  req.Username,
		NodeID:    nodeID,
		Role:      req.Role,
		CreatedAt: time.Now(),
		IsActive:  true,
		Profile:   req.Profile,
		Permissions: am.getDefaultPermissions(req.Role),
	}

	// Hash password
	salt, hash := am.hashPassword(req.Password)
	user.Salt = salt
	user.PasswordHash = hash

	// Store user
	am.users[user.ID] = user
	am.usersByName[user.Username] = user

	fmt.Printf("👤 New user registered: %s (Role: %s, Node: %s)\n", 
		user.Username, user.Role, user.NodeID)

	return user, nil
}

// Login authenticates a user and creates a session
func (am *AuthManager) Login(req LoginRequest) (*LoginResponse, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Find user by username
	user, exists := am.usersByName[req.Username]
	if !exists || !user.IsActive {
		return &LoginResponse{
			Success: false,
			Message: "Invalid username or password",
		}, nil
	}

	// Verify password
	if !am.verifyPassword(req.Password, user.Salt, user.PasswordHash) {
		return &LoginResponse{
			Success: false,
			Message: "Invalid username or password",
		}, nil
	}

	// Update node ID if provided
	if req.NodeID != "" && req.NodeID != user.NodeID {
		user.NodeID = req.NodeID
	}

	// Create session
	session, err := am.createSession(user)
	if err != nil {
		return &LoginResponse{
			Success: false,
			Message: "Failed to create session: " + err.Error(),
		}, nil
	}

	// Update last login
	user.LastLogin = time.Now()
	user.SessionToken = session.Token

	fmt.Printf("🔓 User logged in: %s (Role: %s, Session: %s)\n", 
		user.Username, user.Role, session.ID)

	return &LoginResponse{
		Success:     true,
		Message:     "Login successful",
		Token:       session.Token,
		User:        user,
		ExpiresAt:   session.ExpiresAt,
		Permissions: user.Permissions,
	}, nil
}

// Logout invalidates a user session
func (am *AuthManager) Logout(token string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	session, exists := am.sessions[token]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Mark session as inactive
	session.IsActive = false

	// Clear user session token
	if user, exists := am.users[session.UserID]; exists {
		user.SessionToken = ""
	}

	delete(am.sessions, token)

	fmt.Printf("🔒 User logged out: Session %s\n", session.ID)
	return nil
}

// ValidateSession validates a session token
func (am *AuthManager) ValidateSession(token string) (*User, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	session, exists := am.sessions[token]
	if !exists || !session.IsActive {
		return nil, fmt.Errorf("invalid or expired session")
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Get user
	user, exists := am.users[session.UserID]
	if !exists || !user.IsActive {
		return nil, fmt.Errorf("user not found or inactive")
	}

	// Update last activity
	session.LastActivity = time.Now()

	return user, nil
}

// GetUserByID retrieves a user by ID
func (am *AuthManager) GetUserByID(userID string) (*User, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	user, exists := am.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

// GetAllUsers returns all users (admin only)
func (am *AuthManager) GetAllUsers() []*User {
	am.mu.RLock()
	defer am.mu.RUnlock()

	users := make([]*User, 0, len(am.users))
	for _, user := range am.users {
		users = append(users, user)
	}

	return users
}

// UpdateUser updates user information
func (am *AuthManager) UpdateUser(userID string, updates map[string]interface{}) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	user, exists := am.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}

	// Update fields
	for field, value := range updates {
		switch field {
		case "display_name":
			if v, ok := value.(string); ok {
				user.Profile.DisplayName = v
			}
		case "email":
			if v, ok := value.(string); ok {
				user.Profile.Email = v
			}
		case "role":
			if v, ok := value.(string); ok {
				user.Role = UserRole(v)
				user.Permissions = am.getDefaultPermissions(user.Role)
			}
		case "is_active":
			if v, ok := value.(bool); ok {
				user.IsActive = v
			}
		}
	}

	fmt.Printf("👤 User updated: %s\n", user.Username)
	return nil
}

// createSession creates a new session for a user
func (am *AuthManager) createSession(user *User) (*Session, error) {
	// Check session limit
	if len(am.sessions) >= am.maxSessions {
		return nil, fmt.Errorf("maximum sessions reached")
	}

	// Generate session token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	session := &Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		Token:        token,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(am.sessionTTL),
		LastActivity: time.Now(),
		IsActive:     true,
	}

	am.sessions[token] = session
	return session, nil
}

// hashPassword creates a salted hash of the password
func (am *AuthManager) hashPassword(password string) (string, string) {
	// Generate random salt
	saltBytes := make([]byte, 16)
	rand.Read(saltBytes)
	salt := hex.EncodeToString(saltBytes)

	// Create hash
	hasher := sha256.New()
	hasher.Write([]byte(password + salt))
	hash := hex.EncodeToString(hasher.Sum(nil))

	return salt, hash
}

// verifyPassword verifies a password against its hash
func (am *AuthManager) verifyPassword(password, salt, hash string) bool {
	hasher := sha256.New()
	hasher.Write([]byte(password + salt))
	computedHash := hex.EncodeToString(hasher.Sum(nil))
	return computedHash == hash
}

// getDefaultPermissions returns default permissions for a role
func (am *AuthManager) getDefaultPermissions(role UserRole) []string {
	switch role {
	case RoleSuperAdmin:
		return []string{
			"system.manage",
			"users.manage",
			"files.manage",
			"network.manage",
			"logs.view",
			"stats.view",
			"config.manage",
		}
	case RoleAdmin:
		return []string{
			"users.view",
			"files.manage",
			"network.view",
			"logs.view",
			"stats.view",
		}
	case RoleSender:
		return []string{
			"files.send",
			"files.view",
			"network.view",
			"logs.view.own",
		}
	case RoleReceiver:
		return []string{
			"files.receive",
			"network.view",
			"logs.view.own",
		}
	case RoleUser:
	default:
		return []string{
			"files.send",
			"files.receive",
			"network.view.limited",
		}
	}
	return []string{}
}

// sessionCleanup removes expired sessions
func (am *AuthManager) sessionCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		am.mu.Lock()
		now := time.Now()
		expired := 0

		for token, session := range am.sessions {
			if now.After(session.ExpiresAt) || !session.IsActive {
				delete(am.sessions, token)
				expired++
			}
		}

		am.mu.Unlock()

		if expired > 0 {
			fmt.Printf("🧹 Cleaned up %d expired sessions\n", expired)
		}
	}
}

// HasPermission checks if a user has a specific permission
func (am *AuthManager) HasPermission(user *User, permission string) bool {
	for _, perm := range user.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// GetUserStats returns user statistics
func (am *AuthManager) GetUserStats() map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	stats := map[string]interface{}{
		"total_users":    len(am.users),
		"active_sessions": len(am.sessions),
		"roles": map[string]int{
			"admin":      0,
			"superadmin": 0,
			"sender":     0,
			"receiver":   0,
			"user":       0,
		},
	}

	for _, user := range am.users {
		if count, ok := stats["roles"].(map[string]int)[string(user.Role)]; ok {
			stats["roles"].(map[string]int)[string(user.Role)] = count + 1
		}
	}

	return stats
}

// ExportUsers exports user data (admin only)
func (am *AuthManager) ExportUsers() ([]byte, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Create export structure without sensitive data
	export := make([]map[string]interface{}, 0, len(am.users))
	for _, user := range am.users {
		userData := map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"node_id":    user.NodeID,
			"role":       user.Role,
			"created_at": user.CreatedAt,
			"last_login": user.LastLogin,
			"is_active":  user.IsActive,
			"profile":    user.Profile,
		}
		export = append(export, userData)
	}

	return json.MarshalIndent(export, "", "  ")
}
