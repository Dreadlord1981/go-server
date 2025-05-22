package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/shurcooL/httpgzip"
)

type Hive struct {
	Path  string `json:"path"`
	Hive  string `json:"hive"`
	Host  string `json:"host"`
	Route string `json:"route"`
	Java  bool   `json:"java"`
}

type Hives struct {
	LocalHives    []Hive `json:"localhives"`
	RemoteHives   []Hive `json:"remotehives"`
	CriticalHives []Hive `json:"criticalhives"`
}

type Server struct {
	HTTPS bool   `json:"https"`
	Name  string `json:"name"`
	Host  string `json:"host"`
	Port  string `json:"port"`
	Hives Hives  `json:"hives"`
}

type Servers struct {
	List []Server `json:"servers"`
}

func logRequest(handler http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		fmt.Printf("%s %s\n", r.Method, r.RequestURI)
		handler.ServeHTTP(w, r)
	})
}

func printMenu() {
	fmt.Println("\u001b[2J\u001b[0;0H")
	fmt.Println("**************************")
	fmt.Println("*                        *")
	fmt.Println("*    Server boot menu    *")
	fmt.Println("*                        *")
	fmt.Println("*                        *")
	fmt.Println("**************************")
	fmt.Println("")
	fmt.Println("1 - List servers")
	fmt.Println("2 - Quit")
}

func printList(servers []Server) {

	fmt.Println("\u001b[2J\u001b[0;0H")
	fmt.Println("**************************")
	fmt.Println("*                        *")
	fmt.Println("*       Server List      *")
	fmt.Println("*                        *")
	fmt.Println("*                        *")
	fmt.Println("**************************")
	fmt.Println("")

	for i := 0; i < len(servers); i++ {

		var server = servers[i]

		fmt.Println("Option: " + server.Name)
	}
}

func changeHeaderThenServe(h http.Handler, verbose *bool, caching *bool) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		headers := w.Header()

		if !*caching {
			headers.Add("Cache-Control", "no-store")
		} else {
			headers.Add("Cache-Control", "max-age=3600")
		}

		headers.Add("Service-Worker-Allowed", "/")

		if *verbose {

			requestDump, err := httputil.DumpRequest(r, true)

			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("--------------------------------------------------------------------\n")
				fmt.Printf("Request: \n")
				fmt.Println(string(requestDump))

				if r.Method == "GET" {

					values := r.URL.Query()

					if len(values) > 0 {
						fmt.Println("Params: ")

						for k, _ := range values {
							v := values.Get(k)
							fmt.Printf("Key: %s\t Value: %s \n", k, v)
						}
					}

				}

				fmt.Printf("--------------------------------------------------------------------\n")
			}

		}

		h.ServeHTTP(w, r)
	}
}

func caselessMatcher(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		r.URL.Path = strings.ToLower(r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

func upgradeInsecure(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		headers := w.Header()

		headers.Add("content-security-policy", "upgrade-insecure-requests")

		next.ServeHTTP(w, r)
	})
}

func correct_MIME_TYPE(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if strings.Index(r.URL.Path, ".js") > 0 {

			headers := w.Header()

			if _, ok := headers["Content-Type"]; ok {
				headers.Set("Content-Type", "application/javascript;charset=utf-8")
			} else {
				headers.Add("Content-Type", "application/javascript;charset=utf-8")
			}
		}

		next.ServeHTTP(w, r)
	})
}

