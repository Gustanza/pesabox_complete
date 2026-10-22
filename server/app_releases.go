package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robertkonga/yekonga-server-go/yekonga"
)

// Mobile app releases: the super admin uploads the Android APK from the web
// dashboard, the landing page offers the latest one for download.
//
//	GET    /api/admin/app-releases       every release, newest first   (PermPlatform)
//	POST   /api/admin/app-releases       multipart: file (.apk), version, notes
//	DELETE /api/admin/app-releases/:id   remove a release and its file
//	GET    /api/public/app               latest release (public; {available:false} when none)
//	GET    /api/public/app/download      the latest APK (public)
//
// Files and their metadata (releases.json) live in APP_RELEASES_DIR, default
// ./app-releases next to the binary. Not stored in the database: the files are
// the source of truth and GraphQL never sees them.

const (
	maxApkBytes   = 300 << 20 // the framework caps request bodies at ~310 MB
	apkMimeType   = "application/vnd.android.package-archive"
	releasesIndex = "releases.json"
)

type appRelease struct {
	ID           string    `json:"id"`
	Version      string    `json:"version"`
	Notes        string    `json:"notes"`
	FileName     string    `json:"fileName"`     // stored name on disk
	OriginalName string    `json:"originalName"` // what the admin uploaded
	Size         int64     `json:"size"`
	SHA256       string    `json:"sha256"`
	UploadedAt   time.Time `json:"uploadedAt"`
	UploadedBy   string    `json:"uploadedBy"`
	Downloads    int       `json:"downloads"`
}

