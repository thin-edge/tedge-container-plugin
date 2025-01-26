package c8y

import (
	"context"
	"fmt"

	"github.com/tidwall/gjson"
)

// UserService provides the service provider for the Cumulocity Application API
type UserService service

// UserOptions options that can be provided when using user api calls
type UserOptions struct {
	// Prefix or full username
	Username string `url:"username,omitempty"`

	Groups []string `url:"groups,omitempty"`

	// Exact username
	Owner string `url:"owner,omitempty"`

	// OnlyDevices If set to "true", result will contain only users created during bootstrap process (starting with "device_"). If flag is absent (or false) the result will not contain "device_" users.
	OnlyDevices bool `url:"onlyDevices,omitempty"`

	// WithSubusersCount if set to "true", then each of returned users will contain additional field "subusersCount" - number of direct subusers (users with corresponding "owner").
	WithSubusersCount bool `url:"withSubusersCount,omitempty"`

	PaginationOptions
}

// User todo
type User struct {
	ID                string                    `json:"id,omitempty"`
	Self              string                    `json:"self,omitempty"`
	Username          string                    `json:"userName,omitempty"`
	Password          string                    `json:"password,omitempty"`
	FirstName         string                    `json:"firstName,omitempty"`
	LastName          string                    `json:"lastName,omitempty"`
	Phone             string                    `json:"phone,omitempty"`
	Email             string                    `json:"email,omitempty"`
	Enabled           bool                      `json:"enabled,omitempty"`
	CustomProperties  interface{}               `json:"customProperties,omitempty"`
	Groups            *GroupReferenceCollection `json:"groups,omitempty"`
	Roles             *RoleReferenceCollection  `json:"roles,omitempty"`
	DevicePermissions map[string]interface{}    `json:"devicePermissions,omitempty"`
	EffectiveRoles    []Role                    `json:"effectiveRoles,omitempty"`

	Item gjson.Result `json:"-"`
}

func (u *User) SetFirstName(value string) *User {
	u.FirstName = value
	return u
}

func (u *User) SetLastName(value string) *User {
	u.LastName = value
	return u
}

func (u *User) SetEmail(value string) *User {
	u.Email = value
	return u
}

func (u *User) SetPhone(value string) *User {
	u.Phone = value
	return u
}

func (u *User) SetEnabled(value bool) *User {
	u.Enabled = value
	return u
}

// NewUser returns a new user object
func NewUser(username string, email string, password string) *User {
	return &User{
		Username: username,
		Email:    email,
		Enabled:  true,
		Password: password,
	}
}

// GroupReference represents group information
type GroupReference struct {
	Self  string `json:"self,omitempty"`
	Group *Group `json:"group,omitempty"`
}

type RoleReferenceCollection struct {
	Self       string          `json:"self,omitempty"`
	References []RoleReference `json:"references,omitempty"`
}

type RoleReference struct {
	Self string `json:"self,omitempty"`
	Role *Role  `json:"role,omitempty"`
}

