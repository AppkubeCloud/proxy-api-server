package routing

import (
	"net/http"
	"proxy-api-server/handlers"

	"github.com/gorilla/mux"
)

// Route describes a single route
type Route struct {
	Name          string
	Method        string
	Pattern       string
	HandlerFunc   http.HandlerFunc
	Authenticated bool
}

// Routes holds an array of Route. A note on swagger documentation. The path variables and query parameters
// are defined in ../doc.go.  YOu need to manually associate params and routes.
type Routes struct {
	Routes []Route
}

// NewRoutes creates and returns all the API routes
func NewRoutes() (r *Routes) {
	r = new(Routes)

	r.Routes = []Route{

		// swagger:route GET /grafana/dashboard
		// ---
		// Endpoint to get list of grafana dashboards with panels and datasources
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http, https
		//
		// responses:
		//      500: internalError
		//      200: statusInfo
		{
			"GrafanaDashboard",
			"GET",
			"/grafana/dashboard",
			handlers.GrafanaDashboardHandler,
			true,
		},
		// swagger:route GET /grafana/dashboard/uid
		// ---
		// Endpoint to get the grafana dashboard by uid
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http, https
		//
		// responses:
		//      500: internalError
		//      200: statusInfo
		{
			"GrafanaDashboardByUID",
			"GET",
			"/grafana/dashboard/uid",
			handlers.GetGrafanaDashbordByUidHandler,
			true,
		},
		// swagger:route POST /grafana/create-dashboard
		// ---
		// Endpoint to create dashboard in grafana
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http, https
		//
		// responses:
		//      500: internalError
		//      200: statusInfo
		{
			"GrafanaCreateDashboard",
			"POST",
			"/grafana/create-dashboard",
			handlers.GrafanaApiHandler,
			true,
		},
		// swagger:route GET /grafana/query
		// ---
		// Endpoint to get the grafana details by query
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http, https
		//
		// responses:
		//      500: internalError
		//      200: statusInfo
		{
			"GrafanaQuery",
			"GET",
			"/grafana/query",
			handlers.GrafanaQueryHandler,
			true,
		},
		// swagger:route POST /grafana/query
		// ---
		// Endpoint to get the grafana query
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http, https
		//
		// responses:
		//      500: internalError
		//      200: statusInfo
		{
			"GrafanaQuery",
			"POST",
			"/grafana/query",
			handlers.GrafanaApiHandler,
			true,
		},
		// swagger:route GET /grafana/query-range
		// ---
		// Endpoint to get the grafana details by query range
		//
		//     Produces:
		//     - application/json
		//
		//     Schemes: http, https
		//
		// responses:
		//      500: internalError
		//      200: statusInfo
		{
			"GrafanaQueryRange",
			"GET",
			"/grafana/query-range",
			handlers.GrafanaQueryRangeHandler,
			true,
		},
	}

	return
}

func NewRouter() *mux.Router {

	// conf := config.Get()
	// webRoot := "" //conf.Server.WebRoot
	// webRootWithSlash := webRoot + "/"

	appRouter := mux.NewRouter().StrictSlash(true)
	// appRouter := rootRouter

	// staticFileServer := http.FileServer(http.Dir(conf.Server.StaticContentRootDirectory))s

	// if webRoot != "/" {
	// 	// help the user out - if a request comes in for "/", redirect to our true webroot
	// 	rootRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 		http.Redirect(w, r, webRootWithSlash, http.StatusFound)
	// 	})

	// 	appRouter = rootRouter.PathPrefix(conf.Server.WebRoot).Subrouter()
	// 	staticFileServer = http.StripPrefix(webRootWithSlash, staticFileServer)

	// 	// Because of OIDC, when we receive a request for the webroot without
	// 	// the trailing slash, we can not redirect the user to the correct
	// 	// webroot as the hash params are lost (and they are not sent to the
	// 	// server).
	// 	//
	// 	// See https://github.com/kiali/kiali/issues/3103
	// 	rootRouter.HandleFunc(webRoot, func(w http.ResponseWriter, r *http.Request) {
	// 		r.URL.Path = webRootWithSlash
	// 		rootRouter.ServeHTTP(w, r)
	// 	})
	// } else {
	// webRootWithSlash = "/"
	// }

	// fileServerHandler := func(w http.ResponseWriter, r *http.Request) {
	// 	urlPath := r.RequestURI
	// 	if r.URL != nil {
	// 		urlPath = r.URL.Path
	// 	}

	// 	if urlPath == webRootWithSlash || urlPath == webRoot || urlPath == webRootWithSlash+"index.html" {
	// 		serveIndexFile(w)
	// 	} else if urlPath == webRootWithSlash+"env.js" {
	// 		serveEnvJsFile(w)
	// 	} else {
	// 		staticFileServer.ServeHTTP(w, r)
	// 	}
	// }

	// appRouter = appRouter.StrictSlash(true)

	// Build our API server routes and install them.
	apiRoutes := NewRoutes()
	// authenticationHandler, _ := handlers.NewAuthenticationHandler()
	for _, route := range apiRoutes.Routes {
		// handlerFunction := metricHandler(route.HandlerFunc, route)
		// if route.Authenticated {
		// 	handlerFunction = authenticationHandler.Handle(handlerFunction)
		// } else {
		// 	handlerFunction = authenticationHandler.HandleUnauthenticated(handlerFunction)
		// }
		appRouter.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(route.HandlerFunc)
	}

	// if authController := authentication.GetAuthController(); authController != nil {
	// 	if ac, ok := authController.(*authentication.OpenIdAuthController); ok {
	// 		ac.PostRoutes(appRouter)
	// 	}
	// }

	// All client-side routes are prefixed with /console.
	// They are forwarded to index.html and will be handled by react-router.
	// appRouter.PathPrefix("/console").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	serveIndexFile(w)
	// })

	// if authController := authentication.GetAuthController(); authController != nil {
	// 	if ac, ok := authController.(*authentication.OpenIdAuthController); ok {
	// 		authCallback := ac.GetAuthCallbackHandler(http.HandlerFunc(fileServerHandler))
	// 		rootRouter.Methods("GET").Path(webRootWithSlash).Handler(authCallback)
	// 	}
	// }

	// rootRouter.PathPrefix(webRootWithSlash).HandlerFunc(fileServerHandler)

	return appRouter
}

