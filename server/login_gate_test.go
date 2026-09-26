package main

import (
	"testing"

	"github.com/robertkonga/yekonga-server-go/datatype"
)

func TestLoginAllowed(t *testing.T) {
	users := []datatype.DataMap{
		{"username": "255711000002", "status": "active"},
		{"username": "255711000009", "status": "inactive"},
	}
	groups := []datatype.DataMap{
		{"adminPhone": "0711000005", "status": "Active"},
		{"adminPhone": "0711000008", "status": "Closed"},
	}
	cases := []struct {
		name, username, super string
		users                 []datatype.DataMap
		want                  error
	}{
		{"existing account", "255711000002", "", users, nil},
		{"existing account, other format", "+255 711 000 002", "", users, nil},
		{"deactivated account", "255711000009", "", users, errDeactivated},
		{"stranger", "255799999999", "", users, errNotRegistered},
		{"group admin phone, first login", "255711000005", "", users, nil},
		{"admin of a closed group", "255711000008", "", users, errNotRegistered},
		{"super admin phone", "255799999999", "0799999999,0788", users, nil},
		{"first setup: no users yet", "255799999999", "", nil, nil},
		{"email stranger", "someone@example.com", "", users, errNotRegistered},
	}
	for _, c := range cases {
		if got := loginAllowed(c.users, groups, c.username, c.super); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
