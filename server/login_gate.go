package main

import (
	"errors"
	"os"
	"strings"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// Login is invite-only: the framework creates a User on the first successful
// OTP login, so without this gate any phone number could sign itself up (an
// empty account plus a paid OTP SMS). The gate runs before the OTP is created,
// so a refused number gets no SMS, no UserVerification record and therefore
// can never complete a login.
//
// The clients look for these phrases to show a translated message — keep them
// stable ("not registered" / "deactivated").
var (
	errNotRegistered = errors.New("This phone number is not registered. Ask your group leader or administrator to add you.")
	errDeactivated   = errors.New("This account has been deactivated. Contact your administrator.")
)

// loginAllowed decides whether username may receive a login OTP:
//   - an existing, active account;
//   - the adminPhone of an open group (its Mwenyekiti's first login — the
//     account is created and linked to that group on login);
//   - a SUPER_ADMIN_PHONES number, or anyone while the platform has no users
//     yet (first setup).
func loginAllowed(users, groups []datatype.DataMap, username, superAdminPhones string) error {
	tail := last9(username)
	isPhone := tail != "" && len(tail) == 9
	for _, u := range users {
		match := helper.GetValueOfString(u, "username") == username
		if !match && isPhone {
			match = last9(helper.GetValueOfString(u, "username")) == tail || last9(helper.GetValueOfString(u, "phone")) == tail
		}
		if match {
			if helper.GetValueOfString(u, "status") == "inactive" {
				return errDeactivated
			}
			return nil
		}
	}
	if len(users) == 0 {
		return nil
	}
	if isPhone {
		for _, p := range strings.Split(superAdminPhones, ",") {
			if last9(p) == tail {
				return nil
			}
		}
		for _, g := range groups {
			if helper.GetValueOfString(g, "status") == "Closed" {
				continue
			}
			if last9(helper.GetValueOfString(g, "adminPhone")) == tail {
				return nil
			}
		}
	}
	return errNotRegistered
}

func registerLoginGate(app *yekonga.YekongaData) {
	app.BeforeOtp(func(req *yekonga.RequestContext, ctx *yekonga.QueryContext) (interface{}, error) {
		username := strings.TrimSpace(helper.GetValueOfString(helper.ToDataMap(ctx.Input), "username"))
		if username == "" {
			return errNotRegistered, nil
		}
		if helper.IsPhone(username) {
			username = helper.PhoneFormat(username)
		}
		users := listAll(app, "User")
		groups := listAll(app, "Group")
		if err := loginAllowed(users, groups, username, os.Getenv("SUPER_ADMIN_PHONES")); err != nil {
			return err, nil
		}
		return true, nil
	})
}
