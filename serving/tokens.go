package serving

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TokenDuration = time.Hour * 24

// TokenPayload to parse from the payload part of the token
type TokenPayload struct {
	User       string `json:"user"`
	Expiration int64  `json:"expiration"`
}

// createToken builds a new token for a given login using its secret
func createToken(userName string, userSecret string) (string, error) {
	// Source for JWT implementation: https://medium.com/@cheickzida/golang-implementing-jwt-token-authentication-bba9bfd84d60
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"user":       userName,
			"expiration": time.Now().Add(TokenDuration).Unix(),
		})

	if token, err := token.SignedString([]byte(userSecret)); err != nil {
		return "", err
	} else {
		return token, nil
	}
}

// validateAuthentication reads header and then test if login matches its expected secret.
// Result is login coming from request, true for auth success, the detailed error otherwise
func validateAuthentication(wrapper ServiceParameters, r *http.Request) (string, bool, error) {
	// Explanation are there: https://jwt.io/introduction
	// header should contain Authorization: Bearer <token>
	if r == nil {
		return "", false, fmt.Errorf("empty request")
	}

	var header string
	if values, found := r.Header["Authorization"]; !found {
		return "", false, nil
	} else if len(values) != 1 {
		return "", false, nil
	} else {
		header = strings.Trim(values[0], " ")
	}

	if !strings.HasPrefix(header, "Bearer ") {
		return "", false, nil
	}

	var payload TokenPayload
	// parse token to find payload
	tokenValue := header[7:]
	payloadEncoded := strings.Split(tokenValue, ".")[1]
	if raw, err := base64.StdEncoding.DecodeString(payloadEncoded); err != nil {
		return "", false, errors.Join(err, errors.New("invalid value: "+payloadEncoded+"\t"+string([]byte(payloadEncoded)[92])))
	} else if err := json.Unmarshal(raw, &payload); err != nil {
		return "", false, err
	}

	login := payload.User
	var secret string
	if s, err := wrapper.Dao.FindSecretForActiveUser(wrapper.Ctx, login); err != nil {
		return "", false, err
	} else {
		secret = s
	}

	// details on why are here: https://pkg.go.dev/github.com/golang-jwt/jwt/v5#Keyfunc
	expectedSecretFunc := func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}

	token, err := jwt.Parse(tokenValue, expectedSecretFunc)
	switch {
	case token.Valid:
		return login, true, nil
	case errors.Is(err, jwt.ErrTokenMalformed):
		return login, false, fmt.Errorf("malformed token")
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return login, false, fmt.Errorf("invalid signature")
	case errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet):
		return login, false, fmt.Errorf("invalid token period")
	default:
		return login, false, err
	}
}
