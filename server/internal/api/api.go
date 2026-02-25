package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	l "github.com/bkenks/bs3-logger"
	"github.com/bkenks/bs3/internal/cryptoutil"
	"github.com/bkenks/bs3/internal/vault"
	"github.com/google/uuid"
)

// =====================================================
// API Server Object
// =====================================================

type Server struct {
	Vault        *vault.Vault
	APITokenHash []byte
	TokenExpiry  int64
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/store", s.authMiddleware(s.StoreSecret))
	mux.HandleFunc("/get", s.authMiddleware(s.GetSecret))
	mux.HandleFunc("/delete", s.authMiddleware(s.DeleteSecret))
	mux.HandleFunc("/listsecrets", s.authMiddleware(s.ListSecrets))
	mux.HandleFunc("/token", s.authMiddleware(s.GenerateToken))
	mux.HandleFunc("/deletetoken", s.authMiddleware(s.DeleteToken))
	mux.HandleFunc("/listtokens", s.authMiddleware(s.ListTokens))
	mux.HandleFunc("/adduser", s.authMiddleware(s.AddUser))
	mux.HandleFunc("/deleteuser", s.authMiddleware(s.DeleteUser))
	mux.HandleFunc("/listusers", s.authMiddleware(s.ListUsers))
	mux.HandleFunc("/initvault", s.authMiddleware(s.InitializeVault))
	mux.HandleFunc("/openvault", s.authMiddleware(s.OpenVault))
}

// =====================================================
// END "API Server Object"
// =====================================================

// =====================================================
// Authentication
// =====================================================
// function for handling authentication of http requests
// all requests wrapped with this logic

// ~~~ authMiddleware ~~~
// function between http requests to verify the authenticity of requester
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		bearerStr := "Bearer "

		// Token Verification
		if strings.HasPrefix(auth, bearerStr) {
			urlSafeToken := strings.TrimPrefix(auth, bearerStr)
			rawToken, err := base64.RawURLEncoding.DecodeString(urlSafeToken)
			if err != nil {
				writeError(
					w,
					http.StatusUnauthorized,
					"unauthorized",
					"could not decode token",
					err,
				)
				return
			}

			// Check bootstrap token (in-memory, only present before vault is initialized)
			if s.APITokenHash != nil {
				tokenMatches, _ := cryptoutil.VerifyToken(s.Vault.GetMasterKey(), rawToken, s.APITokenHash)
				if tokenMatches {
					if s.TokenExpiry == 0 || time.Now().Unix() <= s.TokenExpiry {
						next(w, r)
						return
					}
				}
			}

			// Check DB tokens (only when vault is unlocked)
			if s.Vault.IsUnlocked() {
				valid, err := s.Vault.VerifyAPIToken(rawToken)
				if err != nil {
					writeError(
						w,
						http.StatusUnauthorized,
						"unauthorized",
						"could not verify token",
						err,
					)
					return
				}
				if valid {
					next(w, r)
					return
				}
			}
		}

		// Basic Auth Verification
		username, password, ok := r.BasicAuth()
		if ok {
			isUserValid, err := s.Vault.VerifyUser(username, password)
			if err != nil {
				writeError(
					w,
					http.StatusUnauthorized,
					"unauthorized",
					"could not verify user",
					err,
				)
				return
			}

			if isUserValid {
				next(w, r)
				return
			}
		}

		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized user or token",
			nil,
		)
	}
}

// ~~~ InitialToken ~~~
// generates expired initial token to configure vault
func (s *Server) InitialToken() error {
	if s.Vault.IsInitialized() == false {
		// Generate a 32-byte raw token
		newTokenHash, rawToken, err := cryptoutil.GenerateToken(s.Vault.GetMasterKey(), 32)
		if err != nil {
			return err
		}

		s.APITokenHash = newTokenHash
		s.TokenExpiry = 0

		// Encode for URL-safe transmission
		urlSafeToken := base64.RawURLEncoding.EncodeToString(rawToken)
		// Print it to the console for the user to use
		l.LogAddInfo(
			l.Logger.Info,
			"generated initial vault token",
			"token", urlSafeToken,
		)
		l.Logger.Warn("this will only be generated if no vault exists")
		// TODO: add log pointing to documentation on
	}

	return nil
}

// =====================================================
// END "Authentication"
// =====================================================

// =====================================================
// HTTP Request Functions
// =====================================================
// functions that are called on http requests

func (s *Server) InitializeVault(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
			"attempted to call non-POST method on /initvault",
			nil,
		)
		return
	}

	// Check if already initialized
	if s.Vault.IsInitialized() {
		writeError(
			w,
			http.StatusForbidden,
			"unable to initialize vault",
			"vault already initialized",
			nil,
		)
		return
	}

	type InitRequest struct {
		Username         string `json:"username"`
		Password         string `json:"password"`
		MasterPassphrase string `json:"master_passphrase"`
	}

	// Decode request body
	var req InitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid JSON",
			"invalid JSON provided",
			err,
		)
		return
	}
	defer r.Body.Close()

	// Validate inputs
	if len(req.Username) == 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"missing required field",
			"username required",
			nil,
		)
		return
	}
	if len(req.Password) < 8 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid field values",
			"password too short, must be 8 or more characters",
			nil,
		)
		return
	}
	if len(req.MasterPassphrase) < 12 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid field values",
			"master passphrase too short, must be 12 or more characters",
			nil,
		)
		return
	}

	// Initialize vault (DB, user, etc)
	if err := s.Vault.InitializeVault(req.Username, req.Password, req.MasterPassphrase); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"couldn't initialize vault",
			"failed to initialize vault",
			err,
		)
		return
	}

	// Invalidate the bootstrap token now that the vault is initialized
	s.APITokenHash = nil
	s.TokenExpiry = 0

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message":"vault initialized successfully"}`))
}

