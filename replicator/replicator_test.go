package replicator

import (
	"context"
	"errors"
	"github.com/cloudogu/sonarcarp/internal"
	"github.com/cloudogu/sonarcarp/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewReplicator(t *testing.T) {
	cfg := Configuration{
		RequestUserName:     "Admin",
		RequestUserPassword: "Admin",
		Endpoints: Endpoints{
			UserEndpoints: UserEndpoints{
				CreateUserEndpoint: "createUser",
				GetUserEndpoint:    "getUser",
			},
			GroupEndpoints: GroupEndpoints{
				CreateUserGroupEndpoint:     "createGroup",
				GetUserGroupEndpoint:        "getGroup",
				SearchGroupByNameEndpoint:   "searchGroupByName",
				AddUserToGroupEndpoint:      "addUserToGroup",
				RemoveUserFromGroupEndpoint: "removeUserFromGroup",
			},
		},
	}

	r := NewReplicator(cfg)
	assert.NotNil(t, r)
}

func TestDefaultReplicator_Replicate(t *testing.T) {
	t.Run("Replicate without warnings", func(t *testing.T) {
		const uID = userId(1)

		lbm, reset := mocks.CreateLoggingMock(log)
		defer reset()

		uModifier := newMockUserModifier(t)
		gModifier := newMockGroupModifier(t)

		uModifier.EXPECT().createOrGetUser(mock.Anything).Return(uID, nil)
		gModifier.EXPECT().addMissingGroups(uID, mock.Anything).Return(nil)
		gModifier.EXPECT().removeNonExistingGroups(uID, mock.Anything).Return(nil)

		d := DefaultReplicator{
			userModifier:  uModifier,
			groupModifier: gModifier,
		}

		d.Replicate(internal.User{UserName: "test"})

		assert.Equal(t, 0, lbm.WarningCalls)
	})

	t.Run("unable to create or get user", func(t *testing.T) {
		lbm, reset := mocks.CreateLoggingMock(log)
		defer reset()

		uModifier := newMockUserModifier(t)
		gModifier := newMockGroupModifier(t)

		uModifier.EXPECT().createOrGetUser(mock.Anything).Return(0, errors.New("createOrGetUserError"))

		d := DefaultReplicator{
			userModifier:  uModifier,
			groupModifier: gModifier,
		}

		d.Replicate(internal.User{UserName: "test"})

		gModifier.AssertNotCalled(t, "addMissingGroups")
		gModifier.AssertNotCalled(t, "removeNonExistingGroups")

		assert.Equal(t, 1, lbm.WarningCalls)
	})

	t.Run("addMissingGroups returns error", func(t *testing.T) {
		const uID = userId(1)

		lbm, reset := mocks.CreateLoggingMock(log)
		defer reset()

		uModifier := newMockUserModifier(t)
		gModifier := newMockGroupModifier(t)

		uModifier.EXPECT().createOrGetUser(mock.Anything).Return(uID, nil)
		gModifier.EXPECT().addMissingGroups(uID, mock.Anything).Return(errors.New("addMissingGroupsError"))
		gModifier.EXPECT().removeNonExistingGroups(uID, mock.Anything).Return(nil)

		d := DefaultReplicator{
			userModifier:  uModifier,
			groupModifier: gModifier,
		}

		d.Replicate(internal.User{UserName: "test"})

		assert.Equal(t, 1, lbm.WarningCalls)
	})

	t.Run("removeNonExistingGroups returns error", func(t *testing.T) {
		const uID = userId(1)

		lbm, reset := mocks.CreateLoggingMock(log)
		defer reset()

		uModifier := newMockUserModifier(t)
		gModifier := newMockGroupModifier(t)

		uModifier.EXPECT().createOrGetUser(mock.Anything).Return(uID, nil)
		gModifier.EXPECT().addMissingGroups(uID, mock.Anything).Return(nil)
		gModifier.EXPECT().removeNonExistingGroups(uID, mock.Anything).Return(errors.New("removeNonExistingGroupsError"))

		d := DefaultReplicator{
			userModifier:  uModifier,
			groupModifier: gModifier,
		}

		d.Replicate(internal.User{UserName: "test"})

		assert.Equal(t, 1, lbm.WarningCalls)
	})

	t.Run("addMissingGroups and removeNonExistingGroups return errors", func(t *testing.T) {
		const uID = userId(1)

		lbm, reset := mocks.CreateLoggingMock(log)
		defer reset()

		uModifier := newMockUserModifier(t)
		gModifier := newMockGroupModifier(t)

		uModifier.EXPECT().createOrGetUser(mock.Anything).Return(uID, nil)
		gModifier.EXPECT().addMissingGroups(uID, mock.Anything).Return(errors.New("addMissingGroupsError"))
		gModifier.EXPECT().removeNonExistingGroups(uID, mock.Anything).Return(errors.New("removeNonExistingGroupsError"))

		d := DefaultReplicator{
			userModifier:  uModifier,
			groupModifier: gModifier,
		}

		d.Replicate(internal.User{UserName: "test"})

		assert.Equal(t, 2, lbm.WarningCalls)
	})
}

