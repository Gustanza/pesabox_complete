package main

import "os"

// brandName is the product name users see — report headers, export file
// names, SMS text. The product was renamed PesaBox → HelaBox (TODO.md §3).
//
// It is deliberately NOT config.json's "appName": the framework derives its
// data directory (~/.yekonga-server/<app-name>) from appName, so renaming that
// would point the server at a new, empty folder. appName stays "PesaBox" as
// an internal id; set BRAND_NAME to change what users see without a rebuild.
var brandName = "HelaBox"

func init() {
	if v := os.Getenv("BRAND_NAME"); v != "" {
		brandName = v
	}
}
