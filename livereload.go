package sg

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

// LiveReload manages WebSocket connections for automatic browser refresh.
type LiveReload struct {
	mu    sync.Mutex
	conns []net.Conn
}

// NewLiveReload creates a new LiveReload instance.
func NewLiveReload() *LiveReload {
	return &LiveReload{}
}

// LiveReloadScript is the JavaScript snippet injected into served HTML pages.
// It connects to the WebSocket endpoint and reloads the page on receiving a message.
const LiveReloadScript = `<script>
(function() {
  var ws, retry = 1000;
  function connect() {
    ws = new WebSocket("ws://" + location.host + "/ws");
    ws.onmessage = function() { location.reload(); };
    ws.onclose = function() { setTimeout(connect, retry); retry = Math.min(retry * 2, 10000); };
    ws.onopen = function() { retry = 1000; };
  }
  connect();
})();
</script>`

// ServeHTTP handles the WebSocket upgrade handshake using the standard library.
func (lr *LiveReload) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "expected websocket", http.StatusBadRequest)
		return
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	// Compute accept key per RFC 6455.
	h := sha1.New()
	h.Write([]byte(key + "258EAFA5-E914-47DA-95CA-5AB5DC175B18"))
	accept := base64.StdEncoding.EncodeToString(h.Sum(nil))

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "server does not support hijacking", http.StatusInternalServerError)
		return
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send upgrade response.
	fmt.Fprintf(buf, "HTTP/1.1 101 Switching Protocols\r\n")
	fmt.Fprintf(buf, "Upgrade: websocket\r\n")
	fmt.Fprintf(buf, "Connection: Upgrade\r\n")
	fmt.Fprintf(buf, "Sec-WebSocket-Accept: %s\r\n", accept)
	fmt.Fprintf(buf, "\r\n")
	buf.Flush()

	lr.mu.Lock()
	lr.conns = append(lr.conns, conn)
	lr.mu.Unlock()

	// Keep the connection open; read until close.
	discard := make([]byte, 512)
	for {
		if _, err := conn.Read(discard); err != nil {
			break
		}
	}

	lr.mu.Lock()
	for i, c := range lr.conns {
		if c == conn {
			lr.conns = append(lr.conns[:i], lr.conns[i+1:]...)
			break
		}
	}
	lr.mu.Unlock()
	conn.Close()
}

// Reload sends a reload message to all connected WebSocket clients.
func (lr *LiveReload) Reload() {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	msg := []byte("reload")
	frame := wsTextFrame(msg)

	alive := lr.conns[:0]
	for _, conn := range lr.conns {
		if _, err := conn.Write(frame); err != nil {
			conn.Close()
			continue
		}
		alive = append(alive, conn)
	}
	lr.conns = alive
}

// wsTextFrame constructs a WebSocket text frame (opcode 0x1, unmasked).
func wsTextFrame(payload []byte) []byte {
	n := len(payload)
	var frame []byte
	if n < 126 {
		frame = make([]byte, 2+n)
		frame[0] = 0x81 // FIN + text
		frame[1] = byte(n)
		copy(frame[2:], payload)
	} else if n < 65536 {
		frame = make([]byte, 4+n)
		frame[0] = 0x81
		frame[1] = 126
		binary.BigEndian.PutUint16(frame[2:], uint16(n))
		copy(frame[4:], payload)
	} else {
		frame = make([]byte, 10+n)
		frame[0] = 0x81
		frame[1] = 127
		binary.BigEndian.PutUint64(frame[2:], uint64(n))
		copy(frame[10:], payload)
	}
	return frame
}

// InjectLiveReload wraps an http.Handler to inject the LiveReload script
// into HTML responses before the closing </body> tag.
func InjectLiveReload(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ".html") && r.URL.Path != "/" &&
			!strings.HasSuffix(r.URL.Path, "/") {
			next.ServeHTTP(w, r)
			return
		}
		rec := &responseRecorder{ResponseWriter: w, body: make([]byte, 0, 4096)}
		next.ServeHTTP(rec, r)

		body := string(rec.body)
		if idx := strings.LastIndex(body, "</body>"); idx >= 0 {
			body = body[:idx] + LiveReloadScript + "\n" + body[idx:]
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.WriteHeader(rec.statusCode)
		w.Write([]byte(body))
	})
}

// responseRecorder captures the response body for injection.
type responseRecorder struct {
	http.ResponseWriter
	body       []byte
	statusCode int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
}
