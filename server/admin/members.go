package admin

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func (api *API) members(c echo.Context) error {
	userIDs, err := parseUintCSV(c.QueryParam("users"))
	if err != nil {
		return err
	}
	groupIDs, err := parseUintCSV(c.QueryParam("groups"))
	if err != nil {
		return err
	}
	members, err := api.searchMembers(c.QueryParam("q"), userIDs, groupIDs)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, members)
}

func (api *API) readMembers() (Members, error) {
	return api.searchMembers("", nil, nil)
}

func (api *API) searchMembers(q string, userIDs []uint, groupIDs []uint) (Members, error) {
	q = strings.TrimSpace(q)
	userIDs = cleanUintList(userIDs)
	groupIDs = cleanUintList(groupIDs)
	if q == "" && (len(userIDs) > 0 || len(groupIDs) > 0) {
		users, err := api.searchUsers("", nil, 50)
		if err != nil {
			return Members{}, err
		}
		selectedUsers, err := api.searchUsers("", userIDs, len(userIDs))
		if err != nil {
			return Members{}, err
		}
		groups, err := api.searchGroups("", nil, 50)
		if err != nil {
			return Members{}, err
		}
		selectedGroups, err := api.searchGroups("", groupIDs, len(groupIDs))
		if err != nil {
			return Members{}, err
		}
		return Members{Users: mergeUsers(selectedUsers, users), Groups: mergeGroups(selectedGroups, groups)}, nil
	}
	userLimit := 50 + len(userIDs)
	groupLimit := 50 + len(groupIDs)
	if q == "" && len(userIDs) == 0 && len(groupIDs) == 0 {
		userLimit = 200
		groupLimit = 200
	}
	users, err := api.searchUsers(q, userIDs, userLimit)
	if err != nil {
		return Members{}, err
	}
	groups, err := api.searchGroups(q, groupIDs, groupLimit)
	if err != nil {
		return Members{}, err
	}
	return Members{Users: users, Groups: groups}, nil
}

func mergeUsers(lists ...[]User) []User {
	seen := map[uint]bool{}
	items := []User{}
	for _, list := range lists {
		for _, item := range list {
			if seen[item.ID] {
				continue
			}
			seen[item.ID] = true
			items = append(items, item)
		}
	}
	return items
}

func mergeGroups(lists ...[]Group) []Group {
	seen := map[uint]bool{}
	items := []Group{}
	for _, list := range lists {
		for _, item := range list {
			if seen[item.ID] {
				continue
			}
			seen[item.ID] = true
			items = append(items, item)
		}
	}
	return items
}

func includeIDsOrZero(ids []uint) []uint {
	if len(ids) > 0 {
		return ids
	}
	return []uint{0}
}
