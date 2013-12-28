package assets

import (
	"gnd.la/loaders"
	"gnd.la/log"
	"gnd.la/util/hashutil"
	"io"
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	watcher      *Watcher
	loader       loaders.Loader
	prefix       string
	prefixLength int
	cache        map[string]string
	mutex        sync.RWMutex
}

func NewManager(loader loaders.Loader, prefix string) *Manager {
	m := new(Manager)
	m.cache = make(map[string]string)
	m.loader = loader
	m.SetPrefix(prefix)
	runtime.SetFinalizer(m, func(manager *Manager) {
		manager.Close()
	})
	m.watch()
	return m
}

func (m *Manager) watch() {
	if dirloader, ok := m.Loader().(loaders.DirLoader); ok {
		watcher, err := NewWatcher(dirloader.Dir(), func(name string, deleted bool) {
			m.mutex.RLock()
			_, ok := m.cache[name]
			m.mutex.RUnlock()
			if ok {
				m.mutex.Lock()
				if deleted {
					delete(m.cache, name)
				} else {
					h, err := m.hash(name)
					if err == nil {
						m.cache[name] = h
					} else {
						delete(m.cache, name)
					}
				}
				m.mutex.Unlock()
			}
		})
		if err != nil {
			log.Warningf("Error creating watcher for %s: %s", dirloader.Dir, err)
		} else if watcher != nil {
			if err := watcher.Watch(); err == nil {
				m.watcher = watcher
			} else {
				log.Warningf("Error watching %s: %s", dirloader.Dir, err)
				watcher.Close()
			}
		}
	}
}

func (m *Manager) hash(name string) (string, error) {
	r, _, err := m.Load(name)
	if err != nil {
		return "", err
	}
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return hashutil.Adler32(b)[:6], nil
}

func (m *Manager) Loader() loaders.Loader {
	return m.loader
}

func (m *Manager) Load(name string) (loaders.ReadSeekCloser, time.Time, error) {
	return m.loader.Load(name)
}

func (m *Manager) LoadURL(u *url.URL) (loaders.ReadSeekCloser, time.Time, error) {
	p := u.Path
	if !(p[1] == 'f' || p[1] == 'r') && !(p == "/favicon.ico" || p == "/robots.txt") {
		p = p[m.prefixLength:]
	}
	p = filepath.FromSlash(path.Clean(p))
	return m.Load(p)
}

func (m *Manager) Create(name string, overwrite bool) (io.WriteCloser, error) {
	return m.loader.Create(name, overwrite)
}

func (m *Manager) URL(name string) string {
	if strings.HasPrefix(name, "//") || strings.Contains(name, "://") {
		return name
	}
	m.mutex.RLock()
	h, ok := m.cache[name]
	m.mutex.RUnlock()
	if !ok {
		h, _ = m.hash(name)
		m.mutex.Lock()
		m.cache[name] = h
		m.mutex.Unlock()
	}
	clean := path.Clean(path.Join(m.prefix, name))
	if h != "" {
		return clean + "?v=" + h
	}
	return clean
}

func (m *Manager) Prefix() string {
	return m.prefix
}

func (m *Manager) SetPrefix(prefix string) {
	if prefix != "" && prefix[len(prefix)-1] != '/' {
		prefix = prefix + "/"
	}
	m.prefix = prefix
	m.prefixLength = len(prefix)
}

func (m *Manager) Close() error {
	if m.watcher != nil {
		err := m.watcher.Close()
		m.watcher = nil
		return err
	}
	return nil
}