var (
	releasesMu   sync.Mutex
	versionRegex = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z.+_ -]{0,39}$`)
)

func releasesDir() string {
	if d := strings.TrimSpace(os.Getenv("APP_RELEASES_DIR")); d != "" {
		return d
	}
	return "./app-releases"
}

// loadReleases reads the index, newest first. Call with releasesMu held.
func loadReleases() []appRelease {
	raw, err := os.ReadFile(filepath.Join(releasesDir(), releasesIndex))
	if err != nil {
		return []appRelease{}
	}
	var list []appRelease
	if json.Unmarshal(raw, &list) != nil {
		return []appRelease{}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].UploadedAt.After(list[j].UploadedAt) })
	return list
}

// saveReleases writes the index atomically. Call with releasesMu held.
func saveReleases(list []appRelease) error {
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(releasesDir(), releasesIndex+".tmp")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(releasesDir(), releasesIndex))
}

// publicRelease is what anyone may see: no uploader, no stored file name.
func publicRelease(r appRelease) map[string]interface{} {
	return map[string]interface{}{
		"available":   true,
		"version":     r.Version,
		"notes":       r.Notes,
		"size":        r.Size,
		"sha256":      r.SHA256,
		"releasedAt":  r.UploadedAt,
		"downloadUrl": "/api/public/app/download",
	}
}

func registerAppReleases(app *yekonga.YekongaData) {
	if err := os.MkdirAll(releasesDir(), 0o755); err != nil {
		fmt.Println("app releases: cannot create", releasesDir(), err)
	}
	// Public even for a visitor carrying an expired session cookie.
	app.SetPublicRoute("/api/public/*")
	app.SetPublicRoute("/api/public/*/*")

	app.Get("/api/admin/app-releases", func(req *yekonga.Request, res *yekonga.Response) {
		if requirePerm(app, req, res, PermPlatform) == nil {
			return
		}
		releasesMu.Lock()
		list := loadReleases()
		releasesMu.Unlock()
		res.Json(map[string]interface{}{"releases": list})
	})

	app.Post("/api/admin/app-releases", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		r := req.HttpRequest
		version := strings.TrimSpace(r.FormValue("version"))
		notes := strings.TrimSpace(r.FormValue("notes"))
		if !versionRegex.MatchString(version) {
			deny(res, 400, "version is required (e.g. 1.2.0), up to 40 letters, numbers, dots or dashes")
			return
		}
		if len(notes) > 2000 {
			deny(res, 400, "release notes are limited to 2000 characters")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			deny(res, 400, "choose the APK file to upload (the upload may also be larger than the server allows)")
			return
		}
		defer file.Close()
		if !strings.EqualFold(filepath.Ext(header.Filename), ".apk") {
			deny(res, 400, "only Android .apk files can be uploaded")
			return
		}
		if header.Size > maxApkBytes {
			deny(res, 400, "the file is larger than 300 MB")
			return
		}
		// An APK is a zip archive: check the signature before keeping it.
		magic := make([]byte, 4)
		if _, err := io.ReadFull(file, magic); err != nil || !bytes.Equal(magic, []byte("PK\x03\x04")) {
			deny(res, 400, "this file is not a valid APK")
			return
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			deny(res, 500, "could not read the upload")
			return
		}

		id := time.Now().UTC().Format("20060102T150405.000")
		id = strings.ReplaceAll(id, ".", "")
		safeVersion := regexp.MustCompile(`[^0-9A-Za-z.+_-]`).ReplaceAllString(version, "-")
		stored := fmt.Sprintf("helabox-%s-%s.apk", safeVersion, id)
		dst := filepath.Join(releasesDir(), stored)
		tmp := dst + ".part"
		out, err := os.Create(tmp)
		if err != nil {
			deny(res, 500, "could not save the file on the server")
			return
		}
		hash := sha256.New()
		size, err := io.Copy(io.MultiWriter(out, hash), io.LimitReader(file, maxApkBytes+1))
		out.Close()
		if err != nil || size > maxApkBytes {
			os.Remove(tmp)
			deny(res, 400, "the upload failed or is larger than 300 MB")
			return
		}
		if err := os.Rename(tmp, dst); err != nil {
			os.Remove(tmp)
			deny(res, 500, "could not save the file on the server")
			return
		}

		rel := appRelease{
			ID: id, Version: version, Notes: notes, FileName: stored, OriginalName: filepath.Base(header.Filename),
			Size: size, SHA256: hex.EncodeToString(hash.Sum(nil)), UploadedAt: time.Now().UTC(), UploadedBy: a.Name,
		}
		releasesMu.Lock()
		list := append([]appRelease{rel}, loadReleases()...)
		err = saveReleases(list)
		releasesMu.Unlock()
		if err != nil {
			os.Remove(dst)
			deny(res, 500, "could not record the release")
			return
		}
		writeAudit(app, a, "create", "AppRelease", rel.ID, "", fmt.Sprintf("Uploaded app version %s (%s)", rel.Version, humanBytes(rel.Size)),
			map[string]interface{}{"version": rel.Version, "size": rel.Size, "sha256": rel.SHA256})
		res.Json(map[string]interface{}{"release": rel})
	})

	app.Delete("/api/admin/app-releases/:id", func(req *yekonga.Request, res *yekonga.Response) {
		a := requirePerm(app, req, res, PermPlatform)
		if a == nil {
			return
		}
		id := req.Param("id")
		releasesMu.Lock()
		list := loadReleases()
		var removed *appRelease
		kept := make([]appRelease, 0, len(list))
		for i := range list {
			if list[i].ID == id {
				removed = &list[i]
				continue
			}
			kept = append(kept, list[i])
		}
		var err error
		if removed != nil {
			err = saveReleases(kept)
		}
		releasesMu.Unlock()
		if removed == nil {
			deny(res, 404, "release not found")
			return
		}
		if err != nil {
			deny(res, 500, "could not update the release list")
			return
		}
		os.Remove(filepath.Join(releasesDir(), removed.FileName))
		writeAudit(app, a, "delete", "AppRelease", removed.ID, "", "Removed app version "+removed.Version, nil)
		res.Json(map[string]interface{}{"ok": true})
	})

	app.Get("/api/public/app", func(req *yekonga.Request, res *yekonga.Response) {
		releasesMu.Lock()
		list := loadReleases()
		releasesMu.Unlock()
		res.SetHeader("cache-control", "no-cache")
		if len(list) == 0 {
			res.Json(map[string]interface{}{"available": false})
			return
		}
		res.Json(publicRelease(list[0]))
	})

	app.Get("/api/public/app/download", func(req *yekonga.Request, res *yekonga.Response) {
		releasesMu.Lock()
		list := loadReleases()
		if len(list) > 0 && req.HttpRequest.Header.Get("Range") == "" {
			list[0].Downloads++ // count whole downloads, not resumed chunks
			saveReleases(list)
		}
		releasesMu.Unlock()
		if len(list) == 0 {
			deny(res, 404, "no app has been published yet")
			return
		}
		latest := list[0]
		path := filepath.Join(releasesDir(), latest.FileName)
		if _, err := os.Stat(path); err != nil {
			deny(res, 404, "the app file is missing on the server")
			return
		}
		name := fmt.Sprintf("HelaBox-%s.apk", regexp.MustCompile(`[^0-9A-Za-z.+_-]`).ReplaceAllString(latest.Version, "-"))
		res.SetHeader("Content-Type", apkMimeType)
		res.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
		res.SetHeader("X-Content-Type-Options", "nosniff")
		res.File(path)
	})
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}
