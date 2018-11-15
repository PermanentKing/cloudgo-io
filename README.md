# cloudgo-io

设计一个 web 小应用，展示静态文件服务、js 请求支持、模板输出、表单处理、Filter 中间件设计等方面的能力。（不需要数据库支持）

#### 基本要求

1. 支持静态文件服务
2. 支持简单 js 访问
3. 提交表单，并输出一个表格
4. 对 /unknown 给出开发中的提示，返回码 5xx

可以通过以下博客来进行设计和完成作业

[golang web 服务器 request 与 response 处理](https://blog.csdn.net/pmlpml/article/details/78539261)

### 完成结果如下：
#### 静态文件服务

![](pic/pic1.png)

#### 简单js访问

![](pic/pic2.png)

#### 提交表单并输出表格

![](pic/pic4.png)

![](pic/pic3.png)


#### 对/unknown给出开发中提示，返回码5xx

![](pic/pic6.png)

![](pic/pic5.png)

![](pic/pic7.png)

![](pic/pic5.png)



### 分析`gzip`源码

```go
// Package gzip implements a gzip compression handler middleware for Negroni.
package gzip

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/urfave/negroni"
)

// These compression constants are copied from the compress/gzip package.
const (
	encodingGzip = "gzip"

	// request或response的header
	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
	headerContentLength   = "Content-Length"
	headerContentType     = "Content-Type"
	headerVary            = "Vary"
	headerSecWebSocketKey = "Sec-WebSocket-Key"

	//压缩等级
	BestCompression    = gzip.BestCompression
	BestSpeed          = gzip.BestSpeed
	DefaultCompression = gzip.DefaultCompression
	NoCompression      = gzip.NoCompression
)

// gzipResponseWriter is the ResponseWriter that negroni.ResponseWriter is
// wrapped in.
type gzipResponseWriter struct {
	w *gzip.Writer
	negroni.ResponseWriter
	wroteHeader bool
}

// Check whether underlying response is already pre-encoded and disable
// gzipWriter before the body gets written, otherwise encoding headers
//// 检查并设置response的header,并把wroteHeader 设为true
func (grw *gzipResponseWriter) WriteHeader(code int) {
	headers := grw.ResponseWriter.Header()
	if headers.Get(headerContentEncoding) == "" {
		headers.Set(headerContentEncoding, encodingGzip)
		headers.Add(headerVary, headerAcceptEncoding)
	} else {
		grw.w.Reset(ioutil.Discard)
		grw.w = nil
	}

	// Avoid sending Content-Length header before compression. The length would
	// be invalid, and some browsers like Safari will report
	// "The network connection was lost." errors
	grw.Header().Del(headerContentLength)

	// 设置状态码
	grw.ResponseWriter.WriteHeader(code)
	grw.wroteHeader = true
}

// Write writes bytes to the gzip.Writer. It will also set the Content-Type
// header using the net/http library content type detection if the Content-Type
// header was not set yet.
// 把byte数据写入
func (grw *gzipResponseWriter) Write(b []byte) (int, error) {
	if !grw.wroteHeader {
		grw.WriteHeader(http.StatusOK)
	}
	if grw.w == nil {
		return grw.ResponseWriter.Write(b)
	}
	if len(grw.Header().Get(headerContentType)) == 0 {
		grw.Header().Set(headerContentType, http.DetectContentType(b))
	}
	return grw.w.Write(b)
}

type gzipResponseWriterCloseNotifier struct {
	*gzipResponseWriter
}

func (rw *gzipResponseWriterCloseNotifier) CloseNotify() <-chan bool {
	return rw.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

func newGzipResponseWriter(rw negroni.ResponseWriter, w *gzip.Writer) negroni.ResponseWriter {
	wr := &gzipResponseWriter{w: w, ResponseWriter: rw}

	if _, ok := rw.(http.CloseNotifier); ok {
		return &gzipResponseWriterCloseNotifier{gzipResponseWriter: wr}
	}

	return wr
}

// handler struct contains the ServeHTTP method
type handler struct {
	pool sync.Pool
}

// Gzip returns a handler which will handle the Gzip compression in ServeHTTP.
// Valid values for level are identical to those in the compress/gzip package.
// 根据压缩level来进行，返回一个handler去处理ServeHTTP中的gzip压缩
func Gzip(level int) *handler {
	h := &handler{}
	h.pool.New = func() interface{} {
		gz, err := gzip.NewWriterLevel(ioutil.Discard, level)
		if err != nil {
			panic(err)
		}
		return gz
	}
	return h
}

// ServeHTTP wraps the http.ResponseWriter with a gzip.Writer.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Skip compression if the client doesn't accept gzip encoding.
	if !strings.Contains(r.Header.Get(headerAcceptEncoding), encodingGzip) {
		next(w, r)
		return
	}

	// Skip compression if client attempt WebSocket connection
	if len(r.Header.Get(headerSecWebSocketKey)) > 0 {
		next(w, r)
		return
	}

	// Retrieve gzip writer from the pool. Reset it to use the ResponseWriter.
	// This allows us to re-use an already allocated buffer rather than
	// allocating a new buffer for every request.
	// We defer g.pool.Put here so that the gz writer is returned to the
	// pool if any thing after here fails for some reason (functions in
	// next could potentially panic, etc)
	gz := h.pool.Get().(*gzip.Writer)
	defer h.pool.Put(gz)
	gz.Reset(w)

	// Wrap the original http.ResponseWriter with negroni.ResponseWriter
	// and create the gzipResponseWriter.
	nrw := negroni.NewResponseWriter(w)
	grw := newGzipResponseWriter(nrw, gz)

	// Call the next handler supplying the gzipResponseWriter instead of
	// the original.
	next(grw, r)

	gz.Close()
}

```