type Role struct {
	Self string `json:"self,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Group struct {
	ID                uint64                   `json:"id,omitempty"`
	Self              string                   `json:"self,omitempty"`
	Name              string                   `json:"name,omitempty"`
	Roles             *RoleReferenceCollection `json:"roles,omitempty"`
	DevicePermissions map[string]interface{}   `json:"devicePermissions,omitempty"`
}

// GetID returns the group id as a string
func (g *Group) GetID() string {
	if g.ID == 0 {
		return ""
	}
	return fmt.Sprintf("%d", g.ID)
}

// UserCollection contains information about a list of users
type UserCollection struct {
	*BaseResponse

	Users []User `json:"users"`

	Items []gjson.Result `json:"-"`
}

// GetUsers returns a list of users for the given tenant
// Users in the response are sorted by username in ascending order.
func (s *UserService) GetUsers(ctx context.Context, opt *UserOptions) (*UserCollection, *Response, error) {
	data := new(UserCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/" + s.client.TenantName + "/users",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetUser returns a user by its ID
func (s *UserService) GetUser(ctx context.Context, ID string) (*User, *Response, error) {
	data := new(User)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/" + s.client.TenantName + "/users/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// GetUserByUsername returns a user by their username
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*User, *Response, error) {
	data := new(User)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/" + s.client.TenantName + "/userByName/" + username,
		ResponseData: data,
	})
	return data, resp, err
}

// Create adds a new user
func (s *UserService) Create(ctx context.Context, body *User) (*User, *Response, error) {
	data := new(User)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "user/" + s.client.TenantName + "/users",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates an existing user
func (s *UserService) Update(ctx context.Context, ID string, body *User) (*User, *Response, error) {
	data := new(User)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "user/" + s.client.TenantName + "/users/" + ID,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete removes an existing user
func (s *UserService) Delete(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "user/" + s.client.TenantName + "/users/" + ID,
	})
}

// GetCurrentUser returns the current user based on the request's credentials
func (s *UserService) GetCurrentUser(ctx context.Context) (*User, *Response, error) {
	data := new(User)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/currentUser",
		ResponseData: data,
	})
	return data, resp, err
}

// UpdateCurrentUser updates the current user based on the request's credentials
func (s *UserService) UpdateCurrentUser(ctx context.Context, body *User) (*User, *Response, error) {
	data := new(User)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "user/currentUser",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

type UserReferenceCollection struct {
	*BaseResponse

	Self       string          `json:"self,omitempty"`
	References []UserReference `json:"references,omitempty"`
}

type UserReference struct {
	Self string `json:"self,omitempty"`
	User *User  `json:"user,omitempty"`
}

type GroupReferenceCollection struct {
	*BaseResponse

	Self       string           `json:"self,omitempty"`
	References []GroupReference `json:"references,omitempty"`
}

// AddUserToGroup adds the user to an existing group
func (s *UserService) AddUserToGroup(ctx context.Context, user *User, groupID string) (*UserReference, *Response, error) {
	data := new(UserReference)

	body := &UserReference{
		User: &User{
			Self: user.Self,
		},
	}

	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "user/" + s.client.TenantName + "/groups/" + groupID + "/users",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// RemoveUserFromGroup removes a user from a group
func (s *UserService) RemoveUserFromGroup(ctx context.Context, username string, groupID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "user/" + s.client.TenantName + "/groups/" + groupID + "/users/" + username,
	})
}

// GetUsersByGroup returns the list of users in the given group
func (s *UserService) GetUsersByGroup(ctx context.Context, groupID string, opt *UserOptions) (*UserReferenceCollection, *Response, error) {
	data := new(UserReferenceCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/" + s.client.TenantName + "/groups/" + groupID + "/users",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

type GroupCollection struct {
	*BaseResponse

	Self   string  `json:"self,omitempty"`
	Groups []Group `json:"groups,omitempty"`

	Items []gjson.Result `json:"-"`
}

// GroupOptions available options when querying a list of groups
type GroupOptions struct {
	PaginationOptions
}

// GetGroups returns the list of user groups
func (s *UserService) GetGroups(ctx context.Context, opt *GroupOptions) (*GroupCollection, *Response, error) {
	data := new(GroupCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/" + s.client.TenantName + "/groups",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// CreateGroup creates a new group with the given name
func (s *UserService) CreateGroup(ctx context.Context, body *Group) (*Group, *Response, error) {
	data := new(Group)

	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "user/" + s.client.TenantName + "/groups",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// GetGroup returns a group by its id
func (s *UserService) GetGroup(ctx context.Context, ID string) (*Group, *Response, error) {
	data := new(Group)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/" + s.client.TenantName + "/groups/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// GetGroupByName returns the group by its name
func (s *UserService) GetGroupByName(ctx context.Context, name string) (*Group, *Response, error) {
	data := new(Group)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/" + s.client.TenantName + "/groupByName/" + name,
		ResponseData: data,
	})
	return data, resp, err
}

// DeleteGroup deletes an existing group
// Info: ADMINS and DEVICES groups can not be deleted
func (s *UserService) DeleteGroup(ctx context.Context, ID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "user/" + s.client.TenantName + "/groups/" + ID,
	})
}

// UpdateGroup updates properties of an existing group
func (s *UserService) UpdateGroup(ctx context.Context, ID string, body *Group) (*Group, *Response, error) {
	data := new(Group)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "user/" + s.client.TenantName + "/groups/" + ID,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// GetGroupsByUser returns list of groups assigned to a given user
func (s *UserService) GetGroupsByUser(ctx context.Context, username string, opt *GroupOptions) (*GroupReferenceCollection, *Response, error) {
	data := new(GroupReferenceCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/" + s.client.TenantName + "/users/" + username + "/groups",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

/* ROLES */

// RoleCollection is a list of user roles in the platform
type RoleCollection struct {
	*BaseResponse

	Self  string `json:"self,omitempty"`
	Roles []Role `json:"roles,omitempty"`

	Items []gjson.Result `json:"-"`
}

// RoleOptions options to be used when querying for roles
type RoleOptions struct {
	PaginationOptions
}

// GetRoles returns a list of existing roles
func (s *UserService) GetRoles(ctx context.Context, opt *RoleOptions) (*RoleCollection, *Response, error) {
	data := new(RoleCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/roles",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetRole returns the role of the given id
func (s *UserService) GetRole(ctx context.Context, ID string) (*Role, *Response, error) {
	data := new(Role)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/roles/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

func (s *UserService) AssignRoleToUser(ctx context.Context, username string, roleSelfReference string) (*RoleReference, *Response, error) {
	data := new(RoleReference)

	body := &RoleReference{
		Role: &Role{
			Self: roleSelfReference,
		},
	}
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "user/" + s.client.TenantName + "/users/" + username + "/roles",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// UnassignRoleFromUser removes a role from an existing user
func (s *UserService) UnassignRoleFromUser(ctx context.Context, username string, roleName string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "user/" + s.client.TenantName + "/users/" + username + "/roles/" + roleName,
	})
}

// GetRolesByUser returns list of roles of an existing user
func (s *UserService) GetRolesByUser(ctx context.Context, username string, opt *RoleOptions) (*RoleReferenceCollection, *Response, error) {
	data := new(RoleReferenceCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/" + s.client.TenantName + "/users/" + username + "/roles",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// AssignRoleToGroup adds a role to an existing group
func (s *UserService) AssignRoleToGroup(ctx context.Context, groupID string, roleSelfReference string) (*RoleReference, *Response, error) {
	data := new(RoleReference)
	body := &RoleReference{
		Role: &Role{
			Self: roleSelfReference,
		},
	}
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "user/" + s.client.TenantName + "/groups/" + groupID + "/roles",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// UnassignRoleFromGroup removes a role from an existing user
func (s *UserService) UnassignRoleFromGroup(ctx context.Context, groupID string, roleName string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "user/" + s.client.TenantName + "/groups/" + groupID + "/roles/" + roleName,
	})
}

// GetRolesByGroup returns list of roles of an existing group
func (s *UserService) GetRolesByGroup(ctx context.Context, groupID string, opt *RoleOptions) (*RoleReferenceCollection, *Response, error) {
	data := new(RoleReferenceCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "user/" + s.client.TenantName + "/groups/" + groupID + "/roles",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}
