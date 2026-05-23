package gateway

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"
    "net/url"
    "strings"

    "github.com/democryst/go-api-management/internal/core/domain"
    "github.com/democryst/go-api-management/internal/core/ports"
    "github.com/sony/gobreaker"
)

// Ensure Auth0Gateway satisfies the outbound port interface at compile time.
var _ ports.AuthProvider = (*Auth0Gateway)(nil)

// Auth0Gateway provides the resilient OIDC adapter client to communicate with Auth0.
type Auth0Gateway struct {
    domain       string
    clientID     string
    clientSecret string
    apiAudience  string
    redirectURI  string
    client       *http.Client
    breaker      *gobreaker.CircuitBreaker
    logger       *slog.Logger
}

// NewAuth0Gateway constructs a new Auth0 OIDC gateway adapter.
func NewAuth0Gateway(
    domain string,
    clientID string,
    clientSecret string,
    apiAudience string,
    redirectURI string,
    client *http.Client,
    breaker *gobreaker.CircuitBreaker,
    logger *slog.Logger,
) *Auth0Gateway {
    return &Auth0Gateway{
        domain:       domain,
        clientID:     clientID,
        clientSecret: clientSecret,
        apiAudience:  apiAudience,
        redirectURI:  redirectURI,
        client:       client,
        breaker:      breaker,
        logger:       logger,
    }
}

type auth0TokenRequest struct {
    GrantType    string `json:"grant_type"`
    ClientID     string `json:"client_id"`
    ClientSecret string `json:"client_secret,omitempty"`
    Code         string `json:"code,omitempty"`
    CodeVerifier string `json:"code_verifier,omitempty"`
    RedirectURI  string `json:"redirect_uri,omitempty"`
    RefreshToken string `json:"refresh_token,omitempty"`
}

type auth0TokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    IDToken      string `json:"id_token"`
    ExpiresIn    int    `json:"expires_in"`
}

type auth0UserInfo struct {
    Sub           string `json:"sub"`
    Email         string `json:"email"`
    Name          string `json:"name"`
    Picture       string `json:"picture"`
    EmailVerified bool   `json:"email_verified"`
}

// GetAuthorizationURL builds the Auth0 authorize redirect URL with PKCE.
// Time Complexity: O(1)
// Space Complexity: O(1)
func (g *Auth0Gateway) GetAuthorizationURL(ctx context.Context, session domain.AuthSession) (string, error) {
    u := &url.URL{
        Scheme: "https",
        Host:   g.domain,
        Path:   "/authorize",
    }
    q := u.Query()
    q.Set("response_type", "code")
    q.Set("client_id", g.clientID)
    q.Set("redirect_uri", g.redirectURI)
    q.Set("state", session.State)
    q.Set("code_challenge", session.CodeVerifier) // S256 challenge string derived inside service
    q.Set("code_challenge_method", "S256")
    if g.apiAudience != "" {
        q.Set("audience", g.apiAudience)
    }
    q.Set("scope", "openid profile email offline_access")
    u.RawQuery = q.Encode()

    g.logger.InfoContext(ctx, "generated Auth0 authorization URL", slog.String("state", session.State))
    return u.String(), nil
}

// ExchangeCode exchanges the authorization code + verifier for tokens.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (g *Auth0Gateway) ExchangeCode(ctx context.Context, code, codeVerifier string) (domain.TokenPair, error) {
    reqBody := auth0TokenRequest{
        GrantType:    "authorization_code",
        ClientID:     g.clientID,
        ClientSecret: g.clientSecret,
        Code:         code,
        CodeVerifier: codeVerifier,
        RedirectURI:  g.redirectURI,
    }

    respInterface, err := g.breaker.Execute(func() (interface{}, error) {
        return g.sendTokenRequest(ctx, reqBody)
    })
    if err != nil {
        return domain.TokenPair{}, fmt.Errorf("token exchange circuit error: %w", err)
    }

    resp := respInterface.(*auth0TokenResponse)
    return domain.TokenPair{
        AccessToken:  resp.AccessToken,
        RefreshToken: resp.RefreshToken,
        IDToken:      resp.IDToken,
        ExpiresIn:    resp.ExpiresIn,
    }, nil
}

