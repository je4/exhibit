package config

import (
	"embed"
	"emperror.dev/errors"
	"github.com/BurntSushi/toml"
	configutil "github.com/je4/utils/v2/pkg/config"
	"io/fs"
	"os"
)

//go:embed exhibit.toml
var ConfigFS embed.FS

type ExhibitConfig struct {
	LogFile          string              `toml:"logfile"`
	LogLevel         string              `toml:"loglevel"`
	BrowserTimeout   configutil.Duration `toml:"browsertimeout"`
	BrowserURL       string              `toml:"browserurl"`
	BrowserTaskDelay configutil.Duration `toml:"browsertaskdelay"`
	StartTimeout     configutil.Duration `toml:"starttimeout"`
	AllowedPrefixes  []string            `toml:"allowedprefixes"`
}

func LoadExhibitConfig(fSys fs.FS, fp string, conf *ExhibitConfig) error {
	if _, err := fs.Stat(fSys, fp); err != nil {
		path, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "cannot get current working directory")
		}
		fSys = os.DirFS(path)
		fp = "exhibit.toml"
	}
	data, err := fs.ReadFile(fSys, fp)
	if err != nil {
		return errors.Wrapf(err, "cannot read file [%v] %s", fSys, fp)
	}
	_, err = toml.Decode(string(data), conf)
	if err != nil {
		return errors.Wrapf(err, "error loading config file %v", fp)
	}
	return nil
}
