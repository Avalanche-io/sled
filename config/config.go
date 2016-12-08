package config

import (
	"os"
	"strings"

	"github.com/mitchellh/go-homedir"
)

type Config struct {
	Root *Path
	DB   *Filename
}

var (
	DefaultRoot                         = "{{home}}/.sled/"
	DefaultStorageSubdir                = "storage"
	DefaultConfig        func() *Config = dc
)

func dc() *Config {
	home, _ := homedir.Dir()
	p, _ := NewPath(strings.Replace(DefaultRoot, "{{home}}", home, -1))
	cfg := Config{p, nil}
	return &cfg
}

func New() *Config {
	return DefaultConfig()
}

func (c *Config) Mkdirs() {
	path := c.StoragePath().String()
	os.MkdirAll(path, 07777)
}

func (c *Config) DbPath() *string {
	if c.DB == nil {
		return nil
	}
	path := c.Root.String() + c.DB.String()
	return &path
}

func (c *Config) StoragePath() *Path {
	path, _ := NewPath(DefaultStorageSubdir)
	return c.Root.Append(path)
}

func (c *Config) RootPath() *string {
	path := c.Root.String()
	return &path
}

func (c *Config) FilePath(name *Filename) string {
	return c.StoragePath().String() + name.String()
}

func (c *Config) WithRoot(path string) *Config {
	c.Root, _ = NewPath(path)
	return c
}

func (c *Config) WithDB(filename string) *Config {
	c.DB = NewFilename(filename)
	return c
}
