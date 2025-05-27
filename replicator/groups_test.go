package replicator

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
)

var gEndpoints = GroupEndpoints{
	CreateUserGroupEndpoint:     "/api/teams",
	GetUserGroupEndpoint:        "/api/users/%v/teams",
	SearchGroupByNameEndpoint:   "/api/teams/search?name=%s",
	AddUserToGroupEndpoint:      "/api/teams/%v/members",
	RemoveUserFromGroupEndpoint: "/api/teams/%v/members/%v",
}

func TestGetUserGroups(t *testing.T) {
	tests := []struct {
		name        string
		rStatusCode int
		rValue      []byte
		rError      error
		expErr      bool
		expValue    userGroupResponse
	}{
		{
			name:        "get user groups",
			rStatusCode: http.StatusOK,
			rValue:      []byte("[{\"id\":1, \"name\": \"a\"}, {\"id\":2, \"name\": \"b\"}]"),
			rError:      nil,
			expErr:      false,
			expValue: userGroupResponse{
				{
					Id:   1,
					Name: "a",
				},
				{
					Id:   2,
					Name: "b",
				},
			},
		},
		{
			name:        "error while doing request",
			rStatusCode: 0,
			rValue:      nil,
			rError:      errors.New("connectionError"),
			expErr:      true,
			expValue:    nil,
		},
		{
			name:        "status code != ok",
			rStatusCode: http.StatusForbidden,
			rValue:      nil,
			rError:      nil,
			expErr:      true,
			expValue:    nil,
		},
		{
			name:        "invalid json",
			rStatusCode: http.StatusOK,
			rValue:      []byte("[{'id':1, 'name': 'a'}, {'id':2, 'name': 'b'}]"),
			rError:      nil,
			expErr:      true,
			expValue:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newMockRequester(t)
			gm := &defaultGroupModifier{
				GroupEndpoints: gEndpoints,
				requester:      r,
			}
			uID := 1

			r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
				ByteBody: tc.rValue,
				Response: &http.Response{StatusCode: tc.rStatusCode},
			}, tc.rError).Once()

			resp, err := gm.getUserGroups(1)

			if tc.expErr {
				assert.Error(t, err)
				assert.Nil(t, resp)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expValue, resp)
		})
	}
}

func TestCreateGroup(t *testing.T) {
	tests := []struct {
		name        string
		rStatusCode int
		rValue      []byte
		rError      error
		expErr      bool
	}{
		{
			name:        "create group",
			rStatusCode: http.StatusOK,
			rValue:      []byte("{\"message\":\"\",\"teamId\":3}"),
			rError:      nil,
			expErr:      false,
		},
		{
			name:        "status code != 200",
			rStatusCode: http.StatusForbidden,
			rValue:      nil,
			rError:      nil,
			expErr:      true,
		},
		{
			name:        "invalid json in response",
			rStatusCode: http.StatusOK,
			rValue:      []byte("{'message':\"\",\"teamId\":3}"),
			rError:      nil,
			expErr:      true,
		},
		{
			name:        "connection error",
			rStatusCode: 0,
			rValue:      nil,
			rError:      errors.New("connectionError"),
			expErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newMockRequester(t)
			gm := &defaultGroupModifier{
				GroupEndpoints: gEndpoints,
				requester:      r,
			}

			r.On("SendWithJsonBody", http.MethodPost, gEndpoints.CreateUserGroupEndpoint, group{Name: "c"}).Return(
				&ResponseWithBody{
					ByteBody: tc.rValue,
					Response: &http.Response{
						StatusCode: tc.rStatusCode,
					},
				},
				tc.rError,
			).Once()

			gID, err := gm.createGroup("c")

			if tc.expErr {
				assert.Error(t, err)
				assert.Equal(t, 0, int(gID))

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, groupId(3), gID)
		})
	}
}

