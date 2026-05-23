package domain

// TokenPair represents the access, refresh, and ID token payload container.
type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    IDToken      string `json:"id_token,omitempty"`
    ExpiresIn    int    `json:"expires_in"`
}
