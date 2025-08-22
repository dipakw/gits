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

	if err != nil {
		return nil, err
	}

	return r, r.fs.Cd(conf.Name)
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

	stat := r.fs.Stat(conf.Name)

	if stat[0] != 0 {
		return nil, fmt.Errorf("repo '%s' already exists", conf.Name)
	}

	err = r.fs.Mkdir(conf.Name)

	if err != nil {
		return nil, err
	}

	return r, r.fs.Cd(conf.Name)
}

// Helpers.
func (repo *Repo) absPath(path string) string {
	return "/" + repo.conf.Name + "/" + strings.Trim(path, "/")
}