func TestGetGroupIDByName(t *testing.T) {
	tests := []struct {
		name        string
		rStatusCode int
		rValue      []byte
		rError      error
		expErr      bool
		err         error
	}{
		{
			name:        "get groupID by name",
			rStatusCode: http.StatusOK,
			rValue:      []byte("{\"totalCount\": 1, \"teams\": [{\"id\":4, \"name\": \"d\"}]}"),
			rError:      nil,
			expErr:      false,
			err:         nil,
		},
		{
			name:        "return status code != ok || not found",
			rStatusCode: http.StatusForbidden,
			rValue:      nil,
			rError:      nil,
			expErr:      true,
			err:         nil,
		},
		{
			name:        "return status code 404",
			rStatusCode: http.StatusNotFound,
			rValue:      nil,
			rError:      nil,
			expErr:      true,
			err:         ErrNotFound,
		},
		{
			name:        "invalid json in response",
			rStatusCode: http.StatusOK,
			rValue:      []byte("{'totalCount': 1, 'teams': [{'id':4, 'name': 'd'}]}"),
			rError:      nil,
			expErr:      true,
			err:         nil,
		},
		{
			name:        "totalCount = 0",
			rStatusCode: http.StatusOK,
			rValue:      []byte("{\"totalCount\": 0, \"teams\": []}"),
			rError:      nil,
			expErr:      true,
			err:         ErrNotFound,
		},
		{
			name:        "multiple groups with same name",
			rStatusCode: http.StatusOK,
			rValue:      []byte("{\"totalCount\": 2, \"teams\": [{\"id\":4, \"name\": \"d\"}, {\"id\":5, \"name\": \"d\"}]}"),
			rError:      nil,
			expErr:      true,
			err:         nil,
		},
		{
			name:        "connection error",
			rStatusCode: 0,
			rValue:      nil,
			rError:      errors.New("connectionError"),
			expErr:      true,
			err:         nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newMockRequester(t)
			gm := &defaultGroupModifier{
				GroupEndpoints: gEndpoints,
				requester:      r,
			}

			r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "d")).Return(&ResponseWithBody{
				ByteBody: tc.rValue,
				Response: &http.Response{
					StatusCode: tc.rStatusCode,
				},
			}, tc.rError).Once()

			gID, err := gm.getGroupIDByName("d")

			if tc.expErr {
				assert.Error(t, err)
				assert.Equal(t, 0, int(gID))

				if tc.err != nil {
					assert.Equal(t, tc.err, err)
				}

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, groupId(4), gID)
		})
	}
}

func TestAddUserToGroup(t *testing.T) {
	tests := []struct {
		name        string
		rStatusCode int
		rError      error
		expErr      bool
	}{
		{
			name:        "add user to group",
			rStatusCode: http.StatusOK,
			rError:      nil,
			expErr:      false,
		},
		{
			name:        "connection error",
			rStatusCode: 0,
			rError:      errors.New("connectionError"),
			expErr:      true,
		},
		{
			name:        "return status code 404",
			rStatusCode: http.StatusNotFound,
			rError:      nil,
			expErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newMockRequester(t)
			gm := &defaultGroupModifier{
				GroupEndpoints: gEndpoints,
				requester:      r,
			}

			r.On("SendWithJsonBody", http.MethodPost, fmt.Sprintf(gEndpoints.AddUserToGroupEndpoint, "4"), teamMember{UserId: 1}).Return(
				&ResponseWithBody{
					ByteBody: nil,
					Response: &http.Response{
						StatusCode: tc.rStatusCode,
					},
				},
				tc.rError).Once()

			err := gm.addUserToGroup(userId(1), groupId(4))

			if tc.expErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestRemoveUserFromGroup(t *testing.T) {
	tests := []struct {
		name        string
		rStatusCode int
		rError      error
		expErr      bool
	}{
		{
			name:        "remove user from group",
			rStatusCode: http.StatusOK,
			rError:      nil,
			expErr:      false,
		},
		{
			name:        "connection error",
			rStatusCode: 0,
			rError:      errors.New("connectionError"),
			expErr:      true,
		},
		{
			name:        "return status code 403",
			rStatusCode: http.StatusForbidden,
			rError:      nil,
			expErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newMockRequester(t)
			gm := &defaultGroupModifier{
				GroupEndpoints: gEndpoints,
				requester:      r,
			}

			r.On("Send", http.MethodDelete, fmt.Sprintf(gEndpoints.RemoveUserFromGroupEndpoint, "4", "1")).Return(
				&ResponseWithBody{
					ByteBody: nil,
					Response: &http.Response{
						StatusCode: tc.rStatusCode,
					},
				},
				tc.rError).Once()

			err := gm.removeGroupFromUser(userId(1), groupId(4))

			if tc.expErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestCreateOrUpdateGroup(t *testing.T) {
	t.Run("group already exists", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "d")).Return(&ResponseWithBody{
			ByteBody: []byte("{\"totalCount\": 1, \"teams\": [{\"id\":4, \"name\": \"d\"}]}"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		gID, err := gm.createOrUpdateGroup("d")
		assert.NoError(t, err)
		assert.Equal(t, groupId(4), gID)

		r.AssertNotCalled(t, "SendWithJsonBody", mock.Anything, mock.Anything, mock.Anything)
		r.AssertExpectations(t)
	})

	t.Run("create group", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "d")).Return(&ResponseWithBody{
			ByteBody: nil,
			Response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
		}, nil).Once()

		r.On("SendWithJsonBody", http.MethodPost, gEndpoints.CreateUserGroupEndpoint, group{Name: "d"}).Return(
			&ResponseWithBody{
				ByteBody: []byte("{\"message\":\"\",\"teamId\":4}"),
				Response: &http.Response{
					StatusCode: http.StatusOK,
				},
			},
			nil,
		).Once()

		gID, err := gm.createOrUpdateGroup("d")
		assert.NoError(t, err)
		assert.Equal(t, groupId(4), gID)

		r.AssertExpectations(t)
	})

	t.Run("forbidden access to groups", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "d")).Return(&ResponseWithBody{
			ByteBody: nil,
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
			},
		}, nil).Once()

		gID, err := gm.createOrUpdateGroup("d")
		assert.Error(t, err)
		assert.Equal(t, groupId(0), gID)

		r.AssertNotCalled(t, "SendWithJsonBody", mock.Anything, mock.Anything, mock.Anything)
		r.AssertExpectations(t)
	})

	t.Run("forbidden to write to groups", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "d")).Return(&ResponseWithBody{
			ByteBody: nil,
			Response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
		}, nil).Once()

		r.On("SendWithJsonBody", http.MethodPost, gEndpoints.CreateUserGroupEndpoint, group{Name: "d"}).Return(
			&ResponseWithBody{
				ByteBody: nil,
				Response: &http.Response{
					StatusCode: http.StatusForbidden,
				},
			},
			nil,
		).Once()

		gID, err := gm.createOrUpdateGroup("d")
		assert.Error(t, err)
		assert.Equal(t, groupId(0), gID)

		r.AssertExpectations(t)
	})
}