func TestCreateMiddleware(t *testing.T) {
	middleware := CreateMiddleware(Configuration{})
	assert.NotNil(t, middleware)
}

func TestReplicateMiddleware(t *testing.T) {
	t.Run("No user in context", func(t *testing.T) {
		uModifier := newMockUserModifier(t)
		gModifier := newMockGroupModifier(t)

		rWriter := httptest.NewRecorder()

		replicator := &DefaultReplicator{
			userModifier:  uModifier,
			groupModifier: gModifier,
		}

		mh := &mocks.Handler{}
		mh.On("ServeHTTP", mock.Anything, mock.Anything)

		req, err := http.NewRequest(http.MethodGet, "dummyURL", nil)
		require.NoError(t, err)

		m := createMiddleware(replicator)
		m(mh).ServeHTTP(rWriter, req)

		mh.AssertCalled(t, "ServeHTTP")
		uModifier.AssertNotCalled(t, "createOrGetUser")
		gModifier.AssertNotCalled(t, "addMissingGroups")
		gModifier.AssertNotCalled(t, "removeNonExistingGroups")
	})

	t.Run("User replicate not necessary", func(t *testing.T) {
		uModifier := newMockUserModifier(t)
		gModifier := newMockGroupModifier(t)

		rWriter := httptest.NewRecorder()

		replicator := &DefaultReplicator{
			userModifier:  uModifier,
			groupModifier: gModifier,
		}

		mh := &mocks.Handler{}
		mh.On("ServeHTTP", mock.Anything, mock.Anything)

		user := internal.User{
			UserName:  "test",
			Replicate: false,
		}

		userCtx := internal.WithUser(context.TODO(), user)
		replicateMiddleware := createMiddleware(replicator)

		req, err := http.NewRequestWithContext(userCtx, http.MethodGet, "dummyURL", nil)
		require.NoError(t, err)

		replicateMiddleware(mh).ServeHTTP(rWriter, req)

		mh.AssertCalled(t, "ServeHTTP")
		uModifier.AssertNotCalled(t, "createOrGetUser")
		gModifier.AssertNotCalled(t, "addMissingGroups")
		gModifier.AssertNotCalled(t, "removeNonExistingGroups")
	})

	t.Run("replicate user", func(t *testing.T) {
		uModifier := newMockUserModifier(t)
		gModifier := newMockGroupModifier(t)

		rWriter := httptest.NewRecorder()

		uModifier.EXPECT().createOrGetUser(mock.Anything).Return(1, nil)
		gModifier.EXPECT().addMissingGroups(mock.Anything, mock.Anything).Return(nil)
		gModifier.EXPECT().removeNonExistingGroups(mock.Anything, mock.Anything).Return(nil)

		replicator := &DefaultReplicator{
			userModifier:  uModifier,
			groupModifier: gModifier,
		}

		mh := &mocks.Handler{}
		mh.On("ServeHTTP", mock.Anything, mock.Anything)

		user := internal.User{
			UserName:  "test",
			Replicate: true,
		}

		userCtx := internal.WithUser(context.TODO(), user)
		replicateMiddleware := createMiddleware(replicator)

		req, err := http.NewRequestWithContext(userCtx, http.MethodGet, "dummyURL", nil)
		require.NoError(t, err)

		replicateMiddleware(mh).ServeHTTP(rWriter, req)

		mh.AssertCalled(t, "ServeHTTP")
	})
}
