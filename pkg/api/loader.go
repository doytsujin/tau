package api

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-getter"
	log "github.com/sirupsen/logrus"
)

type loader struct {
	config  *Config
	tempDir string
	src     string
	pwd     string
	loaded  map[string]*Module
}

func newLoader(config *Config) (*loader, error) {
	tempDir := filepath.Join(config.WorkingDirectory, ".tau", hash(config.Source))

	if config.CleanTempDir {
		log.Debugf("Cleaning temp directory...")
		// TODO os.RemoveAll(config.WorkingDirectory)
	}

	loader := &loader{
		config:  config,
		tempDir: tempDir,
		src:     config.Source,
		loaded:  map[string]*Module{},
	}

	if err := loader.setSourceDirectory(); err != nil {
		return nil, err
	}

	return loader, nil
}

func (l *loader) load() error {
	log.WithField("blank_before", true).Info("Loading modules...")

	modules, err := l.loadModules(l.src)
	if err != nil {
		return err
	}

	log.WithField("blank_before", true).Info("Loading dependencies...")
	if err := l.loadDependencies(modules, 0); err != nil {
		return err
	}

	return nil
}

func (l *loader) loadModules(src string) ([]*Module, error) {
	dst := filepath.Join(l.tempDir, "sources", hash(src))

	if err := l.loadSources(src, dst); err != nil {
		return nil, err
	}

	files, err := l.findModuleFiles(dst)
	if err != nil {
		return nil, err
	}

	modules := []*Module{}
	for _, file := range files {
		module, err := NewModule(file)
		if err != nil {
			return nil, err
		}

		modules = append(modules, module)
	}

	return modules, nil
}

func (l *loader) loadDependencies(modules []*Module, depth int) error {
	remaining := []*Module{}

	for _, module := range modules {
		if _, ok := l.loaded[module.Hash()]; !ok {
			remaining = append(remaining, module)
		}
	}

	for _, module := range remaining {
		deps, err := l.loadModuleDependencies(module)
		if err != nil {
			return err
		}

		if err := l.loadDependencies(deps, depth+1); err != nil {
			return err
		}
	}

	return nil
}

func (l *loader) loadModuleDependencies(module *Module) ([]*Module, error) {
	l.loaded[module.Hash()] = module
	deps := []*Module{}

	for _, dep := range module.config.Dependencies {
		modules, err := l.loadModules(dep.Source)
		if err != nil {
			return nil, err
		}

		for _, mod := range modules {
			hash := mod.Hash()

			if _, ok := l.loaded[hash]; !ok {
				deps = append(deps, mod)
			} else {
				mod = l.loaded[hash]
			}

			mod.deps[dep.Name] = mod
		}
	}

	return deps, nil
}

func (l *loader) loadSources(src, dst string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	log.Debugf("Loading sources for %v", src)

	client := &getter.Client{
		Ctx:  ctx,
		Src:  src,
		Dst:  dst,
		Pwd:  l.pwd,
		Mode: getter.ClientModeAny,
	}

	return client.Get()
}

func (l *loader) findModuleFiles(dst string) ([]string, error) {

	matches := []string{}

	for _, ext := range []string{"*.hcl", "*.tau"} {
		m, err := filepath.Glob(filepath.Join(dst, ext))
		if err != nil {
			return nil, err
		}

		for _, match := range m {
			fi, err := os.Stat(match)

			if err != nil {
				return nil, err
			}

			if !fi.IsDir() {
				matches = append(matches, match)
			}
		}
	}

	log.Debugf("Found %v template file(s): %v", len(matches), matches)

	return matches, nil
}

// func NewLoader(src string) *Loader {
// 	source, err := getRootSource(src)
// 	if err != nil {
// 		log.Fatalf("Unable to get root source: %s", err)
// 	}

// 	return &Loader{
// 		Source:  *source,
// 		modules: map[string]*Module{},
// 	}
// }

// func (l *Loader) Load() error {
// 	log.WithField("blank_before", true).Info("Loading modules...")

// 	modules, err := l.loadModules(Root)
// 	if err != nil {
// 		return err
// 	}

// 	for _, module := range modules {
// 		l.modules[module.Hash()] = module
// 	}

// 	log.WithField("blank_before", true).Info("Loading dependencies...")
// 	return l.resolveRemainingDependencies(0)
// }

// func (l *Loader) GetSortedModules() []*Module {
// 	modules := []*Module{}

// 	for _, mod := range l.modules {
// 		if mod.level == Root {
// 			modules = append(modules, mod)
// 		}
// 	}

// 	sort.Sort(ByDependencies(modules))

// 	return modules
// }

// func (l *Loader) resolveRemainingDependencies(depth int) error {
// 	if depth >= maxDependencyDepth {
// 		return fmt.Errorf("Max dependency depth reached (%v)", maxDependencyDepth)
// 	}

// 	mods := []*Module{}

// 	for _, m := range l.modules {
// 		if m.deps == nil {
// 			mods = append(mods, m)
// 		}
// 	}

// 	if len(mods) == 0 {
// 		return nil
// 	}

// 	for _, mod := range mods {
// 		if err := mod.resolveDependencies(l.modules); err != nil {
// 			return err
// 		}
// 	}

// 	return l.resolveRemainingDependencies(depth + 1)
// }

func (l *loader) setSourceDirectory() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	getterSource, err := getter.Detect(l.src, pwd, getter.Detectors)
	if err != nil {
		return err
	}

	if strings.Index(getterSource, "file://") == 0 {
		log.Debug("File source detected. Changing source directory")
		rootPath := strings.Replace(getterSource, "file://", "", 1)

		fi, err := os.Stat(rootPath)

		if err != nil {
			return err
		}

		if !fi.IsDir() {
			l.pwd = path.Dir(rootPath)
			l.src = path.Base(rootPath)
		} else {
			l.pwd = rootPath
			l.src = "."
		}

		log.Debugf("New source directory: %v", pwd)
		log.Debugf("New source: %v", l.src)
	}

	return nil
}