func TestAddMissingGroups(t *testing.T) {
	t.Run("Add missing group", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}
		uID := 1
		a := []string{"a", "b", "c", "d"}

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
			ByteBody: []byte("[{\"id\":1, \"name\": \"a\"}, {\"id\":2, \"name\": \"b\"}]"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "c")).Return(&ResponseWithBody{
			ByteBody: []byte(""),
			Response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
		}, nil).Once()

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "d")).Return(&ResponseWithBody{
			ByteBody: []byte("{\"totalCount\": 1, \"teams\": [{\"id\":4, \"name\": \"d\"}]}"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("SendWithJsonBody", http.MethodPost, gEndpoints.CreateUserGroupEndpoint, group{Name: "c"}).Return(
			&ResponseWithBody{
				ByteBody: []byte("{\"message\":\"\",\"teamId\":3}"),
				Response: &http.Response{
					StatusCode: http.StatusOK,
				},
			},
			nil,
		).Once()

		r.On("SendWithJsonBody", http.MethodPost, fmt.Sprintf(gEndpoints.AddUserToGroupEndpoint, "3"), teamMember{UserId: 1}).Return(
			&ResponseWithBody{
				ByteBody: []byte(""),
				Response: &http.Response{
					StatusCode: http.StatusOK,
				},
			},
			nil,
		).Once()

		r.On("SendWithJsonBody", http.MethodPost, fmt.Sprintf(gEndpoints.AddUserToGroupEndpoint, "4"), teamMember{UserId: 1}).Return(
			&ResponseWithBody{
				ByteBody: []byte(""),
				Response: &http.Response{
					StatusCode: http.StatusOK,
				},
			},
			nil,
		).Once()

		err := gm.addMissingGroups(1, a)
		assert.NoError(t, err)

		r.AssertExpectations(t)
	})

	t.Run("no group update needed", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}
		uID := 1
		a := []string{"a", "b", "c", "d"}

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
			ByteBody: []byte("[{\"id\":1, \"name\": \"a\"}, {\"id\":2, \"name\": \"b\"}, {\"id\":3, \"name\": \"c\"}, {\"id\":4, \"name\": \"d\"}]"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		err := gm.addMissingGroups(1, a)
		assert.NoError(t, err)

		r.AssertNotCalled(t, "SendWithJsonBody")
	})

	t.Run("could not get user groups", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}
		uID := 1
		a := []string{"a", "b", "c", "d"}

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
			ByteBody: nil,
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
			},
		}, nil).Once()

		err := gm.addMissingGroups(1, a)
		assert.Error(t, err)

		r.AssertNotCalled(t, "Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "c"))
		r.AssertNotCalled(t, "Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "d"))
		r.AssertNotCalled(t, "SendWithJsonBody", http.MethodPost, gEndpoints.CreateUserGroupEndpoint, group{Name: "c"})
		r.AssertNotCalled(t, "SendWithJsonBody", http.MethodPost, fmt.Sprintf(gEndpoints.AddUserToGroupEndpoint, "3"), teamMember{UserId: 1})
		r.AssertNotCalled(t, "SendWithJsonBody", http.MethodPost, fmt.Sprintf(gEndpoints.AddUserToGroupEndpoint, "4"), teamMember{UserId: 1})
		r.AssertExpectations(t)
	})

	t.Run("dont add member to group when creation of group fails", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}
		uID := 1
		a := []string{"a", "b", "c", "d"}

		cErr := errors.New("createOrUpdateError")

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
			ByteBody: []byte("[{\"id\":1, \"name\": \"a\"}, {\"id\":2, \"name\": \"b\"}]"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "c")).Return(&ResponseWithBody{
			ByteBody: []byte(""),
			Response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
		}, nil).Once()

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "d")).Return(&ResponseWithBody{
			ByteBody: []byte("{\"totalCount\": 1, \"teams\": [{\"id\":4, \"name\": \"d\"}]}"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("SendWithJsonBody", http.MethodPost, gEndpoints.CreateUserGroupEndpoint, group{Name: "c"}).Return(
			nil, cErr).Once()

		r.On("SendWithJsonBody", http.MethodPost, fmt.Sprintf(gEndpoints.AddUserToGroupEndpoint, "4"), teamMember{UserId: 1}).Return(
			&ResponseWithBody{
				ByteBody: []byte(""),
				Response: &http.Response{
					StatusCode: http.StatusOK,
				},
			},
			nil,
		).Once()

		err := gm.addMissingGroups(1, a)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, cErr))

		r.AssertNotCalled(t, "SendWithJsonBody", http.MethodPost, fmt.Sprintf(gEndpoints.AddUserToGroupEndpoint, "3"), teamMember{UserId: 1})
		r.AssertExpectations(t)
	})

	t.Run("create group fails and adding member to another group fails", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}
		uID := 1
		a := []string{"a", "b", "c", "d"}

		cErr := errors.New("createOrUpdateError")
		aErr := errors.New("addMemberToGroupErr")

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
			ByteBody: []byte("[{\"id\":1, \"name\": \"a\"}, {\"id\":2, \"name\": \"b\"}]"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "c")).Return(&ResponseWithBody{
			ByteBody: []byte(""),
			Response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
		}, nil).Once()

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "d")).Return(&ResponseWithBody{
			ByteBody: []byte("{\"totalCount\": 1, \"teams\": [{\"id\":4, \"name\": \"d\"}]}"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("SendWithJsonBody", http.MethodPost, gEndpoints.CreateUserGroupEndpoint, group{Name: "c"}).Return(
			nil, cErr).Once()

		r.On("SendWithJsonBody", http.MethodPost, fmt.Sprintf(gEndpoints.AddUserToGroupEndpoint, "4"), teamMember{UserId: 1}).Return(
			nil, aErr).Once()

		err := gm.addMissingGroups(1, a)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, cErr))
		assert.True(t, errors.Is(err, aErr))

		r.AssertNotCalled(t, "SendWithJsonBody", http.MethodPost, fmt.Sprintf(gEndpoints.AddUserToGroupEndpoint, "3"), teamMember{UserId: 1})
		r.AssertExpectations(t)
	})
}