func main() {

	basePath, _ := os.Getwd()

	serverPtr := flag.String("s", "", "Server to use")
	verbosePtr := flag.Bool("v", false, "Verbose to show hive info")
	listPtr := flag.Bool("l", false, "List all servers")
	helpPtr := flag.Bool("h", false, "Display help page")
	caseFlag := flag.Bool("c", false, "Allows for urls to keep the casing")
	filePtr := flag.String("f", "", "Path to go.json file")
	portPtr := flag.String("p", "", "Port to run the server")
	cachePtr := flag.Bool("a", false, "Enable caching")

	flag.Parse()

	if *helpPtr {
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		os.Exit(2)
	}

	var configPath = "./go.json"

	if *filePtr != "" {
		configPath, _ = filepath.Abs(os.ExpandEnv(*filePtr + "/go.json"))
		basePath, _ = filepath.Abs(os.ExpandEnv(*filePtr))
	}

	file, err := ioutil.ReadFile(configPath)

	if err != nil {
		fmt.Println(err)
		fmt.Println(configPath)
		os.Exit(2)
	}

	var servers Servers
	var server *Server

	jsonerr := json.Unmarshal(file, &servers)

	if jsonerr != nil {
		fmt.Println(jsonerr)
		os.Exit(2)
	}

	var serverName string
	var serverPort string

	if *listPtr {

		printList(servers.List)

		fmt.Print("> ")

		fmt.Scanln(&serverName)
	}

	if serverName == "" && *serverPtr == "" {

		printMenu()

		var nMenu int
		nMenu = 0

		var ans = false

		for !ans {

			fmt.Print("> ")

			if nMenu == 0 {

				var nInput int
				fmt.Scan(&nInput)

				if nInput == 1 {
					printList(servers.List)
					nMenu = 1
				} else if nInput == 2 {
					ans = true
					os.Exit(0)
				} else {
					fmt.Println("Invalid option")
				}

			} else {
				fmt.Scanln(&serverName)
				ans = true
			}
		}

	} else {
		if serverName == "" {
			serverName = *serverPtr
		}

	}

	for i := 0; i < len(servers.List); i++ {

		var s = servers.List[i]

		if s.Name == serverName {
			server = &s
		}
	}

	if server != nil {

		serverPort = server.Port

		if *portPtr != "" {
			serverPort = *portPtr
		}

		fmt.Println("Option: " + serverName)

		router := mux.NewRouter()

		var hives = server.Hives

		for i := 0; i < len(hives.CriticalHives); i++ {

			var hive = hives.CriticalHives[i]

			router.PathPrefix(hive.Path).HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

				baseurl := server.Host

				if hive.Host != "" {
					baseurl = hive.Host
				}

				newBaseURL := baseurl

				destinationURL := fmt.Sprintf("%s%s", newBaseURL, request.RequestURI)

				if hive.Route != "" {
					destinationURL = strings.Replace(destinationURL, hive.Path, hive.Route, 1)
				}

				url, err := url.Parse(destinationURL)

				if err != nil {
					log.Fatalln(err)
				}

				// create the reverse proxy
				target := url
				targetQuery := target.RawQuery
				proxy := &httputil.ReverseProxy{
					Director: func(req *http.Request) {
						req.URL.Scheme = target.Scheme
						req.URL.Host = target.Host
						req.URL.Path = target.Path
						req.URL.RawPath = target.RawPath
						req.Host = target.Host
						if targetQuery == "" || req.URL.RawQuery == "" {
							req.URL.RawQuery = targetQuery + req.URL.RawQuery
						} else {
							req.URL.RawQuery = targetQuery
						}
						if _, ok := req.Header["User-Agent"]; !ok {
							// explicitly disable User-Agent so it's not set to default value
							req.Header.Set("User-Agent", "")
						}

						if *verbosePtr {

							requestDump, err := httputil.DumpRequest(req, true)

							if err != nil {
								fmt.Println(err)
							} else {
								fmt.Printf("--------------------------------------------------------------------\n")
								fmt.Printf("Request: \n")
								fmt.Println(string(requestDump))

								if req.Method == "GET" {

									values := req.URL.Query()

									if len(values) > 0 {
										fmt.Println("Params: ")

										for k, _ := range values {
											v := values.Get(k)
											fmt.Printf("Key: %s\t Value: %s \n", k, v)
										}
									}

								}

								fmt.Printf("--------------------------------------------------------------------\n")
							}

						}
					},
					ModifyResponse: func(r *http.Response) error {

						if *cachePtr {
							r.Header.Set("Cache-Control", "max-age=3600")
						}

						return nil
					},
				}

				proxy.ServeHTTP(writer, request)
			})
		}

		for i := 0; i < len(hives.LocalHives); i++ {

			var hive = hives.LocalHives[i]

			var path, _ = filepath.Abs(os.ExpandEnv(hive.Path))

			if hive.Path != "" {

				if hive.Path[0] == '.' {
					path, _ = filepath.Abs(os.ExpandEnv(basePath + hive.Path))
				}
			}

			if *verbosePtr {
				fmt.Printf("--------------------------------------------------------------------\n")
				fmt.Printf("HIVE: %s\n", hive.Hive)
				fmt.Printf("PATH: %s\n", hive.Path)
				fmt.Printf("FORMATTED: %s\n", path)
				fmt.Printf("--------------------------------------------------------------------\n")

			}

			var serveHive = httpgzip.FileServer(http.Dir(path), httpgzip.FileServerOptions{})

			hivePath := strings.ToLower(hive.Hive)

			router.PathPrefix(hivePath).Handler(http.StripPrefix(hivePath, changeHeaderThenServe(serveHive, verbosePtr, cachePtr)))
		}

		for i := 0; i < len(hives.RemoteHives); i++ {

			var hive = hives.RemoteHives[i]

			router.PathPrefix(hive.Path).HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

				baseurl := server.Host

				if hive.Host != "" {
					baseurl = hive.Host
				}

				newBaseURL := baseurl

				destinationURL := fmt.Sprintf("%s%s", newBaseURL, request.RequestURI)

				if hive.Route != "" {
					destinationURL = strings.Replace(destinationURL, hive.Path, hive.Route, 1)
				}

				url, err := url.Parse(destinationURL)

				if err != nil {
					log.Fatalln(err)
				}

				// create the reverse proxy

				target := url
				targetQuery := target.RawQuery
				proxy := &httputil.ReverseProxy{
					Director: func(req *http.Request) {
						req.URL.Scheme = target.Scheme
						req.URL.Host = target.Host
						req.URL.Path = target.Path
						req.URL.RawPath = target.RawPath
						req.Host = target.Host
						if targetQuery == "" || req.URL.RawQuery == "" {
							req.URL.RawQuery = targetQuery + req.URL.RawQuery
						} else {
							req.URL.RawQuery = targetQuery
						}
						if _, ok := req.Header["User-Agent"]; !ok {
							// explicitly disable User-Agent so it's not set to default value
							req.Header.Set("User-Agent", "")
						}

						if *verbosePtr {

							requestDump, err := httputil.DumpRequest(req, true)

							if err != nil {
								fmt.Println(err)
							} else {
								fmt.Printf("--------------------------------------------------------------------\n")
								fmt.Printf("Request: \n")
								fmt.Printf("Schema: %s", target.Scheme)
								fmt.Printf("\n")
								fmt.Println(string(requestDump))

								if req.Method == "GET" {

									values := req.URL.Query()

									if len(values) > 0 {
										fmt.Println("Params: ")

										for k, _ := range values {
											v := values.Get(k)
											fmt.Printf("Key: %s\t Value: %s \n", k, v)
										}
									}

								}
								fmt.Printf("--------------------------------------------------------------------\n")
							}

						}
					},
					ModifyResponse: func(r *http.Response) error {

						if *cachePtr {
							r.Header.Set("Cache-Control", "max-age=3600")
						}

						return nil
					},
				}

				proxy.ServeHTTP(writer, request)
			})
		}

		serveBase := http.FileServer(http.Dir(basePath))

		router.Handle("/{path:.*}", serveBase)
		router.Use(correct_MIME_TYPE)

		var handler http.Handler

		if !*caseFlag {
			handler = caselessMatcher(router)
		} else {
			handler = router
		}

		http.Handle("/", handler)

		if server.HTTPS {

			fmt.Println("Running server at https://localhost:" + serverPort)

			cert, err := filepath.Abs(os.ExpandEnv("$POPATH/../ssl/server.cert"))

			if err != nil {
				log.Fatalln(err.Error())
			}

			key, err := filepath.Abs(os.ExpandEnv("$POPATH/../ssl/server.key"))

			if err != nil {
				log.Fatalln(err.Error())
			}

			log.Fatal(http.ListenAndServeTLS(":"+serverPort, cert, key, upgradeInsecure(logRequest(http.DefaultServeMux))))

		} else {

			fmt.Println("Running server at http://localhost:" + serverPort)

			log.Fatal(http.ListenAndServe(":"+serverPort, logRequest(http.DefaultServeMux)))

		}

	} else {
		fmt.Println("No server found with the name " + serverName)
		fmt.Println("")
		fmt.Println("Try one of the following...:")
		fmt.Println("****************************")

		for i := 0; i < len(servers.List); i++ {

			var server = servers.List[i]

			fmt.Println(server.Name)
		}
		fmt.Println("****************************")

	}
}