func (s *Server) OpenVault(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
			"attempted to call non-POST method on /openvault",
			nil,
		)
		return
	}

	// Check if initialized
	if s.Vault.IsInitialized() == false {
		writeError(
			w,
			http.StatusForbidden,
			"couldn't open vault",
			"vault not initialized",
			nil,
		)
		return
	}

	type OpenRequest struct {
		MasterPassphrase string `json:"master_passphrase"`
	}

	// Decode request body
	var req OpenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid JSON",
			"invalid JSON provided",
			err,
		)
		return
	}
	defer r.Body.Close()

	if len(req.MasterPassphrase) < 12 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid field values",
			"master passphrase too short, must be 12 or more characters",
			nil,
		)
		return
	}

	err := s.Vault.OpenVault(req.MasterPassphrase)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"could not open vault",
			"could not open vault",
			err,
		)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"vault opened successfully"}`))
}

func (s *Server) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Vault.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list users", "failed to list users", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (s *Server) AddUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "attempted to call non-POST method on /adduser", nil)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON", "invalid JSON provided", err)
		return
	}
	defer r.Body.Close()

	if len(req.Username) == 0 {
		writeError(w, http.StatusBadRequest, "missing required field", "username required", nil)
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "invalid field values", "password too short, must be 8 or more characters", nil)
		return
	}

	if err := s.Vault.AddUser(req.Username, req.Password); err != nil {
		writeError(w, http.StatusInternalServerError, "could not add user", "failed to add user", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message":"user added successfully"}`))
}

func (s *Server) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "attempted to call non-DELETE method on /deleteuser", nil)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter", "username parameter is required", nil)
		return
	}

	if err := s.Vault.DeleteUser(username); err != nil {
		writeError(w, http.StatusBadRequest, "could not delete user", "failed to delete user", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) GenerateToken(w http.ResponseWriter, r *http.Request) {
	// http://server:port/token?name=personal&ttl=3600
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter", "name parameter is required", nil)
		return
	}

	// Read optional TTL query parameter (seconds)
	ttl := int64(0) // default: no expiration
	if val := r.URL.Query().Get("ttl"); val != "" {
		parsedTTL, err := strconv.ParseInt(val, 10, 64)
		if err != nil || parsedTTL < 0 {
			writeError(w, http.StatusBadRequest, "invalid ttl", "ttl must be a non-negative integer", nil)
			return
		}
		ttl = parsedTTL
	}

	newTokenHash, newRawToken, err := cryptoutil.GenerateToken(s.Vault.GetMasterKey(), 32)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"could not generate token",
			"failed to generate token",
			err,
		)
		return
	}

	var expiresAt *int64
	if ttl > 0 {
		t := time.Now().Unix() + ttl
		expiresAt = &t
	}

	if err := s.Vault.StoreToken(name, newTokenHash, expiresAt); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"could not store token",
			"failed to store token",
			err,
		)
		return
	}

	urlSafeToken := base64.RawURLEncoding.EncodeToString(newRawToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":       name,
		"token":      urlSafeToken,
		"expires_in": ttl,
	})
}

func (s *Server) DeleteToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "attempted to call non-DELETE method on /deletetoken", nil)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter", "name parameter is required", nil)
		return
	}

	if err := s.Vault.DeleteToken(name); err != nil {
		writeError(
			w,
			http.StatusNotFound,
			"could not delete token",
			"failed to delete token",
			err,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ListTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := s.Vault.ListTokens()
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"could not list tokens",
			"failed to list tokens",
			err,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

func (s *Server) StoreSecret(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Secret string `json:"secret"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid JSON",
			"failed to read json body",
			err,
		)
		return
	}
	defer r.Body.Close()
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid JSON",
			"invalid JSON provided",
			err,
		)
		return
	}

	if err := s.Vault.StoreSecret(req.Name, []byte(req.Secret)); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"could not store secret",
			"failed to store secret",
			err,
		)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) GetSecret(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	secret, err := s.Vault.GetSecret(name)
	if err != nil {
		writeError(
			w,
			http.StatusNotFound,
			"could not retreive secret",
			"failed to retreive secret",
			err,
		)
		return
	}

	resp := map[string]string{
		"name":   name,
		"secret": string(secret),
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) DeleteSecret(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if err := s.Vault.DeleteSecret(name); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"could not delete secret",
			"failed to delete secret",
			err,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ListSecrets(w http.ResponseWriter, r *http.Request) {
	secrets, err := s.Vault.ListSecrets()
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"could not list secrets",
			"failed to list secrets",
			err,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(secrets)
}

// =====================================================
// END "Secret CRUD Operations"
// =====================================================

// =====================================================
// Error Helpers
// =====================================================
// functions to help properly handle errors

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	ErrorID string `json:"error_id,omitempty"`
}

func writeError(w http.ResponseWriter, status int, httpMsg string, logMsg string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var errorID string
	errorID = uuid.NewString()

	l.LogError(
		l.Logger.Error,

		logMsg,
		"error_id", errorID,
		"status", status,
		"err", err,
	)

	json.NewEncoder(w).Encode(ErrorResponse{
		Status:  fmt.Sprintf("%d", status),
		Message: httpMsg,
		ErrorID: errorID,
	})
}

// =====================================================
// END "Error Helpers"
// =====================================================
