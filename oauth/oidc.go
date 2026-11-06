package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func init() {
	Register("oidc", &OIDCProvider{})
}

// OIDCProvider implements OAuth for OIDC
type OIDCProvider struct{}

type oidcOAuthResponse struct {
	Code   string `json:"code"`
	Msg    string `json:"msg"`
	Result struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    string `json:"expiresIn"`
		Jti          string `json:"jti"`
	} `json:"result"`
	// Flat fields for standard OIDC providers that return tokens directly
	AccessTokenFlat  string `json:"accessToken"`
	IDToken          string `json:"id_token"`
	RefreshTokenFlat string `json:"refreshToken"`
	TokenType        string `json:"token_type"`
	ExpiresInInt     int    `json:"expiresIn"`
	Scope            string `json:"scope"`
}

// effectiveAccessToken returns the access token from either the nested
// "result" wrapper or the flat top-level field, depending on provider.
func (r *oidcOAuthResponse) effectiveAccessToken() string {
	if r.Result.AccessToken != "" {
		return r.Result.AccessToken
	}
	return r.AccessTokenFlat
}

func (r *oidcOAuthResponse) effectiveRefreshToken() string {
	if r.Result.RefreshToken != "" {
		return r.Result.RefreshToken
	}
	return r.RefreshTokenFlat
}

type oidcUser struct {
	Code   string `json:"code"`
	Msg    string `json:"msg"`
	Result struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Account string `json:"account"`
		Email   string `json:"email"`
		Avatar  string `json:"avatar"`
	} `json:"result"`
	// Flat fields for standard OIDC providers that return user info directly
	OpenID            string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Picture           string `json:"picture"`
}

func (u *oidcUser) effectiveOpenID() string {
	if u.Result.ID != "" {
		return u.Result.ID
	}
	return u.OpenID
}

func (u *oidcUser) effectiveName() string {
	if u.Result.Name != "" {
		return u.Result.Name
	}
	return u.Name
}

func (u *oidcUser) effectiveUsername() string {
	if u.Result.Email != "" {
		return strings.SplitN(u.Result.Email, "@", 2)[0]
	}
	if u.Result.Account != "" {
		return u.Result.Account
	}
	if u.PreferredUsername != "" {
		return u.PreferredUsername
	}
	return u.Name
}

func (u *oidcUser) effectiveEmail() string {
	if u.Result.Email != "" {
		return u.Result.Email
	}
	return u.Email
}

func (u *oidcUser) effectivePicture() string {
	if u.Result.Avatar != "" {
		return u.Result.Avatar
	}
	return u.Picture
}

func (p *OIDCProvider) GetName() string {
	return system_setting.GetOIDCSettings().GetEffectiveDisplayName()
}

func (p *OIDCProvider) IsEnabled() bool {
	return system_setting.GetOIDCSettings().Enabled
}

