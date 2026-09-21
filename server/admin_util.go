package main

import (
	"encoding/json"
	"io"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// bodyMap returns the JSON request body as a DataMap. The framework sometimes
// fails to populate req.Body() for REST routes (RawBody stays nil), so this
// falls back to reading the underlying http request body directly.
func bodyMap(req *yekonga.Request) datatype.DataMap {
	body := helper.ToDataMap(req.Body())
	if len(body) > 0 {
		return body
	}
	if req.HttpRequest.Body == nil {
		return body
	}
	raw, err := io.ReadAll(req.HttpRequest.Body)
	if err != nil {
		return body
	}
	var m map[string]interface{}
	if json.Unmarshal(raw, &m) == nil {
		return datatype.DataMap(m)
	}
	return body
}

// sessionRole looks up the real "role" field on the User record for userId.
// AuthPayload/the JWT never carries a role, so any check on it has to go back
// to the User record itself.
func sessionRole(app *yekonga.YekongaData, userId string) string {
	if userId == "" {
		return ""
	}
	rec := app.ModelQuery("User").SkipBeforeCommit().Where("id", userId).First(nil)
	if rec == nil {
		return ""
	}
	return helper.GetValueOfString(*rec, "role")
}

// isPlatformAdmin reports whether a User.role may see every group and change
// platform-wide settings: the dashboard's "super_admin" or the framework's
// built-in "admin".
func isPlatformAdmin(role string) bool { return role == "super_admin" || role == "admin" }
