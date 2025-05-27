package replicator

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudogu/sonarcarp/internal"
	"github.com/cloudogu/sonarcarp/random"
	"net/http"
)

type userId int

type userDTO struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

type currentUserResponse struct {
	Id    userId `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Login string `json:"login"`
}
type createUserResponse struct {
	Id      userId `json:"id"`
	Message string `json:"message"`
}

type UserEndpoints struct {
	CreateUserEndpoint string
	GetUserEndpoint    string
}

type defaultUserModifier struct {
	UserEndpoints
	requester
}

func (u defaultUserModifier) createOrGetUser(ctxUser internal.User) (userId, error) {
	userID, err := u.getUser(ctxUser.UserName)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return 0, fmt.Errorf("unable to get user: %w", err)
		}

		return u.createUser(ctxUser)
	}

	return userID, nil
}

func (u defaultUserModifier) getUser(userName string) (userId, error) {
	getUserURL := fmt.Sprintf(u.GetUserEndpoint, userName)

	resp, err := u.requester.Send(http.MethodGet, getUserURL)
	if err != nil {
		return 0, fmt.Errorf("failed to do get request to user endpoint %s: %w", getUserURL, err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return 0, ErrNotFound
		}

		return 0, fmt.Errorf("received unexpected error code in response: %d", resp.StatusCode)
	}

	var result currentUserResponse

	err = json.Unmarshal(resp.ByteBody, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal user response: %w", err)
	}

	return result.Id, nil
}

func (u defaultUserModifier) createUser(user internal.User) (userId, error) {
	randomStr, err := random.String(64)
	if err != nil {
		return 0, err
	}
	uDTO := userDTO{
		Email:    user.GetMail(),
		Login:    user.UserName,
		Name:     user.GetDisplayName(),
		Password: randomStr,
	}

	resp, err := u.requester.SendWithJsonBody(http.MethodPost, u.CreateUserEndpoint, uDTO)
	if err != nil {
		return 0, fmt.Errorf("could not perform post request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("error while creating user: %w", err)
	}

	var result createUserResponse

	err = json.Unmarshal(resp.ByteBody, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal user response: %w", err)
	}

	return result.Id, nil
}