func (p *OIDCProvider) ExchangeToken(ctx context.Context, code string, c *gin.Context) (*OAuthToken, error) {
	if code == "" {
		return nil, NewOAuthError(i18n.MsgOAuthInvalidCode, nil)
	}

	logger.LogDebug(ctx, "[OAuth-OIDC] ExchangeToken: code=%s...", code[:min(len(code), 10)])

	settings := system_setting.GetOIDCSettings()
	redirectUri := fmt.Sprintf("%s/oauth/oidc", system_setting.ServerAddress)
	values := url.Values{}
	values.Set("client_id", settings.ClientId)
	values.Set("client_secret", settings.ClientSecret)
	values.Set("code", code)
	values.Set("grant_type", "authorization_code")
	values.Set("redirect_uri", redirectUri)

	logger.LogDebug(ctx, "[OAuth-OIDC] ExchangeToken: token_endpoint=%s, redirect_uri=%s", settings.TokenEndpoint, redirectUri)

	req, err := http.NewRequestWithContext(ctx, "POST", settings.TokenEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-OIDC] ExchangeToken error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "OIDC"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-OIDC] ExchangeToken response status: %d", res.StatusCode)

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-OIDC] ExchangeToken read body error: %s", err.Error()))
		return nil, err
	}
	logger.LogDebug(ctx, "[OAuth-OIDC] ExchangeToken response body: %s", string(bodyBytes))

	var oidcResponse oidcOAuthResponse
	err = json.Unmarshal(bodyBytes, &oidcResponse)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-OIDC] ExchangeToken decode error: %s", err.Error()))
		return nil, err
	}

	accessToken := oidcResponse.effectiveAccessToken()
	if accessToken == "" {
		logger.LogError(ctx, "[OAuth-OIDC] ExchangeToken failed: empty access token")
		return nil, NewOAuthError(i18n.MsgOAuthTokenFailed, map[string]any{"Provider": "OIDC"})
	}

	var expiresIn int
	if oidcResponse.Result.ExpiresIn != "" {
		expiresIn, _ = strconv.Atoi(oidcResponse.Result.ExpiresIn)
	} else {
		expiresIn = oidcResponse.ExpiresInInt
	}

	logger.LogDebug(ctx, "[OAuth-OIDC] ExchangeToken success: scope=%s", oidcResponse.Scope)

	return &OAuthToken{
		AccessToken:  accessToken,
		TokenType:    oidcResponse.TokenType,
		RefreshToken: oidcResponse.effectiveRefreshToken(),
		ExpiresIn:    expiresIn,
		Scope:        oidcResponse.Scope,
		IDToken:      oidcResponse.IDToken,
	}, nil
}

func (p *OIDCProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*OAuthUser, error) {
	settings := system_setting.GetOIDCSettings()

	logger.LogDebug(ctx, "[OAuth-OIDC] GetUserInfo: userinfo_endpoint=%s", settings.UserInfoEndpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", settings.UserInfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-OIDC] GetUserInfo error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "OIDC"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-OIDC] GetUserInfo response status: %d", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-OIDC] GetUserInfo failed: status=%d", res.StatusCode))
		return nil, NewOAuthError(i18n.MsgOAuthGetUserErr, nil)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-OIDC] GetUserInfo read body error: %s", err.Error()))
		return nil, err
	}
	logger.LogDebug(ctx, "[OAuth-OIDC] GetUserInfo response body: %s", string(bodyBytes))

	var oidcUser oidcUser
	err = json.Unmarshal(bodyBytes, &oidcUser)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-OIDC] GetUserInfo decode error: %s", err.Error()))
		return nil, err
	}

	openID := oidcUser.effectiveOpenID()
	if openID == "" {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-OIDC] GetUserInfo failed: empty user id"))
		return nil, NewOAuthError(i18n.MsgOAuthUserInfoEmpty, map[string]any{"Provider": "OIDC"})
	}

	email := oidcUser.effectiveEmail()
	username := oidcUser.effectiveUsername()
	if username == "" {
		username = oidcUser.effectiveName()
	}

	logger.LogDebug(ctx, "[OAuth-OIDC] GetUserInfo success: id=%s, username=%s, name=%s, email=%s", openID, username, oidcUser.effectiveName(), email)

	return &OAuthUser{
		ProviderUserID: openID,
		Username:       username,
		DisplayName:    oidcUser.effectiveName(),
		Email:          email,
	}, nil
}

func (p *OIDCProvider) IsUserIDTaken(providerUserID string) bool {
	return model.IsOidcIdAlreadyTaken(providerUserID)
}

func (p *OIDCProvider) FillUserByProviderID(user *model.User, providerUserID string) error {
	user.OidcId = providerUserID
	return user.FillUserByOidcId()
}

func (p *OIDCProvider) SetProviderUserID(user *model.User, providerUserID string) {
	user.OidcId = providerUserID
}

func (p *OIDCProvider) GetProviderPrefix() string {
	return "oidc_"
}

// ProviderUserIDColumn returns the users-table column storing this provider's user ID.
func (p *OIDCProvider) ProviderUserIDColumn() string {
	return "oidc_id"
}