// statusResponseWriter contains a ResponseWriter and a StatusCode to read in the metrics middleware
type statusResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

// WriteHeader will be called by any function that needs to set an status code, in this function the StatusCode is also set
func (srw *statusResponseWriter) WriteHeader(code int) {
	srw.ResponseWriter.WriteHeader(code)
	srw.StatusCode = code
}

// updateMetric evaluates the StatusCode, if there is an error, increase the API failure counter, otherwise save the duration
// func updateMetric(route string, srw *statusResponseWriter, timer *prometheus.Timer) {
// 	// Always measure the duration even if the API call ended in an error
// 	timer.ObserveDuration()
// 	// Increase the error counter on 500 and 503 errors
// 	if srw.StatusCode == http.StatusInternalServerError || srw.StatusCode == http.StatusServiceUnavailable {
// 		internalmetrics.GetAPIFailureMetric(route).Inc()
// 	}
// }

// func metricHandler(next http.Handler, route Route) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// By default, if there is no call to WriteHeader, an 200 will be
// 		srw := &statusResponseWriter{
// 			ResponseWriter: w,
// 			StatusCode:     http.StatusOK,
// 		}
// 		promtimer := internalmetrics.GetAPIProcessingTimePrometheusTimer(route.Name)
// 		defer updateMetric(route.Name, srw, promtimer)
// 		next.ServeHTTP(srw, r)
// 	})
// }

// serveEnvJsFile generates the env.js file needed by the UI from Kiali configs. The
// generated file is sent to the HTTP response.
// func serveEnvJsFile(w http.ResponseWriter) {
// 	conf := config.Get()
// 	var body string
// 	if len(conf.Server.WebHistoryMode) > 0 {
// 		body += fmt.Sprintf("window.HISTORY_MODE='%s';", conf.Server.WebHistoryMode)
// 	}

// 	body += "window.WEB_ROOT = document.getElementsByTagName('base')[0].getAttribute('href').replace(/^https?:\\/\\/[^#?\\/]+/g, '').replace(/\\/+$/g, '')"

// 	w.Header().Set("content-type", "text/javascript")
// 	_, err := io.WriteString(w, body)
// 	if err != nil {
// 		log.Errorf("HTTP I/O error [%v]", err.Error())
// 	}
// }

// serveIndexFile takes UI's index.html as a template to generate a modified index file that takes
// into account the web_root path configured in the Kiali CR. The result is sent to the HTTP response.
// func serveIndexFile(w http.ResponseWriter) {
// 	webRootPath := config.Get().Server.WebRoot
// 	webRootPath = strings.TrimSuffix(webRootPath, "/")

// 	path, _ := filepath.Abs("./console/index.html")
// 	b, err := ioutil.ReadFile(path)
// 	if err != nil {
// 		log.Errorf("File I/O error [%v]", err.Error())
// 		handlers.RespondWithDetailedError(w, http.StatusInternalServerError, "Unable to read index.html template file", err.Error())
// 		return
// 	}

// 	html := string(b)
// 	newHTML := html

// 	if len(webRootPath) != 0 {
// 		searchStr := `<base href="/"`
// 		newStr := `<base href="` + webRootPath + `/"`
// 		newHTML = strings.Replace(html, searchStr, newStr, -1)
// 	}

// 	w.Header().Set("content-type", "text/html")
// 	_, err = io.WriteString(w, newHTML)
// 	if err != nil {
// 		log.Errorf("HTTP I/O error [%v]", err.Error())
// 	}
// }
