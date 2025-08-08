// /home/krylon/go/src/github.com/blicero/carebear/server/server.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 06. 2024 by Benjamin Walkenhorst
// (c) 2024 Benjamin Walkenhorst
// Time-stamp: <2025-08-08 18:56:07 krylon>

package web

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database"
	"github.com/blicero/carebear/logdomain"
	"github.com/blicero/carebear/model"
	"github.com/gorilla/mux"
)

const ( // nolint: deadcode
	poolSize     = 4
	cacheControl = "max-age=3600, public"
	noCache      = "no-store, max-age=0"
)

//go:embed assets
var assets embed.FS

// Server wraps the state required for the web interface
type Server struct {
	addr      string
	log       *log.Logger
	pool      *database.Pool
	lock      sync.RWMutex // nolint: unused,structcheck
	active    atomic.Bool
	router    *mux.Router
	mbuf      *msgBuf
	tmpl      *template.Template
	web       http.Server
	mimeTypes map[string]string
}

// Create creates and returns a new Server.
func Create(addr string) (*Server, error) {
	var (
		err error
		msg string
		srv = &Server{
			addr: addr,
			mbuf: newMsgBuf(),
			mimeTypes: map[string]string{
				".css":  "text/css",
				".map":  "application/json",
				".js":   "text/javascript",
				".png":  "image/png",
				".jpg":  "image/jpeg",
				".jpeg": "image/jpeg",
				".webp": "image/webp",
				".gif":  "image/gif",
				".json": "application/json",
				".html": "text/html",
			},
		}
	)

	if srv.log, err = common.GetLogger(logdomain.Web); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Error creating Logger: %s\n",
			err.Error())
		return nil, err
	} else if srv.pool, err = database.NewPool(poolSize); err != nil {
		srv.log.Printf("[ERROR] Cannot allocate database connection pool: %s\n",
			err.Error())
		return nil, err
	} else if srv.pool == nil {
		srv.log.Printf("[CANTHAPPEN] Database pool is nil!\n")
		return nil, errors.New("Database pool is nil")
	}

	const tmplFolder = "assets/templates"
	var templates []fs.DirEntry
	var tmplRe = regexp.MustCompile("[.]tmpl$")

	if templates, err = assets.ReadDir(tmplFolder); err != nil {
		srv.log.Printf("[ERROR] Cannot read embedded templates: %s\n",
			err.Error())
		return nil, err
	}

	srv.tmpl = template.New("").Funcs(funcmap)
	for _, entry := range templates {
		var (
			content []byte
			path    = filepath.Join(tmplFolder, entry.Name())
		)

		if !tmplRe.MatchString(entry.Name()) {
			continue
		} else if content, err = assets.ReadFile(path); err != nil {
			msg = fmt.Sprintf("Cannot read embedded file %s: %s",
				path,
				err.Error())
			srv.log.Printf("[CRITICAL] %s\n", msg)
			return nil, errors.New(msg)
		} else if srv.tmpl, err = srv.tmpl.Parse(string(content)); err != nil {
			msg = fmt.Sprintf("Could not parse template %s: %s",
				entry.Name(),
				err.Error())
			srv.log.Println("[CRITICAL] " + msg)
			return nil, errors.New(msg)
		} else if common.Debug {
			srv.log.Printf("[TRACE] Template \"%s\" was parsed successfully.\n",
				entry.Name())
		}
	}

	srv.router = mux.NewRouter()
	srv.web.Addr = addr
	srv.web.ErrorLog = srv.log
	srv.web.Handler = srv.router

	// Web interface handlers
	srv.router.HandleFunc("/favicon.ico", srv.handleFavIco)
	srv.router.HandleFunc("/static/{file}", srv.handleStaticFile)
	srv.router.HandleFunc("/{page:(?:index|main|start)?$}", srv.handleMain)
	srv.router.HandleFunc("/network/all", srv.handleNetworkAll)
	srv.router.HandleFunc("/network/{id:(?:\\d+)$}", srv.handleNetworkDetails)
	srv.router.HandleFunc("/device/all", srv.handleDeviceAll)
	srv.router.HandleFunc("/device/{id:(?:\\d+)$}", srv.handleDeviceDetails)

	// AJAX Handlers
	srv.router.HandleFunc("/ajax/beacon", srv.handleBeacon)

	return srv, nil
} // func Create(addr string) (*Server, error)

// IsActive returns the Server's active flag.
func (srv *Server) IsActive() bool {
	return srv.active.Load()
} // func (srv *Server) IsActive() bool

// Stop clears the Server's active flag.
func (srv *Server) Stop() {
	srv.active.Store(false)
} // func (srv *Server) Stop()

