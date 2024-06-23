package serving

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

var (
	// valid actions
	validActions = []string{"grant", "revoke"}
	// valid roles
	validRoles = []string{"manager", "modifier", "granter", "observer"}
	// valid classes
	validClasses = []string{"user", "graph"}
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

// listUserResourcesHandler lists user data and supervised users (to get their auth and id)
func listUserResourcesHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	currentUser, hasUser := wrapper.CurrentUser()
	if !hasUser {
		return NewServiceForbiddenError("need authentication")
	}

	if values, err := wrapper.Dao.ListUserDataAndSupervisedUsers(wrapper.Ctx, currentUser); err != nil {
		return BuildApiErrorFromStorageError(err)
	} else {
		json.NewEncoder(w).Encode(values)
	}

	return nil
}

// upsertUserHandler creates an user with that info, or updates its secret and password.
// It may apply to an user that wants to change its password.
func upsertUserHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	currentUser, hasUser := wrapper.CurrentUser()
	if !hasUser {
		return NewServiceForbiddenError("need authentication")
	}

	var userInput UserInformationInput
	if body, errBody := io.ReadAll(r.Body); errBody != nil {
		return NewServiceUnprocessableEntityError(errBody.Error())
	} else if err := json.Unmarshal(body, &userInput); err != nil {
		return NewServiceUnprocessableEntityError(err.Error())
	} else if err := wrapper.Dao.UpsertUser(wrapper.Ctx, currentUser, userInput.Username, userInput.Password); err == nil {
		return nil
	} else {
		// No out parameter, so deal with error message
		message := err.Error()
		if strings.Contains(message, "unauthorized") {
			return NewServiceForbiddenError(message)
		}

		return NewServiceInternalServerError(message)
	}
}

// lockUserHandler inactivates user if authorizations match
func lockUserHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	currentUser, hasUser := wrapper.CurrentUser()
	if !hasUser {
		return NewServiceForbiddenError("need authentication")
	}

	var userParameter string
	if userToLock, err := url.QueryUnescape(r.PathValue("userId")); err != nil {
		return NewServiceHttpClientError(err.Error())
	} else {
		userParameter = userToLock
	}

	errLocking := wrapper.Dao.LockUser(wrapper.Ctx, currentUser, userParameter)
	if errLocking != nil {
		return BuildApiErrorFromStorageError(errLocking)
	}

	return nil
}

// unlockUserHandler activates user if authorizations match.
func unlockUserHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	currentUser, hasUser := wrapper.CurrentUser()
	if !hasUser {
		return NewServiceForbiddenError("need authentication")
	}

	var userParameter string
	if userToUnlock, err := url.QueryUnescape(r.PathValue("userId")); err != nil {
		return NewServiceHttpClientError(err.Error())
	} else {
		userParameter = userToUnlock
	}

	errUnlocking := wrapper.Dao.UnlockUser(wrapper.Ctx, currentUser, userParameter)
	if errUnlocking != nil {
		return BuildApiErrorFromStorageError(errUnlocking)
	}

	return nil
}

// deleteUserHandler deletes the user
func deleteUserHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	currentUser, hasUser := wrapper.CurrentUser()
	if !hasUser {
		return NewServiceForbiddenError("need authentication")
	}

	userToDelete := r.PathValue("userId")
	errDelete := wrapper.Dao.DeleteUser(wrapper.Ctx, currentUser, userToDelete)
	if errDelete != nil {
		return BuildApiErrorFromStorageError(errDelete)
	}

	return nil
}

// manageAuthForSpecificResourceUserHandler grants or revoke access to a specific resource for a given user
func manageAuthForSpecificResourceUserHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	currentUser, hasUser := wrapper.CurrentUser()
	if !hasUser {
		return NewServiceForbiddenError("need authentication")
	}

	action := r.PathValue("action")
	role := r.PathValue("role")
	class := r.PathValue("class")
	if !slices.Contains(validActions, action) {
		return NewServiceHttpClientError("invalid action")
	} else if !slices.Contains(validRoles, role) {
		return NewServiceHttpClientError("invalid role")
	} else if !slices.Contains(validClasses, class) {
		return NewServiceHttpClientError("invalid class")
	}

	login := r.PathValue("login")
	resource := r.PathValue("resource")
	if len(login) == 0 {
		return NewServiceHttpClientError("invalid login")
	} else if len(resource) == 0 {
		return NewServiceHttpClientError("invalid resource")
	}

	switch {
	case action == "grant":
		return wrapper.Dao.GrantResourcesTo(wrapper.Ctx, currentUser, login, role, class, resource)
	case action == "revoke":
		return wrapper.Dao.RevokeResourcesTo(wrapper.Ctx, currentUser, login, role, class, resource)
	default:
		return NewServiceInternalServerError("invalid operation")
	}
}

// manageAuthForAllResourcesUserHandler grants or revoke access to all resources for a given user
func manageAuthForAllResourcesUserHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	currentUser, hasUser := wrapper.CurrentUser()
	if !hasUser {
		return NewServiceForbiddenError("need authentication")
	}

	action := r.PathValue("action")
	role := r.PathValue("role")
	class := r.PathValue("class")
	if !slices.Contains(validActions, action) {
		return NewServiceHttpClientError("invalid action")
	} else if !slices.Contains(validRoles, role) {
		return NewServiceHttpClientError("invalid role")
	} else if !slices.Contains(validClasses, class) {
		return NewServiceHttpClientError("invalid class")
	}

	login := r.PathValue("login")
	if len(login) == 0 {
		return NewServiceHttpClientError("invalid login")
	}

	switch {
	case action == "grant":
		return wrapper.Dao.GrantAllResourcesTo(wrapper.Ctx, currentUser, login, role, class)
	case action == "revoke":
		return wrapper.Dao.RevokeAllResourcesTo(wrapper.Ctx, currentUser, login, role, class)
	default:
		return NewServiceInternalServerError("invalid operation")
	}
}
