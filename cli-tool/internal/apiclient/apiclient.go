package apiclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	Username   string
	Password   string
	AuthMethod string
	Client     *http.Client
}

// ~~~ NewClient ~~~
// creates a new http client to communicate with BS3 server
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		Client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// setAuth attaches credentials to a request.
// When AuthMethod is "basic", Basic Auth is used unconditionally.
// Otherwise (default), Bearer token is used if set.
func (c *Client) setAuth(req *http.Request) {
	if c.AuthMethod == "basic" {
		req.SetBasicAuth(c.Username, c.Password)
	} else {
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
	}
}

// ~~~ GetSecret ~~~
// sends GET http request to BS3 server to retreive secret by name
// returning a map with the name and secret value
func (c *Client) GetSecret(name string) (map[string]string, error) {
	endpoint := fmt.Sprintf("%s/get?name=%s", c.BaseURL, url.QueryEscape(name))
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error fetching secret: %s", body)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return result, nil
}

// ~~~ ListSecret ~~~
// sends a GET http request to BS3 server to retreive secret names,
// created times, and updated times returning a list of strings
func (c *Client) ListSecrets() ([]string, error) {
	endpoint := fmt.Sprintf("%s/listsecrets", c.BaseURL)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing secrets: %s", body)
	}

	var secrets []string
	if err := json.NewDecoder(resp.Body).Decode(&secrets); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return secrets, nil
}

// SecretMeta holds metadata about a secret returned from the BS3 server.
type SecretMeta struct {
	Name      string `json:"Name"`
	CreatedAt string `json:"CreatedAt"`
	UpdatedAt string `json:"UpdatedAt"`
}

// ~~~ ListSecretsMeta ~~~
// sends a GET http request to BS3 server to retrieve secret metadata
// (name, created_at, updated_at) for all stored secrets
func (c *Client) ListSecretsMeta() ([]SecretMeta, error) {
	endpoint := fmt.Sprintf("%s/listsecrets", c.BaseURL)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing secrets: %s", body)
	}

	var secrets []SecretMeta
	if err := json.NewDecoder(resp.Body).Decode(&secrets); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return secrets, nil
}

// ~~~ UserMeta ~~~
// holds metadata about a user returned from the BS3 server.
type UserMeta struct {
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

// ~~~ ListUsers ~~~
// sends a GET http request to BS3 server to retrieve all user metadata
func (c *Client) ListUsers() ([]UserMeta, error) {
	endpoint := fmt.Sprintf("%s/listusers", c.BaseURL)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing users: %s", body)
	}

	var users []UserMeta
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return users, nil
}

// ~~~ AddUser ~~~
// sends a POST http request to BS3 server to add a new user
func (c *Client) AddUser(username, password string) error {
	payload := map[string]string{"username": username, "password": password}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/adduser", c.BaseURL)
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error adding user: %s", body)
	}

	return nil
}

// ~~~ DeleteUser ~~~
// sends a DELETE http request to BS3 server to delete a user by username
func (c *Client) DeleteUser(username string) error {
	endpoint := fmt.Sprintf("%s/deleteuser?username=%s", c.BaseURL, url.QueryEscape(username))
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	c.setAuth(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting user: %s", body)
	}

	return nil
}

// ~~~ TokenMeta ~~~
// holds metadata about an API token returned from the BS3 server.
type TokenMeta struct {
	Name      string `json:"name"`
	ExpiresAt *int64 `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

// ~~~ GeneratedToken ~~~
// holds the response from a successful token generation request.
type GeneratedToken struct {
	Name      string `json:"name"`
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
}

// ~~~ ListTokens ~~~
// sends a GET http request to BS3 server to retrieve all API token metadata
func (c *Client) ListTokens() ([]TokenMeta, error) {
	endpoint := fmt.Sprintf("%s/listtokens", c.BaseURL)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing tokens: %s", body)
	}

	var tokens []TokenMeta
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return tokens, nil
}

// ~~~ GenerateToken ~~~
// sends a GET http request to BS3 server to generate a new API token
// ttl is expiry in seconds; 0 means no expiry
func (c *Client) GenerateToken(name string, ttl int64) (*GeneratedToken, error) {
	endpoint := fmt.Sprintf("%s/token?name=%s&ttl=%d", c.BaseURL, url.QueryEscape(name), ttl)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error generating token: %s", body)
	}

	var result GeneratedToken
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result, nil
}

// ~~~ DeleteToken ~~~
// sends a DELETE http request to BS3 server to delete an API token by name
func (c *Client) DeleteToken(name string) error {
	endpoint := fmt.Sprintf("%s/deletetoken?name=%s", c.BaseURL, url.QueryEscape(name))
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	c.setAuth(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting token: %s", body)
	}

	return nil
}

// ~~~ DeleteSecret ~~~
// sends a DELETE http request to BS3 server to delete a secret by name
func (c *Client) DeleteSecret(name string) error {
	endpoint := fmt.Sprintf("%s/delete?name=%s", c.BaseURL, url.QueryEscape(name))
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	c.setAuth(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting secret: %s", body)
	}

	return nil
}

// ~~~ InitializeVault ~~~
// sends a POST http request to BS3 server to initialize the vault with an initial
// admin user and master passphrase; requires the bootstrap bearer token
func (c *Client) InitializeVault(username, password, masterPassphrase string) error {
	payload := map[string]string{
		"username":          username,
		"password":          password,
		"master_passphrase": masterPassphrase,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/initvault", c.BaseURL)
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error initializing vault: %s", body)
	}

	return nil
}

// ~~~ OpenVault ~~~
// sends a POST http request to BS3 server to open the vault using basic auth
// returns the response body as a raw byte slice
func (c *Client) OpenVault(username, password, masterPassphrase string) ([]byte, error) {
	payload := map[string]string{"master_passphrase": masterPassphrase}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/openvault", c.BaseURL)
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error opening vault: %s", body)
	}

	return body, nil
}

// ~~~ StoreSecret ~~~
// sends a POST http request to BS3 server to send a secret name and value
// to be encrypted and stored in the vault
func (c *Client) StoreSecret(name, value string) error {
	payload := map[string]string{
		"name":   name,
		"secret": value,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/store", c.BaseURL)
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error storing secret: %s", body)
	}

	return nil
}
