package replicator

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
)

type groupId int

type group struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type createGroupResponse struct {
	Message string  `json:"message"`
	GroupId groupId `json:"teamId"`
}

type userGroupResponse []struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type teamMember struct {
	UserId userId `json:"UserId"`
}

type groupByNameResponse struct {
	TotalCount int `json:"totalCount"`
	Groups     []struct {
		Id   groupId `json:"id"`
		Name string  `json:"name"`
	} `json:"teams"`
}

type GroupEndpoints struct {
	CreateUserGroupEndpoint     string
	GetUserGroupEndpoint        string
	SearchGroupByNameEndpoint   string
	AddUserToGroupEndpoint      string
	RemoveUserFromGroupEndpoint string
}

type defaultGroupModifier struct {
	GroupEndpoints
	requester
}

func groupInGroupsResponse(a string, groups userGroupResponse) bool {
	for _, b := range groups {
		if b.Name == a {
			return true
		}
	}
	return false
}

func (d defaultGroupModifier) addMissingGroups(userId userId, groups []string) error {
	log.Debugf("Add missing groups to user with id %v, groups: %v", userId, groups)

	currentGroups, err := d.getUserGroups(userId)
	if err != nil {
		return fmt.Errorf("failed to get groups of user with id '%v': %w", userId, err)
	}
	log.Debugf("Current groups: %v", currentGroups)

	var multiErr []error
	for _, g := range groups {
		if groupInGroupsResponse(g, currentGroups) {
			log.Debugf("User with id '%v' is already member of group '%s'", userId, g)

			continue
		}

		log.Debugf("User not in group '%v', try to add group to user", g)
		gId, lErr := d.createOrUpdateGroup(g)
		if lErr != nil {
			log.Debugf("Error creating group '%s': '%v'", g, lErr)
			multiErr = append(multiErr, lErr)

			continue
		}

		lErr = d.addUserToGroup(userId, gId)
		if lErr != nil {
			log.Debugf("Error adding group '%s' to user: %v", g, lErr)
			multiErr = append(multiErr, lErr)
		}
	}

	if len(multiErr) > 0 {
		return fmt.Errorf("the following errors occured on missing group add: %w", errors.Join(multiErr...))
	}

	return nil
}

func (d defaultGroupModifier) removeNonExistingGroups(userId userId, groups []string) error {
	log.Debugf("Remove non existing groups of user with id '%v', groups: %v", userId, groups)

	currentGroups, err := d.getUserGroups(userId)
	if err != nil {
		return fmt.Errorf("failed to get the groups of user with id '%v': %w", userId, err)
	}
	log.Debugf("Got current groups: %v", currentGroups)

	var multiErr []error
	for _, g := range currentGroups {
		if slices.Contains(groups, g.Name) {
			log.Debugf("The group '%s' exists in identity provider as well as locally and the user with id '%v' does not need to be updated", g.Name, userId)

			continue
		}

		log.Debugf("Try to remove group '%s' from user with id %v", g.Name, userId)
		gID, lErr := d.createOrUpdateGroup(g.Name)
		if lErr != nil {
			log.Debugf("Error getting group id: %v", g.Name)
			multiErr = append(multiErr, lErr)

			continue
		}

		lErr = d.removeGroupFromUser(userId, gID)
		if lErr != nil {
			log.Debugf("Error removing group '%s' from user with id %v", g.Name, userId)
			multiErr = append(multiErr, lErr)
		}
	}

	if len(multiErr) > 0 {
		return fmt.Errorf("the following errors occured on non existing group remove: %w", errors.Join(multiErr...))
	}

	return nil
}

func (d defaultGroupModifier) createOrUpdateGroup(groupName string) (groupId, error) {
	log.Debugf("Create or update group with name '%s'", groupName)

	gID, err := d.getGroupIDByName(groupName)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return 0, fmt.Errorf("unable to get group with name %s: %w", groupName, err)
		}

		return d.createGroup(groupName)
	}

	return gID, nil
}

func (d defaultGroupModifier) getGroupIDByName(gName string) (groupId, error) {
	groupSearchURL := fmt.Sprintf(d.SearchGroupByNameEndpoint, gName)
	resp, err := d.requester.Send(http.MethodGet, groupSearchURL)
	if err != nil {
		return 0, fmt.Errorf("could get group %s: %w", groupSearchURL, err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return 0, ErrNotFound
		}

		return 0, fmt.Errorf("received unexpected error code in response: %d", resp.StatusCode)
	}

	var result groupByNameResponse
	err = json.Unmarshal(resp.ByteBody, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal group response: %w", err)
	}

	if result.TotalCount == 0 {
		return 0, ErrNotFound
	}

	if result.TotalCount > 1 {
		return 0, fmt.Errorf("found multiple groups with name %s, count: %d", gName, result.TotalCount)
	}

	return result.Groups[0].Id, nil
}

func (d defaultGroupModifier) createGroup(gName string) (groupId, error) {
	newGroup := group{Name: gName}

	resp, err := d.requester.SendWithJsonBody(http.MethodPost, d.CreateUserGroupEndpoint, newGroup)
	if err != nil {
		return 0, fmt.Errorf("could not do create request for group %s: %w", gName, err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("could not create group, received error code: %d", resp.StatusCode)
	}

	var result createGroupResponse
	err = json.Unmarshal(resp.ByteBody, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return result.GroupId, nil
}

func (d defaultGroupModifier) addUserToGroup(userId userId, groupId groupId) error {
	log.Debugf("Add user with id '%v' to group with id '%v'", userId, groupId)
	newTeamMember := teamMember{
		UserId: userId,
	}

	addMemberToGroupURL := fmt.Sprintf(d.AddUserToGroupEndpoint, groupId)

	resp, err := d.requester.SendWithJsonBody(http.MethodPost, addMemberToGroupURL, newTeamMember)
	if err != nil {
		return fmt.Errorf("failed to execute post request to add user to group endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got response, but status code is not ok: %d", resp.StatusCode)
	}

	return nil
}

func (d defaultGroupModifier) removeGroupFromUser(userId userId, groupId groupId) error {
	log.Debugf("Remove group with id '%v' from user with id '%v'", groupId, userId)

	removeUserFromGroupURL := fmt.Sprintf(d.RemoveUserFromGroupEndpoint, groupId, userId)

	resp, err := d.requester.Send(http.MethodDelete, removeUserFromGroupURL)
	if err != nil {
		return fmt.Errorf("failed to do delete request to remove user from group endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not remove user with id %d from group %d: %w", userId, groupId, err)
	}

	return nil
}

func (d defaultGroupModifier) getUserGroups(userId userId) (userGroupResponse, error) {
	log.Debugf("Get groups of user with id '%v'", userId)

	getUserGroupsURL := fmt.Sprintf(d.GetUserGroupEndpoint, userId)
	resp, err := d.requester.Send(http.MethodGet, getUserGroupsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to do get request to groups endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got response, but status code is not ok: %d", resp.StatusCode)
	}

	var response userGroupResponse
	err = json.Unmarshal(resp.ByteBody, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal groups response body: %w", err)
	}

	return response, nil
}