func TestRemoveNonExistingGroups(t *testing.T) {
	t.Run("remove non existing groups", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}

		uID := 1
		a := []string{"a"}

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
			ByteBody: []byte("[{\"id\":1, \"name\": \"a\"}, {\"id\":2, \"name\": \"b\"}]"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "b")).Return(&ResponseWithBody{
			ByteBody: []byte("{\"totalCount\": 1, \"teams\": [{\"id\":2, \"name\": \"b\"}]}"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("Send", http.MethodDelete, fmt.Sprintf(gEndpoints.RemoveUserFromGroupEndpoint, "2", "1")).Return(
			&ResponseWithBody{
				ByteBody: nil,
				Response: &http.Response{
					StatusCode: http.StatusOK,
				},
			},
			nil).Once()

		err := gm.removeNonExistingGroups(1, a)
		assert.NoError(t, err)

		r.AssertExpectations(t)
	})

	t.Run("no group updated needed", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}

		uID := 1
		a := []string{"a", "b"}

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
			ByteBody: []byte("[{\"id\":1, \"name\": \"a\"}, {\"id\":2, \"name\": \"b\"}]"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		err := gm.removeNonExistingGroups(1, a)
		assert.NoError(t, err)

		r.AssertNotCalled(t, "Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "a"))
		r.AssertNotCalled(t, "Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "b"))
		r.AssertNotCalled(t, "Send", http.MethodDelete, fmt.Sprintf(gEndpoints.RemoveUserFromGroupEndpoint, "1", "1"))
		r.AssertNotCalled(t, "Send", http.MethodDelete, fmt.Sprintf(gEndpoints.RemoveUserFromGroupEndpoint, "2", "1"))

		r.AssertExpectations(t)
	})

	t.Run("could not read user groups", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}

		uID := 1
		a := []string{"a", "b"}

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
			ByteBody: nil,
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
			},
		}, nil).Once()

		err := gm.removeNonExistingGroups(1, a)
		assert.Error(t, err)

		r.AssertNotCalled(t, "Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "a"))
		r.AssertNotCalled(t, "Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "b"))
		r.AssertNotCalled(t, "Send", http.MethodDelete, fmt.Sprintf(gEndpoints.RemoveUserFromGroupEndpoint, "1", "1"))
		r.AssertNotCalled(t, "Send", http.MethodDelete, fmt.Sprintf(gEndpoints.RemoveUserFromGroupEndpoint, "2", "1"))

		r.AssertExpectations(t)
	})

	t.Run("dont remove member in case group does not exist", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}

		uID := 1
		a := []string{"a"}

		cErr := errors.New("createOrUpdateError")

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
			ByteBody: []byte("[{\"id\":1, \"name\": \"a\"}, {\"id\":2, \"name\": \"b\"}]"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "b")).Return(nil, cErr).Once()

		err := gm.removeNonExistingGroups(1, a)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, cErr))

		r.AssertNotCalled(t, "Send", http.MethodDelete, fmt.Sprintf(gEndpoints.RemoveUserFromGroupEndpoint, "2", "1"))
		r.AssertExpectations(t)
	})

	t.Run("get group fails and removing member from another group fails", func(t *testing.T) {
		r := newMockRequester(t)
		gm := &defaultGroupModifier{
			GroupEndpoints: gEndpoints,
			requester:      r,
		}

		uID := 1
		a := []string{"a"}

		cErr := errors.New("createOrUpdateError")
		rErr := errors.New("removeMemberFromGroupErr")

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.GetUserGroupEndpoint, uID)).Return(&ResponseWithBody{
			ByteBody: []byte("[{\"id\":1, \"name\": \"a\"}, {\"id\":2, \"name\": \"b\"}, {\"id\":3, \"name\": \"c\"}]"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "b")).Return(&ResponseWithBody{
			ByteBody: []byte("{\"totalCount\": 1, \"teams\": [{\"id\":2, \"name\": \"b\"}]}"),
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil).Once()

		r.On("Send", http.MethodGet, fmt.Sprintf(gEndpoints.SearchGroupByNameEndpoint, "c")).Return(nil, cErr).Once()

		r.On("Send", http.MethodDelete, fmt.Sprintf(gEndpoints.RemoveUserFromGroupEndpoint, "2", "1")).Return(nil, rErr).Once()

		err := gm.removeNonExistingGroups(1, a)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, cErr))
		assert.True(t, errors.Is(err, rErr))

		r.AssertNotCalled(t, "Send", http.MethodDelete, fmt.Sprintf(gEndpoints.RemoveUserFromGroupEndpoint, "3", "1"))

		r.AssertExpectations(t)
	})

}
