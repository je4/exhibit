package main

import (
	"flag"
	"fmt"
	"github.com/je4/exhibit/v2/config"
	"github.com/je4/exhibit/v2/pkg/browserControl"
	configutil "github.com/je4/utils/v2/pkg/config"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/rs/zerolog"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var configfile = flag.String("config", "", "location of toml configuration file")

func main() {
	flag.Parse()

	var cfgFS fs.FS
	var cfgFile string
	if *configfile != "" {
		cfgFS = os.DirFS(filepath.Dir(*configfile))
		cfgFile = filepath.Base(*configfile)
	} else {
		cfgFS = config.ConfigFS
		cfgFile = "exhibit.toml"
	}

	conf := &config.ExhibitConfig{
		LogFile:          "",
		LogLevel:         "ERROR",
		BrowserTimeout:   configutil.Duration(time.Minute * 5),
		BrowserTaskDelay: configutil.Duration(time.Second * 2),
		BrowserURL:       "https://performance.ausstellung.cc/zoom/de?exhibition",
		AllowedPrefixes:  []string{"https://ba14ns21403-sec1.fhnw.ch"},
	}

	if err := config.LoadExhibitConfig(cfgFS, cfgFile, conf); err != nil {
		log.Fatalf("cannot load toml from [%v] %s: %v", cfgFS, cfgFile, err)
	}

	// create logger instance
	var out io.Writer = os.Stdout
	if conf.LogFile != "" {
		fp, err := os.OpenFile(conf.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("cannot open logfile %s: %v", conf.LogFile, err)
		}
		defer fp.Close()
		out = fp
	}

	//	output := zerolog.ConsoleWriter{Out: out, TimeFormat: time.RFC3339}
	_logger := zerolog.New(out).With().Timestamp().Logger()
	switch strings.ToUpper(conf.LogLevel) {
	case "DEBUG":
		_logger = _logger.Level(zerolog.DebugLevel)
	case "INFO":
		_logger = _logger.Level(zerolog.InfoLevel)
	case "WARN":
		_logger = _logger.Level(zerolog.WarnLevel)
	case "ERROR":
		_logger = _logger.Level(zerolog.ErrorLevel)
	case "FATAL":
		_logger = _logger.Level(zerolog.FatalLevel)
	case "PANIC":
		_logger = _logger.Level(zerolog.PanicLevel)
	default:
		_logger = _logger.Level(zerolog.DebugLevel)
	}
	var logger zLogger.ZLogger = &_logger

	var b *browserControl.BrowserControl
	opts := map[string]any{
		"headless":                            false,
		"start-fullscreen":                    true,
		"disable-notifications":               true,
		"disable-infobars":                    true,
		"disable-gpu":                         false,
		"disable-audio-output":                false,
		"mute-audio":                          false,
		"allow-insecure-localhost":            true,
		"enable-immersive-fullscreen-toolbar": true,
		"views-browser-windows":               false,
		"kiosk":                               true,
		"disable-session-crashed-bubble":      true,
		"incognito":                           true,
		//				"enable-features":                     "PreloadMediaEngagementData,AutoplayIgnoreWebAudio,MediaEngagementBypassAutoplayPolicies",
		//			"disable-features": "InfiniteSessionRestore,TranslateUI,PreloadMediaEngagementData,AutoplayIgnoreWebAudio,MediaEngagementBypassAutoplayPolicies",
		"disable-features": "InfiniteSessionRestore,TranslateUI,PreloadMediaEngagementData,AutoplayIgnoreWebAudio,MediaEngagementBypassAutoplayPolicies",
		//"no-first-run":                        true,
		"enable-fullscreen-toolbar-reveal": false,
		"useAutomationExtension":           false,
		"enable-automation":                false,
	}
	homeUrl, err := url.Parse(conf.BrowserURL)
	if err != nil {
		logger.Panic().Msgf("cannot parse %s: %v", conf.BrowserURL, err)
	}
	url2, _ := url.Parse(conf.BrowserURL)
	url2.Path = ""
	url2.RawQuery = ""
	b, err = browserControl.NewBrowserControl(append(conf.AllowedPrefixes, url2.String()), homeUrl, opts, time.Duration(conf.BrowserTimeout), time.Duration(conf.BrowserTaskDelay), logger)
	if err != nil {
		logger.Panic().Msgf("cannot create browser control: %v", err)
	}
	time.Sleep(time.Millisecond * 500)
	b.Start()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	fmt.Println("press ctrl+c to stop server")
	s := <-done
	fmt.Println("got signal:", s)
	b.Shutdown()
}
