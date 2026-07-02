package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/crevas/Apple-Ads-CLI/internal/config"
)

const (
	tokenScope         = "searchadsorg"
	jwtLifetime        = 180 * 24 * time.Hour
	tokenRefreshBuffer = 60 * time.Second
)

type TokenSource struct {
	cfg        config.Config
	httpClient *http.Client
	privateKey *ecdsa.PrivateKey

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func NewTokenSource(cfg config.Config) (*TokenSource, error) {
	if err := cfg.ValidateAuth(); err != nil {
		return nil, err
	}

	privateKey, err := readPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	return &TokenSource{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout()},
		privateKey: privateKey,
	}, nil
}

func (s *TokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.accessToken != "" && time.Now().Before(s.expiresAt.Add(-tokenRefreshBuffer)) {
		return s.accessToken, nil
	}

	return s.refresh(ctx)
}

func (s *TokenSource) refresh(ctx context.Context) (string, error) {
	clientSecret, err := s.clientSecret()
	if err != nil {
		return "", err
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {s.cfg.ClientID},
		"client_secret": {clientSecret},
		"scope":         {tokenScope},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("token response did not include access_token")
	}

	expiresIn := parsed.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	s.accessToken = parsed.AccessToken
	s.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	return s.accessToken, nil
}

func (s *TokenSource) clientSecret() (string, error) {
	now := time.Now()
	header := map[string]string{
		"alg": "ES256",
		"kid": s.cfg.KeyID,
		"typ": "JWT",
	}
	claims := map[string]any{
		"iss": s.cfg.TeamID,
		"sub": s.cfg.ClientID,
		"aud": "https://appleid.apple.com",
		"iat": now.Unix(),
		"exp": now.Add(jwtLifetime).Unix(),
	}

	encodedHeader, err := encodeJSONSegment(header)
	if err != nil {
		return "", err
	}
	encodedClaims, err := encodeJSONSegment(claims)
	if err != nil {
		return "", err
	}
	signingInput := encodedHeader + "." + encodedClaims
	digest := sha256.Sum256([]byte(signingInput))
	r, ss, err := ecdsa.Sign(rand.Reader, s.privateKey, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	signature := joseSignature(r, ss, s.privateKey.Params().BitSize)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func encodeJSONSegment(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal jwt segment: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func joseSignature(r *big.Int, s *big.Int, bitSize int) []byte {
	keyBytes := (bitSize + 7) / 8
	signature := make([]byte, keyBytes*2)
	r.FillBytes(signature[:keyBytes])
	s.FillBytes(signature[keyBytes:])
	return signature
}

func readPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("private key file has no PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		if ecKey, ecErr := x509.ParseECPrivateKey(block.Bytes); ecErr == nil {
			return ecKey, nil
		}
		if sec1Key, sec1Err := parseSEC1PrivateKey(block.Bytes); sec1Err == nil {
			return sec1Key, nil
		}
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not ECDSA")
	}
	return ecKey, nil
}

func parseSEC1PrivateKey(der []byte) (*ecdsa.PrivateKey, error) {
	var raw struct {
		Version       int
		PrivateKey    []byte
		NamedCurveOID asn1.ObjectIdentifier `asn1:"optional,explicit,tag:0"`
		PublicKey     asn1.BitString        `asn1:"optional,explicit,tag:1"`
	}
	if _, err := asn1.Unmarshal(der, &raw); err != nil {
		return nil, err
	}
	return x509.ParseECPrivateKey(der)
}
