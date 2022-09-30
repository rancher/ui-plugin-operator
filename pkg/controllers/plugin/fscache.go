package plugin

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/rancher/ui-plugin-operator/pkg/apis/catalog.cattle.io/v1"
	"github.com/sirupsen/logrus"
)

const (
	FSCacheRootDir = "/home/uipluginoperator/cache"

	// Cache states used by custom resources
	Cached   = "cached"
	Disabled = "disabled"
	Pending  = "pending"
)

var (
	FsCache     = FSCache{}
	osRemoveAll = os.RemoveAll
	osStat      = os.Stat
	isDirEmpty  = isDirectoryEmpty
)

type FSCache struct{}

// SyncWithControllersCache takes in a slice of UI Plugins objects and syncs the filesystem cache with it
func (c FSCache) SyncWithControllersCache(cachedPlugins []*v1.UIPlugin) error {
	for _, p := range cachedPlugins {
		plugin := p.Spec.Plugin
		if plugin.NoCache {
			logrus.Debugf("skipped caching plugin [Name: %s Version: %s] cache is disabled [noCache: %v]", plugin.Name, plugin.Version, plugin.NoCache)
			continue
		}
		if isCached, err := c.isCached(plugin.Name, plugin.Version); err != nil {
			return err
		} else if isCached {
			logrus.Debugf("skipped caching plugin [Name: %s Version: %s] is already cached", plugin.Name, plugin.Version)
			continue
		}
		err := c.Save(plugin.Name, plugin.Version)
		if err != nil {
			return err
		}
		urlFilesTxt := fmt.Sprintf("%s/%s", plugin.Endpoint, FilesTxtFilename)
		files, err := getPluginFiles(urlFilesTxt)
		if err != nil {
			return err
		}
		for _, file := range files {
			if file == "" {
				continue
			}
			err = fetchFile(plugin.Endpoint, plugin.Name, plugin.Version, file)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// SyncWithIndex syncs up entries in the filesystem cache with the index's entries
// Entries that aren't in the index, but present in the filesystem cache are deleted
func (c FSCache) SyncWithIndex(index *SafeIndex, fsCacheFiles []string) error {
	for _, file := range fsCacheFiles {
		logrus.Debugf("syncing index with filesystem cache")
		// Splits /{root}/{pluginName}/{pluginVersion}/* from a fs cache path
		rel, _ := filepath.Rel(FSCacheRootDir, file)
		s := strings.Split(rel, "/")
		name := s[0]
		version := s[1]
		_, ok := index.Entries[name]
		if ok && index.Entries[name].Version == version {
			continue
		} else {
			err := c.Delete(name, version)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c FSCache) Delete(name, version string) error {
	// Delete plugin entry from filesystem cache
	err := osRemoveAll(filepath.Join(FSCacheRootDir, name))
	if err != nil {
		err = fmt.Errorf("failed to delete entry [Name: %s Version: %s] from filesystem cache: %s", name, version, err.Error())
		return err
	}
	logrus.Debugf("deleted plugin entry from cache [Name: %s Version: %s]", name, version)

	return nil
}

// Save takes in the name and version of a plugin and creates an entry for it in the filesystem cache
func (c FSCache) Save(name, version string) error {
	err := os.MkdirAll(filepath.Join(FSCacheRootDir, name, version), os.ModePerm)
	if err != nil {
		logrus.Debugf("failed to cache plugin [Name: %s Version: %s] in filesystem", name, version)
		return err
	}

	return nil
}

// isCache takes in the name and version of a plugin and returns true if
// it is cached (entry exists and files were fetched), returns false otherwise
func (c FSCache) isCached(name, version string) (bool, error) {
	path := filepath.Join(FSCacheRootDir, name, version)
	_, err := osStat(path)
	if !errors.Is(err, os.ErrNotExist) {
		isEmpty, err := isDirEmpty(path)
		if err != nil {
			return false, err
		} else if !isEmpty {
			return true, nil
		}
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return false, nil
}

func fsCacheFilepathGlob(pattern string) ([]string, error) {
	files, err := filepath.Glob(pattern)
	logrus.Debugf("files matching glob pattern [%s] found in filesystem cache: %+v", pattern, files)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// getPluginFiles takes in a URL for a plugin's files.txt, reads it, and returns a slice of the file paths contained in files.txt
func getPluginFiles(urlFilesTxt string) ([]string, error) {
	var files []string
	logrus.Debugf("fetching file [%s]", urlFilesTxt)
	resp, err := http.Get(urlFilesTxt)
	if err != nil {
		return files, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return files, err
	}
	files = strings.Split(string(b), "\n")

	return files, nil
}

func fetchFile(endpoint, name, version, file string) error {
	url := endpoint + "/" + file
	logrus.Debugf("fetching file [%s]", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	path := filepath.Join(FSCacheRootDir, name, version, file)
	logrus.Debugf("creating file [%s]", url)
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	io.Copy(out, resp.Body)

	return nil
}

func isDirectoryEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}
