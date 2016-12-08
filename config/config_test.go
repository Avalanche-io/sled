package config_test

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"

	"github.com/cheekybits/is"

	"github.com/Avalanche-io/sled/config"
)

func TestDefaultRootPath(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	homedir, _ := homedir.Dir()

	tests := []struct {
		Template           string
		ReplaceDefaultPath bool
	}{
		{
			Template:           "{{home}}/.sled/",
			ReplaceDefaultPath: false,
		},
		{
			Template:           "{{home}}/test/.sled/",
			ReplaceDefaultPath: true,
		},
		{
			Template:           "/mnt/database/sled/",
			ReplaceDefaultPath: true,
		},
	}

	t.Log("do")
	for _, test := range tests {
		path := strings.Replace(test.Template, "{{home}}", homedir, -1)
		expected, err := config.NewPath(path)
		is.NoErr(err)
		if test.ReplaceDefaultPath {
			config.DefaultRoot = test.Template
		}
		cfg := config.New()

		t.Log("check")
		is.Equal(cfg.Root, expected)
	}
}

func TestWithRoot(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	path := "/mnt/sled/"

	cfg := config.New().WithRoot(path)
	is.Equal(*cfg.RootPath(), path)
}

func TestFilePath(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	path := "/mnt/sled/"

	cfg := config.New().WithRoot(path)
	filename := config.NewFilename("foo.bar")
	is.Equal(cfg.FilePath(filename), path+"storage/foo.bar")
}

func TestDefaultMkdirs(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	dir, err := ioutil.TempDir("/tmp", "sledTest_TestReadWrite_")
	path := dir + "/.sled/"
	is.NoErr(err)
	defer os.RemoveAll(dir)

	t.Log("do")
	config.DefaultRoot = path
	cfg := config.New()
	cfg.Mkdirs()

	t.Log("check")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		is.Failf("Expected path %s doesn't exist.", path)
	}
}

func TestDb(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	tmp, err := ioutil.TempDir("/tmp", "sledTest_TestReadWrite_")
	path := tmp + "/.sled/"
	is.NoErr(err)
	defer os.RemoveAll(tmp)

	t.Log("do")
	config.DefaultRoot = path
	cfgDefault := config.New()

	cfgDb := config.New().WithDB("sled.db")

	t.Log("check")
	is.Nil(cfgDefault.DbPath())
	is.Equal(*cfgDb.DbPath(), path+"sled.db")
}
