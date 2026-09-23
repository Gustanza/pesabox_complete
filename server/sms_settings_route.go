package main

import (
	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// registerSmsSettings wires GET /api/admin/sms/settings: the persisted SMS
// switches and language the web Settings screen shows. (Saving them is
// POST /api/admin/settings in registerSmsAdmin.)
func registerSmsSettings(app *yekonga.YekongaData) {
	app.Get("/api/admin/sms/settings", func(req *yekonga.Request, res *yekonga.Response) {
		if req.Auth() == nil {
			res.Status(401)
			res.Json(map[string]string{"error": "unauthorized"})
			return
		}
		res.Json(notificationSettings())
	})
}
