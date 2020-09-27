package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/filebrowser/filebrowser/v2/errors"
	"github.com/filebrowser/filebrowser/v2/files"
	"github.com/filebrowser/filebrowser/v2/fileutils"

	libErrors "github.com/filebrowser/filebrowser/v2/errors"
)

var resourceGetHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	file, err := files.NewFileInfo(files.FileOptions{
		Fs:      d.user.Fs,
		Path:    r.URL.Path,
		Modify:  d.user.Perm.Modify,
		Expand:  true,
		Checker: d,
	})
	if err != nil {
		return errToStatus(err), err
	}

	if file.IsDir {
		file.Listing.Sorting = d.user.Sorting
		file.Listing.ApplySort()
		return renderJSON(w, r, file)
	}

	if checksum := r.URL.Query().Get("checksum"); checksum != "" {
		err := file.Checksum(checksum)
		if err == errors.ErrInvalidOption {
			return http.StatusBadRequest, nil
		} else if err != nil {
			return http.StatusInternalServerError, err
		}

		// do not waste bandwidth if we just want the checksum
		file.Content = ""
	}

	return renderJSON(w, r, file)
})

// func removeFilesInCache(uuid string, absdir string) error {
//     dir, err := ioutil.ReadDir(absdir)
//     if err != nil {
//         return err
//     }
//     for _, item := range dir {
//         if item.IsDir() { // dfs
//             removeFilesInCache(uuid, filepath.Join(absdir, item.Name()))
//         } else {
//             full := filepath.Join(absdir, item.Name())
//             cache.RemoveConfig(uuid, absdir, full, ConfigDB)
//             cache.RemoveConfig(uuid, absdir, full, ConfigXML)
//             cache.RemoveConfig(uuid, absdir, full, ConfigSVR)
//         }
//     }
//     return nil
// }

func resourceDeleteHandler(fileCache FileCache) handleFunc {
	return withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
		// filter requests without uuid or with inconsistent uuid
		// uuid := r.URL.Query().Get("uuid")
		// if uuid == "" {
		//     return http.StatusForbidden, nil
		// }

		// mtx.Lock()
		// if found, _ := cache.Get(uuid); !found {
		//     mtx.Unlock()
		//     return http.StatusForbidden, nil
		// }
		// mtx.Unlock()

		if r.URL.Path == "/" || !d.user.Perm.Delete {
			return http.StatusForbidden, nil
		}

		file, err := files.NewFileInfo(files.FileOptions{
			Fs:      d.user.Fs,
			Path:    r.URL.Path,
			Modify:  d.user.Perm.Modify,
			Expand:  true,
			Checker: d,
		})
		if err != nil {
			return errToStatus(err), err
		}

		// delete thumbnails
		for _, previewSizeName := range PreviewSizeNames() {
			size, _ := ParsePreviewSize(previewSizeName)
			if err := fileCache.Delete(r.Context(), previewCacheKey(file.Path, size)); err != nil { //nolint:govet
				return errToStatus(err), err
			}
		}

		err = d.RunHook(func() error {
			// full := filepath.Join(d.user.Scope, r.URL.Path)
			// mtx.Lock()
			// if strings.HasSuffix(r.URL.Path, "/") { // dir
			//     err := removeFilesInCache(uuid, full)
			//     if err != nil {
			//         mtx.Unlock()
			//         return err
			//     }
			// } else { // file
			//     dir := filepath.Dir(full)
			//     cache.RemoveConfig(uuid, dir, full, ConfigDB)
			//     cache.RemoveConfig(uuid, dir, full, ConfigXML)
			//     cache.RemoveConfig(uuid, dir, full, ConfigSVR)
			// }
			// mtx.Unlock()
			return d.user.Fs.RemoveAll(r.URL.Path)
		}, "delete", r.URL.Path, "", d.user)

		if err != nil {
			return errToStatus(err), err
		}

		return http.StatusOK, nil
	})
}

var resourcePostPutHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	uuid := r.URL.Query().Get("uuid")
	if uuid == "" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Operation is prohibited without uuid\n"))
		return errToStatus(libErrors.ErrPermissionDenied), libErrors.ErrPermissionDenied
	}

	dir := r.URL.Query().Get("dir")
	if dir == "" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Operation is prohibited without upload directory\n"))
		return errToStatus(libErrors.ErrPermissionDenied), libErrors.ErrPermissionDenied
	}
	absdir := filepath.Join(d.user.Scope, dir)

	mtx.Lock()
	if cache.Size() == 0 {
		vals := make(map[string]*CacheData)
		// upload all files within one hour
		// only one key will be successfully set
		if ok := cache.Set(uuid, vals, duration); !ok {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Cache uuid failed, please try again later!\n"))
			mtx.Unlock()
			return errToStatus(libErrors.ErrPermissionDenied), libErrors.ErrPermissionDenied
		}
	}

	// not allowed if uuid not exists
	if found := cache.IsKeyExisted(uuid); !found {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Someone is currently hot reloading, please try again later!\n"))
		mtx.Unlock()
		return errToStatus(libErrors.ErrPermissionDenied), libErrors.ErrPermissionDenied
	}
	mtx.Unlock()

	if !d.user.Perm.Create && r.Method == http.MethodPost {
		return http.StatusForbidden, nil
	}

	if !d.user.Perm.Modify && r.Method == http.MethodPut {
		return http.StatusForbidden, nil
	}

	defer func() {
		_, _ = io.Copy(ioutil.Discard, r.Body)
	}()

	// For directories, only allow POST for creation.
	if strings.HasSuffix(r.URL.Path, "/") {
		if r.Method == http.MethodPut {
			return http.StatusMethodNotAllowed, nil
		}

		err := d.user.Fs.MkdirAll(r.URL.Path, 0775)
		return errToStatus(err), err
	}

	if r.Method == http.MethodPost && r.URL.Query().Get("override") != "true" {
		if _, err := d.user.Fs.Stat(r.URL.Path); err == nil {
			return http.StatusConflict, nil
		}
	}

	action := "upload"
	if r.Method == http.MethodPut {
		action = "save"
	}

	err := d.RunHook(func() error {
		name := strings.ReplaceAll(r.URL.Path, dir, "")
		name = strings.TrimLeft(name, "/")

		err := d.user.Fs.MkdirAll(dir, 0775)
		if err != nil {
			return err
		}

		mtx.Lock()
		if found := cache.IsDirCacheExisted(uuid, absdir); !found {
			cache.SetCacheData(uuid, absdir, newCacheData())
		}
		mtx.Unlock()

		// If file exists, need backup
		if _, err := d.user.Fs.Stat(r.URL.Path); err == nil {
			// Note(youngerli): backup directory with uuid
			// uuid->dirname->{bak: dirname_uuid_timestamp, files: {xml:Set, db:Set, svr:Set} }
			mtx.Lock()
			bakdir := cache.GetBakDir(uuid, absdir)
			if bakdir == "" {
				timestamp := time.Unix(time.Now().Unix(), 0).Format("20060102_150405")
				arr := []string{dir, uuid, timestamp}
				bakdir = strings.Join(arr, "_")
				cache.SetBakDir(uuid, absdir, bakdir)
			}
			// Lock to ensure that the folder has been created
			src := r.URL.Path
			dst := filepath.Join(bakdir, name)
			err = d.user.Fs.MkdirAll(filepath.Dir(dst), 0775)
			if err != nil {
				mtx.Unlock()
				return err
			}
			mtx.Unlock()
			err = d.user.Fs.Rename(src, dst)
			if err != nil {
				return err
			}
		}

		file, err := d.user.Fs.OpenFile(r.URL.Path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0775)

		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, r.Body)
		if err != nil {
			return err
		}

		// Gets the info about the file.
		info, err := file.Stat()
		if err != nil {
			return err
		}

		etag := fmt.Sprintf(`"%x%x"`, info.ModTime().UnixNano(), info.Size())
		w.Header().Set("ETag", etag)

		return nil
	}, action, r.URL.Path, "", d.user)

	if err != nil {
		_ = d.user.Fs.RemoveAll(r.URL.Path)
	} else { // cache without error
		// Note(youngerli): Except for the ClientConfig and ServerConfig,
		// other files or directories are not regarded as configuration so they will not cached
		// reload strategy: ServerConfig is fully reloaded,
		// the db in ClientConfig is reloaded according to SvrLoadList.xml,
		// and the xml is fully reloaded
		full := filepath.Join(d.user.Scope, r.URL.Path)
		mtx.Lock()
		if strings.Contains(dir, "ClientConfig") {
			if strings.HasSuffix(full, ".xml") {
				err = cache.AddConfig(uuid, absdir, full, ConfigXML)
			} else if strings.HasSuffix(full, ".db") {
				err = cache.AddConfig(uuid, absdir, full, ConfigDB)
			}
		} else if strings.Contains(dir, "ServerConfig") {
			err = cache.AddConfig(uuid, absdir, full, ConfigSVR)
		}
		mtx.Unlock()
	}

	return errToStatus(err), err
})

var resourcePatchHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	src := r.URL.Path
	dst := r.URL.Query().Get("destination")
	action := r.URL.Query().Get("action")
	dst, err := url.QueryUnescape(dst)
	if err != nil {
		return errToStatus(err), err
	}
	if dst == "/" || src == "/" {
		return http.StatusForbidden, nil
	}
	if err = checkParent(src, dst); err != nil {
		return http.StatusBadRequest, err
	}

	override := r.URL.Query().Get("override") == "true"
	rename := r.URL.Query().Get("rename") == "true"
	if !override && !rename {
		if _, err = d.user.Fs.Stat(dst); err == nil {
			return http.StatusConflict, nil
		}
	}
	if rename {
		dst = addVersionSuffix(dst, d.user.Fs)
	}

	err = d.RunHook(func() error {
		switch action {
		// TODO: use enum
		case "copy":
			if !d.user.Perm.Create {
				return errors.ErrPermissionDenied
			}

			return fileutils.Copy(d.user.Fs, src, dst)
		case "rename":
			if !d.user.Perm.Rename {
				return errors.ErrPermissionDenied
			}
			dst = filepath.Clean("/" + dst)

			return d.user.Fs.Rename(src, dst)
		default:
			return fmt.Errorf("unsupported action %s: %w", action, errors.ErrInvalidRequestParams)
		}
	}, action, src, dst, d.user)

	return errToStatus(err), err
})

func checkParent(src string, dst string) error {
	rel, err := filepath.Rel(src, dst)
	if err != nil {
		return err
	}

	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, "../") && rel != ".." && rel != "." {
		return errors.ErrSourceIsParent
	}

	return nil
}

func addVersionSuffix(path string, fs afero.Fs) string {
	counter := 1
	dir, name := filepath.Split(path)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	for {
		if _, err := fs.Stat(path); err != nil {
			break
		}
		renamed := fmt.Sprintf("%s(%d)%s", base, counter, ext)
		path = filepath.ToSlash(dir) + renamed
		counter++
	}

	return path
}
