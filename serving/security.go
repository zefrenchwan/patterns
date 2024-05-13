package serving

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Source for JWT implementation: https://medium.com/@cheickzida/golang-implementing-jwt-token-authentication-bba9bfd84d60

// createToken builds a new token for a given login using its secret
func createToken(userName string, userSecret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": userName,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

	if token, err := token.SignedString([]byte(userSecret)); err != nil {
		return "", err
	} else {
		return token, nil
	}
}

// UserInformationInput is input for /token endpoint
type UserInformationInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func checkAndGenerateToken(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	var userInput UserInformationInput
	if body, errBody := io.ReadAll(r.Body); errBody != nil {
		return NewServiceUnprocessableEntityError(errBody.Error())
	} else if err := json.Unmarshal(body, &userInput); err != nil {
		return NewServiceUnprocessableEntityError(err.Error())
	}

	if found, err := wrapper.Dao.CheckApiUser(wrapper.Ctx, userInput.Username, userInput.Password); err != nil {
		return NewServiceInternalServerError(err.Error())
	} else if !found {
		return NewServiceForbiddenError("invalid user")
	}

	// generate token
	var newToken string
	if secret, errLoad := wrapper.Dao.FindSecretForActiveUser(wrapper.Ctx, userInput.Username); errLoad != nil {
		return NewServiceInternalServerError(errLoad.Error())
	} else if token, err := createToken(userInput.Username, secret); err != nil {
		return NewServiceInternalServerError(err.Error())
	} else {
		newToken = token
	}

	json.NewEncoder(w).Encode(newToken)
	return nil
}
