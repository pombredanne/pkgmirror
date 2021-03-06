// Copyright © 2016 Thomas Rabaix <thomas.rabaix@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package git

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/AaronO/go-git-http"
	log "github.com/Sirupsen/logrus"
	"github.com/rande/goapp"
	"github.com/rande/gonode/core/vault"
	"github.com/rande/pkgmirror"
	"goji.io"
	"goji.io/pat"
	"golang.org/x/net/context"
)

func ConfigureApp(config *pkgmirror.Config, l *goapp.Lifecycle) {

	l.Register(func(app *goapp.App) error {
		logger := app.Get("logger").(*log.Logger)

		vault := &vault.Vault{
			Algo: "no_op",
			Driver: &vault.DriverFs{
				Root: fmt.Sprintf("%s/git", config.CacheDir),
			},
		}

		for name, conf := range config.Git {
			if !conf.Enabled {
				continue
			}

			app.Set(fmt.Sprintf("pkgmirror.git.%s", name), func(name string, conf *pkgmirror.GitConfig) func(app *goapp.App) interface{} {

				return func(app *goapp.App) interface{} {
					s := NewGitService()
					s.Config.Server = conf.Server
					s.Config.PublicServer = config.PublicServer
					s.Config.DataDir = fmt.Sprintf("%s/git", config.DataDir)
					s.Config.Clone = conf.Clone
					s.Vault = vault
					s.Logger = logger.WithFields(log.Fields{
						"handler": "git",
						"code":    name,
					})
					s.StateChan = pkgmirror.GetStateChannel(fmt.Sprintf("pkgmirror.git.%s", name), app.Get("pkgmirror.channel.state").(chan pkgmirror.State))
					s.Init(app)

					return s
				}
			}(name, conf))
		}

		return nil
	})

	l.Prepare(func(app *goapp.App) error {
		for name, conf := range config.Git {
			if !conf.Enabled {
				continue
			}

			ConfigureHttp(name, conf, app)
		}

		logger := app.Get("logger").(*log.Logger)

		mux := app.Get("mux").(*goji.Mux)

		// disable push, RO repository
		gitServer := githttp.New(config.DataDir)
		gitServer.ReceivePack = false
		gitServer.EventHandler = func(ev githttp.Event) {
			entry := logger.WithFields(log.Fields{
				"commit": ev.Commit,
				"type":   ev.Type.String(),
				"dir":    ev.Dir,
			})

			if ev.Error != nil {
				entry.WithError(ev.Error).Info("Git server error")
			} else {
				entry.Debug("Git command received")
			}
		}

		preAction := func(fn http.Handler) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {

				for name := range config.Git {
					path := "/git/" + name

					if len(r.URL.Path) > len(path) && path == r.URL.Path[0:len(path)] {

						//found match
						s := app.Get(fmt.Sprintf("pkgmirror.git.%s", name)).(*GitService)

						if len(s.Config.Clone) == 0 {
							break // not configured, so skip clone
						}

						reg := regexp.MustCompile(fmt.Sprintf(`/git/%s/((.*)\.git)(|.*)`, name))

						path := ""
						if results := reg.FindStringSubmatch(r.URL.Path); len(results) > 0 {
							path = results[1]
						} else {
							break // not valid
						}

						if s.Has(path) { // repository exists, nothing to do
							break
						}

						// not available, clone the repository
						if err := s.Clone(path); err != nil {
							logger.WithError(err).Error("Unable to clone the repository")
						}

						break
					}
				}

				fn.ServeHTTP(w, r)
			}
		}

		mux.HandleFunc(pat.Get("/git/*"), preAction(gitServer))
		mux.HandleFunc(pat.Post("/git/*"), preAction(gitServer))

		return nil
	})

	for name, conf := range config.Git {
		if !conf.Enabled {
			continue
		}

		l.Run(func(name string) func(app *goapp.App, state *goapp.GoroutineState) error {
			return func(app *goapp.App, state *goapp.GoroutineState) error {
				s := app.Get(fmt.Sprintf("pkgmirror.git.%s", name)).(pkgmirror.MirrorService)
				s.Serve(state)

				return nil
			}
		}(name))
	}
}

func ConfigureHttp(name string, conf *pkgmirror.GitConfig, app *goapp.App) {
	gitService := app.Get(fmt.Sprintf("pkgmirror.git.%s", name)).(*GitService)

	mux := app.Get("mux").(*goji.Mux)

	mux.HandleFuncC(NewGitPat(conf.Server), func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		if err := gitService.WriteArchive(w, fmt.Sprintf("%s.git", pat.Param(ctx, "path")), pat.Param(ctx, "ref")); err != nil {
			pkgmirror.SendWithHttpCode(w, 500, err.Error())
		}
	})
}
