package gits

import (
	"fmt"
	"strings"
)

func OpenRepo(conf *Config) (*Repo, error) {
	r := &Repo{
		conf: conf,
	}

	var err error

	if r.fs == nil {
		r.fs, err = NewDiskFS(conf.Dir)
	} else {
		r.fs, err = conf.FS(conf.Dir)
	}

	return r, err
}

func InitRepo(conf *Config) (*Repo, error) {
	r := &Repo{
		conf: conf,
	}

	var err error

	if r.fs == nil {
		r.fs, err = NewDiskFS(conf.Dir)
	} else {
		r.fs, err = conf.FS(conf.Dir)
	}

	if err != nil {
		return nil, err
	}

	dir := r.absPath(conf.Name)
	stat := r.fs.Stat(dir)

	if stat[0] != 0 {
		return nil, fmt.Errorf("repo '%s' already exists", conf.Name)
	}

	return r, r.fs.Mkdir(dir)
}

// Helpers.
func (repo *Repo) absPath(path string) string {
	return "/" + repo.conf.Name + "/" + strings.Trim(path, "/")
}
