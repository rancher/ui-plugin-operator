package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	v1 "github.com/rancher/ui-plugin-operator/pkg/apis/catalog.cattle.io/v1"
	plugincontroller "github.com/rancher/ui-plugin-operator/pkg/generated/controllers/catalog.cattle.io/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	Index = SafeIndex{}
)

type SafeIndex struct {
	mu      sync.RWMutex
	Entries map[string]*v1.UIPluginEntry `json:"entries,omitempty"`
}

// Generate generates a new index from a UIPluginCache object
func (s *SafeIndex) Generate(namespace string, cache plugincontroller.UIPluginCache) error {
	logrus.Debug("generating index from plugin controller's cache")
	s.mu.Lock()
	defer s.mu.Unlock()
	cachedPlugins, err := cache.List(namespace, labels.Everything())
	if err != nil {
		return err
	}
	s.Entries = make(map[string]*v1.UIPluginEntry, len(cachedPlugins))
	for _, plugin := range cachedPlugins {
		entry := &plugin.Spec.Plugin
		logrus.Debugf("adding plugin to index: %+v", *entry)
		s.Entries[entry.Name] = entry
	}

	return nil
}

// SyncWithFsCache syncs up entries in the filesystem cache with the index's entries
// Entries that aren't in the index, but present in the filesystem cache are deleted
func (s *SafeIndex) SyncWithFsCache() error {
	pattern := FSCacheRootDir + "/*/*"
	files, err := filepath.Glob(pattern)
	logrus.Debugf("files matching glob pattern [%s] found in filesystem cache: %+v", pattern, files)
	if err != nil {
		return err
	}
	for _, file := range files {
		logrus.Debugf("syncing index with filesystem cache")
		// Splits /{root}/{pluginName}/{pluginVersion}/* from a fs cache path
		rel, _ := filepath.Rel(FSCacheRootDir, file)
		s := strings.Split(rel, "/")
		name := s[0]
		version := s[1]
		_, ok := Index.Entries[name]
		if ok && Index.Entries[name].Version == version {
			continue
		} else {
			// Delete plugin entry from filesystem cache
			err = os.RemoveAll(filepath.Join(FSCacheRootDir, name))
			if err != nil {
				logrus.Errorf("failed to delete entry [Name: %s Version: %s] from filesystem cache: %s", name, version, err.Error())
				return err
			}
			logrus.Debugf("deleted plugin entry from cache [Name: %s Version: %s]", name, version)
		}
	}

	return nil
}
