package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// weakSecrets is a list of common weak JWT secrets to test against.
var weakSecrets = []string{
	"secret",
	"password",
	"123456",
	"key",
	"secret123",
	"jwt",
	"token",
	"mysecret",
	"change_me",
	"your-256-bit-secret",
	"your-secret-key",
	"supersecret",
	"admin",
	"letmein",
	"qwerty",
	"passw0rd",
	"jwtkey",
	"mykey",
	"test",
	"dev",
}

// AnalyzeJWT decodes and analyzes a JWT token for vulnerabilities.
func AnalyzeJWT(tokenStr string) JWTAnalysis {
	tokenStr = strings.TrimSpace(tokenStr)

	analysis := JWTAnalysis{
		Raw:   tokenStr,
		Valid: false,
	}

	// Split into parts
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities, "Invalid JWT format: expected 3 parts separated by '.'")
		return analysis
	}

	headerJSON, err := base64URLDecode(parts[0])
	if err != nil {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities, fmt.Sprintf("Failed to decode header: %v", err))
		return analysis
	}

	payloadJSON, err := base64URLDecode(parts[1])
	if err != nil {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities, fmt.Sprintf("Failed to decode payload: %v", err))
		return analysis
	}

	// Parse header
	var header map[string]interface{}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities, fmt.Sprintf("Failed to parse header JSON: %v", err))
		return analysis
	}
	analysis.Header = header

	// Parse payload
	var payload map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities, fmt.Sprintf("Failed to parse payload JSON: %v", err))
		return analysis
	}
	analysis.Payload = payload
	analysis.Valid = true

	// Check algorithm
	if alg, ok := header["alg"].(string); ok {
		analysis.Algorithm = alg
		checkAlgorithm(&analysis, alg)
	} else {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities, "Missing 'alg' field in header")
	}

	// Check expiration
	checkExpiration(&analysis, payload)

	// Check for missing claims
	checkClaims(&analysis, payload)

	// Try weak secrets if HMAC-based
	if strings.HasPrefix(analysis.Algorithm, "HS") {
		checkWeakSecrets(&analysis, parts[0]+"."+parts[1], parts[2], analysis.Algorithm)
	}

	return analysis
}

func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

func checkAlgorithm(analysis *JWTAnalysis, alg string) {
	algLower := strings.ToLower(alg)

	if algLower == "none" {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities,
			"CRITICAL: Algorithm 'none' accepted - authentication bypass possible")
	}

	if algLower == "hs256" || algLower == "hs384" || algLower == "hs512" {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities,
			"INFO: HMAC algorithm - susceptible to weak secret attacks")
	}

	// RS256/ES256 tokens might be vulnerable to algorithm confusion
	if algLower == "rs256" || algLower == "rs384" || algLower == "rs512" {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities,
			"INFO: Algorithm confusion possible (RS256 → HS256) if server uses public key as HMAC secret")
	}
}

func checkExpiration(analysis *JWTAnalysis, payload map[string]interface{}) {
	if exp, ok := payload["exp"]; ok {
		var expTime int64
		switch v := exp.(type) {
		case float64:
			expTime = int64(v)
		case int64:
			expTime = v
		}
		if expTime > 0 {
			t := time.Unix(expTime, 0)
			analysis.ExpiresAt = &t
			if time.Now().After(t) {
				analysis.IsExpired = true
				analysis.Vulnerabilities = append(analysis.Vulnerabilities,
					fmt.Sprintf("Token is expired (expired at %s)", t.Format(time.RFC3339)))
			}
		}
	} else {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities,
			"Missing 'exp' claim - token never expires")
	}

	if iat, ok := payload["iat"]; ok {
		var iatTime int64
		switch v := iat.(type) {
		case float64:
			iatTime = int64(v)
		case int64:
			iatTime = v
		}
		if iatTime > 0 {
			t := time.Unix(iatTime, 0)
			analysis.IssuedAt = &t
		}
	}
}

func checkClaims(analysis *JWTAnalysis, payload map[string]interface{}) {
	if _, ok := payload["nbf"]; !ok {
		// not before missing - not critical
	}
	if _, ok := payload["iss"]; !ok {
		analysis.Vulnerabilities = append(analysis.Vulnerabilities,
			"Missing 'iss' (issuer) claim - token origin cannot be verified")
	}
	if _, ok := payload["aud"]; !ok {
		// audience missing - informational
	}
}

func checkWeakSecrets(analysis *JWTAnalysis, signingInput, signature, alg string) {
	var hashFunc func() interface{}
	_ = hashFunc

	for _, secret := range weakSecrets {
		if verifyHMAC(signingInput, signature, secret, alg) {
			analysis.Vulnerabilities = append(analysis.Vulnerabilities,
				fmt.Sprintf("CRITICAL: Weak secret detected: %q - token can be forged", secret))
			return
		}
	}
}

func verifyHMAC(signingInput, signature, secret, alg string) bool {
	var h func() ([]byte, error)

	switch strings.ToUpper(alg) {
	case "HS256":
		h = func() ([]byte, error) {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(signingInput))
			return mac.Sum(nil), nil
		}
	default:
		return false
	}

	expected, err := h()
	if err != nil {
		return false
	}

	// Encode expected to base64url (no padding)
	expectedB64 := base64.URLEncoding.EncodeToString(expected)
	expectedB64 = strings.TrimRight(expectedB64, "=")

	// Also try raw encoding
	expectedRawB64 := base64.RawURLEncoding.EncodeToString(expected)

	return signature == expectedB64 || signature == expectedRawB64
}
