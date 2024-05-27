package serving

import (
	"encoding/json"
	"io"
	"net/http"
)

// UserInformationInput is input for /token endpoint
type UserInformationInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// checkUserAndGenerateTokenHandler reads user data, and, if authentication matches, returns a token for this user
func checkUserAndGenerateTokenHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	var userInput UserInformationInput
	if body, errBody := io.ReadAll(r.Body); errBody != nil {
		return NewServiceUnprocessableEntityError(errBody.Error())
	} else if err := json.Unmarshal(body, &userInput); err != nil {
		return NewServiceUnprocessableEntityError(err.Error())
	} else if found, err := wrapper.Dao.CheckUser(wrapper.Ctx, userInput.Username, userInput.Password); err != nil {
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

	result := map[string]string{"token": newToken, "duration": TokenDuration.String()}
	json.NewEncoder(w).Encode(result)
	return nil
}
