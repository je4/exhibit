package browserControl

import (
	"context"
	"emperror.dev/errors"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/je4/exhibit/v2/pkg/browser"
	"github.com/je4/utils/v2/pkg/zLogger"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type BrowserControl struct {
	browser       *browser.Browser
	homeUrl       *url.URL
	opts          map[string]any
	timeout       int64
	logger        zLogger.ZLogger
	lastLog       int64
	lastLogMutex  sync.RWMutex
	stop          chan any
	allowedPrefix []string
	taskDelay     time.Duration
}

func NewBrowserControl(allowedPrefix []string, homeUrl *url.URL, opts map[string]any, timeout, taskDelay time.Duration, logger zLogger.ZLogger) (*BrowserControl, error) {
	bc := &BrowserControl{
		allowedPrefix: allowedPrefix,
		homeUrl:       homeUrl,
		opts:          opts,
		timeout:       int64(timeout.Seconds()),
		taskDelay:     taskDelay,
		logger:        logger,
	}
	return bc, nil
}

func (bc *BrowserControl) log(str string, evs ...any) {
	atomic.StoreInt64(&bc.lastLog, time.Now().Unix())
	if len(evs) == 0 {
		return
	}
	ev := evs[0]
	switch ev := ev.(type) {
	case *network.EventRequestWillBeSent:
		// must not block
		go func(ctx context.Context, ev *network.EventRequestWillBeSent) {
			var ok bool
			for _, prefix := range bc.allowedPrefix {
				if strings.HasPrefix(ev.DocumentURL, prefix) {
					ok = true
					break
				}
			}
			if !ok {
				bc.logger.Info().Msgf("forbidden URL: %s", ev.DocumentURL)
				tasks := chromedp.Tasks{
					chromedp.Navigate(bc.homeUrl.String()),
					//		browser.MouseClickXYAction(2,2),
				}
				if err := bc.browser.Tasks(tasks); err != nil {
					bc.logger.Err(err).Msgf("could not navigate: %v", err)
				}
			}
		}(bc.browser.TaskCtx, ev)
	}
	//bc.logger.Debugf("%s - %v", str, param)
}

func (bc *BrowserControl) Start() error {
	var err error
	bc.browser, err = browser.NewBrowser(bc.opts, bc.logger, bc.log)
	if err != nil {
		return errors.Wrap(err, "cannot create browser instance")
	}
	// ensure that the browser process is started
	if err := bc.browser.Run(); err != nil {
		return errors.Wrap(err, "cannot run browser")
	}

	path := filepath.Join(bc.browser.TempDir, "DevToolsActivePort")
	bs, err := os.ReadFile(path)
	if err != nil {
		bc.logger.Panic().Msgf("error reading DevToolsActivePort: %v", err)
	}
	//	lines := bytes.Split(bs, []byte("\n"))
	bc.logger.Debug().Msgf("DevToolsActivePort:\n%v", string(bs))
	tasks := chromedp.Tasks{
		chromedp.Navigate(bc.homeUrl.String()),
		//		browser.MouseClickXYAction(2,2),
		fetch.Disable(),
	}
	time.Sleep(bc.taskDelay)
	err = bc.browser.Tasks(tasks)
	if err != nil {
		bc.logger.Err(err).Msgf("could not navigate: %v", err)
	}

	bc.stop = make(chan any)

	go bc.mainLoop()

	return nil
}

func (bc *BrowserControl) Shutdown() {
	close(bc.stop)
	if bc.browser == nil {
		return
	}
	bc.browser.Close()
}

func (bc *BrowserControl) mainLoop() {
	for {
		select {
		case <-time.After(time.Second * 3):
		case <-bc.stop:
			return
		}
		if !bc.browser.IsRunning() {
			bc.browser.Startup()
			tasks := chromedp.Tasks{
				chromedp.Navigate(bc.homeUrl.String()),
				//		browser.MouseClickXYAction(2,2),
			}
			if err := bc.browser.Tasks(tasks); err != nil {
				bc.logger.Err(err).Msgf("could not navigate: %v", err)
			}
		}
		llog := atomic.LoadInt64(&bc.lastLog)
		timeout := time.Now().Unix()-llog > bc.timeout
		if timeout {
			tasks := chromedp.Tasks{
				chromedp.Navigate(bc.homeUrl.String()),
				//		browser.MouseClickXYAction(2,2),
			}
			if err := bc.browser.Tasks(tasks); err != nil {
				bc.logger.Err(err).Msgf("could not navigate: %v", err)
			}
		}
	}
}