// RefreshToken executes refresh token rotation.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (g *Auth0Gateway) RefreshToken(ctx context.Context, refreshToken string) (domain.TokenPair, error) {
    reqBody := auth0TokenRequest{
        GrantType:    "refresh_token",
        ClientID:     g.clientID,
        ClientSecret: g.clientSecret,
        RefreshToken: refreshToken,
    }

    respInterface, err := g.breaker.Execute(func() (interface{}, error) {
        return g.sendTokenRequest(ctx, reqBody)
    })
    if err != nil {
        return domain.TokenPair{}, fmt.Errorf("token refresh circuit error: %w", err)
    }

    resp := respInterface.(*auth0TokenResponse)
    return domain.TokenPair{
        AccessToken:  resp.AccessToken,
        RefreshToken: resp.RefreshToken,
        IDToken:      resp.IDToken,
        ExpiresIn:    resp.ExpiresIn,
    }, nil
}

// RevokeToken revokes the given refresh token on the Auth0 endpoint.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (g *Auth0Gateway) RevokeToken(ctx context.Context, token string) error {
    _, err := g.breaker.Execute(func() (interface{}, error) {
        u := fmt.Sprintf("https://%s/oauth/revoke", g.domain)
        data := url.Values{}
        data.Set("client_id", g.clientID)
        if g.clientSecret != "" {
            data.Set("client_secret", g.clientSecret)
        }
        data.Set("token", token)

        req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(data.Encode()))
        if err != nil {
            return nil, err
        }
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

        resp, err := g.client.Do(req)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return nil, fmt.Errorf("failed token revocation, status: %d", resp.StatusCode)
        }
        return nil, nil
    })

    if err != nil {
        return fmt.Errorf("token revocation failed: %w", err)
    }
    return nil
}

// GetUserInfo fetches OIDC profile attributes associated with the access token.
// Time Complexity: O(1) (network bound)
// Space Complexity: O(1)
func (g *Auth0Gateway) GetUserInfo(ctx context.Context, accessToken string) (domain.UserIdentity, error) {
    respInterface, err := g.breaker.Execute(func() (interface{}, error) {
        u := fmt.Sprintf("https://%s/userinfo", g.domain)
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
        if err != nil {
            return nil, err
        }
        req.Header.Set("Authorization", "Bearer "+accessToken)

        resp, err := g.client.Do(req)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return nil, fmt.Errorf("userinfo request failed, status: %d", resp.StatusCode)
        }

        var userInfo auth0UserInfo
        if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
            return nil, err
        }
        return &userInfo, nil
    })

    if err != nil {
        return domain.UserIdentity{}, fmt.Errorf("userinfo fetching error: %w", err)
    }

    info := respInterface.(*auth0UserInfo)
    return domain.UserIdentity{
        ID:            info.Sub,
        Email:         domain.MaskedEmail(info.Email),
        Name:          domain.MaskedName(info.Name),
        Picture:       info.Picture,
        EmailVerified: info.EmailVerified,
    }, nil
}

func (g *Auth0Gateway) sendTokenRequest(ctx context.Context, reqBody auth0TokenRequest) (*auth0TokenResponse, error) {
    u := fmt.Sprintf("https://%s/oauth/token", g.domain)
    bodyBytes, err := json.Marshal(reqBody)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(bodyBytes))
    if err != nil {
        return nil, fmt.Errorf("create token request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := g.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("do token request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        var errBody map[string]interface{}
        _ = json.NewDecoder(resp.Body).Decode(&errBody)
        return nil, fmt.Errorf("token request returned status %d: %v", resp.StatusCode, errBody)
    }

    var tokenResp auth0TokenResponse
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return nil, fmt.Errorf("decode token response: %w", err)
    }

    return &tokenResp, nil
}
