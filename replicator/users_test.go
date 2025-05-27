package replicator

import (
	"encoding/json"
	"errors"
	"github.com/cloudogu/sonarcarp/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestDefaultUserModifier_createOrGetUser(t *testing.T) {
	t.Run("Get user", func(t *testing.T) {
		r := newMockRequester(t)

		expReturnValue := currentUserResponse{Id: 1}
		b, err := json.Marshal(expReturnValue)
		require.NoError(t, err)

		r.EXPECT().Send(mock.Anything, mock.Anything).Return(&ResponseWithBody{
			Response: &http.Response{StatusCode: http.StatusOK},
			ByteBody: b,
		}, nil)

		um := defaultUserModifier{
			UserEndpoints: UserEndpoints{
				CreateUserEndpoint: "createUser",
				GetUserEndpoint:    "getUser/%s",
			},
			requester: r,
		}

		uID, err := um.createOrGetUser(internal.User{UserName: "test"})
		assert.NoError(t, err)
		assert.Equal(t, userId(1), uID)

		r.AssertNotCalled(t, "SendWithJsonBody")
	})

	t.Run("Unauthorized Get user request", func(t *testing.T) {
		r := newMockRequester(t)

		r.EXPECT().Send(mock.Anything, mock.Anything).Return(&ResponseWithBody{
			Response: &http.Response{StatusCode: http.StatusUnauthorized},
			ByteBody: nil,
		}, nil)

		um := defaultUserModifier{
			UserEndpoints: UserEndpoints{
				CreateUserEndpoint: "createUser",
				GetUserEndpoint:    "getUser/%s",
			},
			requester: r,
		}

		_, err := um.createOrGetUser(internal.User{UserName: "test"})
		assert.Error(t, err)

		r.AssertNotCalled(t, "SendWithJsonBody")
	})

	t.Run("Invalid json in Get user response", func(t *testing.T) {
		r := newMockRequester(t)

		r.EXPECT().Send(mock.Anything, mock.Anything).Return(&ResponseWithBody{
			Response: &http.Response{StatusCode: http.StatusOK},
			ByteBody: []byte("invalid#+.\""),
		}, nil)

		um := defaultUserModifier{
			UserEndpoints: UserEndpoints{
				CreateUserEndpoint: "createUser",
				GetUserEndpoint:    "getUser/%s",
			},
			requester: r,
		}

		_, err := um.createOrGetUser(internal.User{UserName: "test"})
		assert.Error(t, err)

		r.AssertNotCalled(t, "SendWithJsonBody")
	})

	t.Run("Requester Send returns error", func(t *testing.T) {
		r := newMockRequester(t)

		r.EXPECT().Send(mock.Anything, mock.Anything).Return(&ResponseWithBody{}, errors.New("testError"))

		um := defaultUserModifier{
			UserEndpoints: UserEndpoints{
				CreateUserEndpoint: "createUser",
				GetUserEndpoint:    "getUser/%s",
			},
			requester: r,
		}

		_, err := um.createOrGetUser(internal.User{UserName: "test"})
		assert.Error(t, err)

		r.AssertNotCalled(t, "SendWithJsonBody")
	})

	t.Run("Create user", func(t *testing.T) {
		r := newMockRequester(t)

		expReturnValue := createUserResponse{Id: 1}
		b, err := json.Marshal(expReturnValue)
		require.NoError(t, err)

		r.EXPECT().Send(mock.Anything, mock.Anything).Return(&ResponseWithBody{
			Response: &http.Response{StatusCode: http.StatusNotFound},
			ByteBody: nil,
		}, nil)

		r.EXPECT().SendWithJsonBody(http.MethodPost, "createUser", mock.AnythingOfType("userDTO")).Return(&ResponseWithBody{
			Response: &http.Response{StatusCode: http.StatusOK},
			ByteBody: b,
		}, nil)

		um := defaultUserModifier{
			UserEndpoints: UserEndpoints{
				CreateUserEndpoint: "createUser",
				GetUserEndpoint:    "getUser/%s",
			},
			requester: r,
		}

		u := internal.User{
			UserName:  "test",
			Replicate: false,
			Attributes: map[string][]string{
				"mail":        {"test@test.com"},
				"displayName": {"test_test"},
			},
		}

		uID, err := um.createOrGetUser(u)
		assert.NoError(t, err)
		assert.Equal(t, userId(1), uID)
	})

	t.Run("Unauthorized Create user request", func(t *testing.T) {
		r := newMockRequester(t)

		r.EXPECT().Send(mock.Anything, mock.Anything).Return(&ResponseWithBody{
			Response: &http.Response{StatusCode: http.StatusNotFound},
			ByteBody: nil,
		}, nil)

		r.EXPECT().SendWithJsonBody(http.MethodPost, "createUser", mock.AnythingOfType("userDTO")).Return(&ResponseWithBody{
			Response: &http.Response{StatusCode: http.StatusUnauthorized},
			ByteBody: nil,
		}, nil)

		um := defaultUserModifier{
			UserEndpoints: UserEndpoints{
				CreateUserEndpoint: "createUser",
				GetUserEndpoint:    "getUser/%s",
			},
			requester: r,
		}

		u := internal.User{
			UserName:  "test",
			Replicate: false,
			Attributes: map[string][]string{
				"mail":        {"test@test.com"},
				"displayName": {"test_test"},
			},
		}

		_, err := um.createOrGetUser(u)
		assert.Error(t, err)
	})

	t.Run("Invalid json in create user response", func(t *testing.T) {
		r := newMockRequester(t)

		r.EXPECT().Send(mock.Anything, mock.Anything).Return(&ResponseWithBody{
			Response: &http.Response{StatusCode: http.StatusNotFound},
			ByteBody: nil,
		}, nil)

		r.EXPECT().SendWithJsonBody(http.MethodPost, "createUser", mock.AnythingOfType("userDTO")).Return(&ResponseWithBody{
			Response: &http.Response{StatusCode: http.StatusOK},
			ByteBody: []byte("invalid#+.\""),
		}, nil)

		um := defaultUserModifier{
			UserEndpoints: UserEndpoints{
				CreateUserEndpoint: "createUser",
				GetUserEndpoint:    "getUser/%s",
			},
			requester: r,
		}

		u := internal.User{
			UserName:  "test",
			Replicate: false,
			Attributes: map[string][]string{
				"mail":        {"test@test.com"},
				"displayName": {"test_test"},
			},
		}

		_, err := um.createOrGetUser(u)
		assert.Error(t, err)
	})

	t.Run("Requester SendWithJsonBody returns error", func(t *testing.T) {
		r := newMockRequester(t)

		r.EXPECT().Send(mock.Anything, mock.Anything).Return(&ResponseWithBody{
			Response: &http.Response{StatusCode: http.StatusNotFound},
			ByteBody: nil,
		}, nil)

		r.EXPECT().
			SendWithJsonBody(http.MethodPost, "createUser", mock.AnythingOfType("userDTO")).
			Return(&ResponseWithBody{}, errors.New("createError"))

		um := defaultUserModifier{
			UserEndpoints: UserEndpoints{
				CreateUserEndpoint: "createUser",
				GetUserEndpoint:    "getUser/%s",
			},
			requester: r,
		}

		u := internal.User{
			UserName:  "test",
			Replicate: false,
			Attributes: map[string][]string{
				"mail":        {"test@test.com"},
				"displayName": {"test_test"},
			},
		}

		_, err := um.createOrGetUser(u)
		assert.Error(t, err)
	})
}
