```File: server  
Type: inuse_space  
Time: Oct 21, 2025 at 4:22pm (BST)  
Duration: 40.02s, Total samples = 1991.92kB   
Showing nodes accounting for -544.67kB, 27.34% of 1991.92kB total  
   flat    flat%   sum%        cum   cum%  
 -544.67kB 27.34% 27.34%  -544.67kB 27.34%  compress/flate.(*compressor).init  
         0     0% 27.34%  -544.67kB 27.34%  compress/flate.NewWriter (inline)  
         0     0% 27.34%   902.59kB 45.31%  compress/gzip.(*Writer).Close  
         0     0% 27.34%  -544.67kB 27.34%  compress/gzip.(*Writer).Write  
         0     0% 27.34% -1447.25kB 72.66%  github.com/go-chi/chi/v5.(*Mux).Mount.func1  
         0     0% 27.34%  -544.67kB 27.34%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP  
         0     0% 27.34% -1447.25kB 72.66%  github.com/go-chi/chi/v5.(*Mux).routeHTTP  
         0     0% 27.34%  -544.67kB 27.34%  github.com/skdiver33/metrics-collector/internal/server.  (*MetricsHandler).GzipHandle-fm.(*MetricsHandler).GzipHandle.func1  
         0     0% 27.34%  -544.67kB 27.34%  github.com/skdiver33/metrics-collector/internal/server.(*MetricsHandler).RequestLogger-fm.(*MetricsHandler).RequestLogger.func1  
         0     0% 27.34% -1447.25kB 72.66%  github.com/skdiver33/metrics-collector/internal/server.(*MetricsHandler).SetJSONMetrics  
         0     0% 27.34%  -544.67kB 27.34%  github.com/skdiver33/metrics-collector/internal/server.(*MetricsHandler).SigningHandle-fm.(*MetricsHandler).SigningHandle.func1  
         0     0% 27.34% -1447.25kB 72.66%  github.com/skdiver33/metrics-collector/internal/server.gzipWriter.Write  
         0     0% 27.34%  -544.67kB 27.34%  net/http.(*conn).serve  
         0     0% 27.34%  -544.67kB 27.34%  net/http.HandlerFunc.ServeHTTP  
         0     0% 27.34%  -544.67kB 27.34%  net/http.serverHandler.ServeHTTP  ```