// SendMessage puts a message in the Server's message buffer.
func (srv *Server) SendMessage(msg string) {
	var m = &message{
		Timestamp: time.Now(),
		Level:     "DEBUG",
		Message:   msg,
	}
	srv.mbuf.put(m)
} // func (srv *Server) SendMessage(msg *message)

// Run executes the Server's loop, waiting for new connections and starting
// goroutines to handle them.
func (srv *Server) Run() {
	var err error

	defer srv.log.Println("[INFO] Web server is shutting down")

	srv.log.Printf("[INFO] Web frontend is going online at %s\n", srv.addr)
	http.Handle("/", srv.router)

	if err = srv.web.ListenAndServe(); err != nil {
		if err.Error() != "http: Server closed" {
			srv.log.Printf("[ERROR] ListenAndServe returned an error: %s\n",
				err.Error())
		} else {
			srv.log.Println("[INFO] HTTP Server has shut down.")
		}
	}
} // func (srv *Server) Run()

func (srv *Server) handleMain(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	const (
		tmplName  = "main"
		recentCnt = 20
	)

	var (
		err  error
		msg  string
		db   *database.Database
		tmpl *template.Template
		data = tmplDataIndex{
			tmplDataBase: tmplDataBase{
				Title: "Main",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	data.Messages = srv.mbuf.getAll()

	w.Header().Set("Cache-Control", cacheControl)
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.SendMessage(msg)
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleMain(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleNetworkAll(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	const (
		tmplName = "network_all"
	)

	var (
		err  error
		msg  string
		db   *database.Database
		tmpl *template.Template
		data = tmplDataNetworkAll{
			tmplDataBase: tmplDataBase{
				Title: "All Networks",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Networks, err = db.NetworkGetAll(); err != nil {
		msg = fmt.Sprintf("Failed to load networks from database: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.DevCnt, err = db.NetworkDevCnt(); err != nil {
		msg = fmt.Sprintf("Failed to load device count per network from database: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		srv.log.Printf("[ERROR] Failed to render template %s: %s\n",
			tmplName,
			err.Error())
	}
} // func (srv *Server) handleNetworkAll(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleNetworkDetails(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	const (
		tmplName = "network_details"
	)

	var (
		err   error
		msg   string
		db    *database.Database
		tmpl  *template.Template
		vars  map[string]string
		idStr string
		netID int64
		data  = tmplDataNetworkDetails{
			tmplDataBase: tmplDataBase{
				//Title: "Networks",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	vars = mux.Vars(r)
	idStr = vars["id"]

	if netID, err = strconv.ParseInt(idStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse network ID %q: %s",
			idStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Network, err = db.NetworkGetByID(netID); err != nil {
		msg = fmt.Sprintf("Failed to lookup network #%d: %s",
			netID,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.Devices, err = db.DeviceGetByNetwork(data.Network); err != nil {
		msg = fmt.Sprintf("Failed to load devices for Network %d (%s): %s",
			netID,
			data.Network.Addr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	data.Title = fmt.Sprintf("Details for Network %s (%d)",
		data.Network.Addr,
		data.Network.ID)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		srv.log.Printf("[ERROR] Failed to render template %s: %s\n",
			tmplName,
			err.Error())
	}
} // func (srv *Server) handleNetworkDetails(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleDeviceAll(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	const (
		tmplName = "device_all"
	)

	var (
		err  error
		msg  string
		db   *database.Database
		tmpl *template.Template
		data = tmplDataDeviceAll{
			tmplDataBase: tmplDataBase{
				Title: "All Devices",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Devices, err = db.DeviceGetAll(); err != nil {
		msg = fmt.Sprintf("Failed to load all devices: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		srv.log.Printf("[ERROR] Failed to render template %s: %s\n",
			tmplName,
			err.Error())
	}
} // func (srv *Server) handleDeviceAll(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleDeviceDetails(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	const (
		tmplName = "device_details"
	)

	var (
		err        error
		msg, idStr string
		id         int64
		vars       map[string]string
		db         *database.Database
		upd        []*model.Updates
		uptime     []*model.Uptime
		tmpl       *template.Template
		data       = tmplDataDeviceDetails{
			tmplDataBase: tmplDataBase{
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	vars = mux.Vars(r)
	idStr = vars["id"]

	if id, err = strconv.ParseInt(idStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse Device ID %q: %s",
			idStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Device, err = db.DeviceGetByID(id); err != nil {
		msg = fmt.Sprintf("Failed to load device %d: %s",
			id,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.Device == nil {
		msg = fmt.Sprintf("Device %d was not found in database", id)
		srv.log.Printf("[INFO] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.Network, err = db.NetworkGetByID(data.Device.NetID); err != nil {
		msg = fmt.Sprintf("Failed to load Network %d for device %s (%d): %s",
			data.Device.NetID,
			data.Device.Name,
			data.Device.ID,
			err.Error())
		srv.log.Printf("[ERROR] %s\n",
			msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if uptime, err = db.UptimeGetByDevice(data.Device, 1); err != nil {
		msg = fmt.Sprintf("Failed to load system load average for %s (%d): %s",
			data.Device.Name,
			data.Device.ID,
			err.Error())
		srv.log.Printf("[ERROR] %s\n",
			msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if upd, err = db.UpdatesGetByDevice(data.Device, 1); err != nil {
		msg = fmt.Sprintf("Failed to load recent Updates for %s (%d): %s",
			data.Device.Name,
			data.Device.ID,
			err.Error())
		srv.log.Printf("[ERROR] %s\n",
			msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	if len(upd) > 0 {
		data.Updates = upd[0]
	}
	if len(uptime) > 0 {
		data.Uptime = uptime[0]
	}
	data.Title = fmt.Sprintf("Details for Device %s", data.Device.Name)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		srv.log.Printf("[ERROR] Failed to render template %s: %s\n",
			tmplName,
			err.Error())
	}
} // func (srv *Server) handleDeviceDetails(w http.ResponseWriter, r *http.Request)

//////////////////////////////////////////////////////////////////////////////
/// Handle static assets /////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////

func (srv *Server) handleFavIco(w http.ResponseWriter, request *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		request.URL.EscapedPath())

	const (
		filename = "assets/static/favicon.ico"
		mimeType = "image/vnd.microsoft.icon"
	)

	w.Header().Set("Content-Type", mimeType)

	if !common.Debug {
		w.Header().Set("Cache-Control", "max-age=7200")
	} else {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
	}

	var (
		err error
		fh  fs.File
	)

	if fh, err = assets.Open(filename); err != nil {
		msg := fmt.Sprintf("ERROR - cannot find file %s", filename)
		srv.sendErrorMessage(w, msg)
	} else {
		defer fh.Close()
		w.WriteHeader(200)
		io.Copy(w, fh) // nolint: errcheck
	}
} // func (srv *Server) handleFavIco(w http.ResponseWriter, request *http.Request)

func (srv *Server) handleStaticFile(w http.ResponseWriter, request *http.Request) {
	// srv.log.Printf("[TRACE] Handle request for %s\n",
	// 	request.URL.EscapedPath())

	// Since we controll what static files the server has available, we
	// can easily map MIME type to slice. Soon.

	vars := mux.Vars(request)
	filename := vars["file"]
	path := filepath.Join("assets", "static", filename)

	var mimeType string

	srv.log.Printf("[TRACE] Delivering static file %s to client\n", filename)

	var match []string

	if match = common.SuffixPattern.FindStringSubmatch(filename); match == nil {
		mimeType = "text/plain"
	} else if mime, ok := srv.mimeTypes[match[1]]; ok {
		mimeType = mime
	} else {
		srv.log.Printf("[ERROR] Did not find MIME type for %s\n", filename)
	}

	w.Header().Set("Content-Type", mimeType)

	if common.Debug {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
	} else {
		w.Header().Set("Cache-Control", "max-age=7200")
	}

	var (
		err error
		fh  fs.File
	)

	if fh, err = assets.Open(path); err != nil {
		msg := fmt.Sprintf("ERROR - cannot find file %s", path)
		srv.sendErrorMessage(w, msg)
	} else {
		defer fh.Close()
		w.WriteHeader(200)
		io.Copy(w, fh) // nolint: errcheck
	}
} // func (srv *Server) handleStaticFile(w http.ResponseWriter, request *http.Request)

func (srv *Server) sendErrorMessage(w http.ResponseWriter, msg string) {
	html := `
<!DOCTYPE html>
<html>
  <head>
    <title>Internal Error</title>
  </head>
  <body>
    <h1>Internal Error</h1>
    <hr />
    We are sorry to inform you an internal application error has occured:<br />
    %s
    <p>
    Back to <a href="/index">Homepage</a>
    <hr />
    &copy; 2018 <a href="mailto:krylon@gmx.net">Benjamin Walkenhorst</a>
  </body>
</html>
`

	srv.log.Printf("[ERROR] %s\n", msg)

	output := fmt.Sprintf(html, msg)
	w.WriteHeader(500)
	_, _ = w.Write([]byte(output)) // nolint: gosec
} // func (srv *Server) sendErrorMessage(w http.ResponseWriter, msg string)
