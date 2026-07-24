package figmaasset

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type daemonServer struct {
	port     int
	http     *http.Server
	upgrader websocket.Upgrader
	mu       sync.Mutex
	writeMu  sync.Mutex
	plugin   *websocket.Conn
	pending  map[string]chan PluginExportResult
	shutdown chan struct{}
	shutOnce sync.Once
}

func RunDaemon(port int) error {
	listenAddr := daemonListenAddress(port)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", listenAddr, err)
	}

	state, err := writeDaemonState(os.Getpid())
	if err != nil {
		_ = listener.Close()
		return err
	}
	defer removeDaemonState(state.Nonce)

	server := &daemonServer{
		port: port,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		pending:  map[string]chan PluginExportResult{},
		shutdown: make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc(DefaultPluginPath, server.handlePlugin)
	mux.HandleFunc("/v1/export/png", server.handleExportPNG)
	mux.HandleFunc("/v1/export/svg", server.handleExportSVG)
	mux.HandleFunc("/shutdown", server.handleShutdown)

	server.http = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Fprintf(os.Stderr, "%s daemon listening on http://%s\n", ServiceName, listenAddr)
	err = server.http.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *daemonServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	s.mu.Lock()
	connected := s.plugin != nil
	pending := len(s.pending)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, HealthResponse{
		OK:              true,
		Name:            ServiceName,
		PluginConnected: connected,
		PendingTasks:    pending,
	})
}

func (s *daemonServer) handlePlugin(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.mu.Lock()
	if s.plugin != nil {
		_ = s.plugin.Close()
	}
	s.plugin = conn
	s.mu.Unlock()

	for {
		var result PluginExportResult
		if err := conn.ReadJSON(&result); err != nil {
			break
		}
		s.completeTask(result)
	}

	s.mu.Lock()
	if s.plugin == conn {
		s.plugin = nil
	}
	s.mu.Unlock()
	_ = conn.Close()
}

func (s *daemonServer) handleExportPNG(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var request ExportPNGRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := validateExportPNG(request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Fill platform default scales if not specified.
	if len(request.Scales) == 0 {
		scales, ok := defaultScales(request.Platform)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported platform: " + request.Platform})
			return
		}
		request.Scales = scales
	}

	result, err := s.exportPNG(request)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *daemonServer) handleExportSVG(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var request ExportSVGRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := validateExportSVG(request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := s.exportSVG(request)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *daemonServer) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	go s.stop()
}

func (s *daemonServer) exportPNG(request ExportPNGRequest) (ExportPNGResponse, error) {
	if err := s.waitForPlugin(30 * time.Second); err != nil {
		return ExportPNGResponse{}, err
	}

	taskID := randomID()
	resultCh := make(chan PluginExportResult, 1)

	s.mu.Lock()
	s.pending[taskID] = resultCh
	conn := s.plugin
	s.mu.Unlock()

	if conn == nil {
		s.removePending(taskID)
		return ExportPNGResponse{}, errors.New("Figma plugin is not connected")
	}

	task := PluginTask{
		ID:      taskID,
		Version: 1,
		Action:  "figma.exportNodePng",
		Payload: ExportNodePNGPayload{
			NodeID:       request.NodeID,
			Scales:       request.Scales,
			ContentsOnly: true,
		},
	}
	s.writeMu.Lock()
	err := conn.WriteJSON(task)
	s.writeMu.Unlock()
	if err != nil {
		s.removePending(taskID)
		return ExportPNGResponse{}, err
	}

	select {
	case result := <-resultCh:
		if !result.OK {
			if result.Error != "" {
				return ExportPNGResponse{}, errors.New(result.Error)
			}
			if len(result.Errors) > 0 {
				msgs := make([]string, len(result.Errors))
				for i, e := range result.Errors {
					msgs[i] = e.Message
				}
				return ExportPNGResponse{}, fmt.Errorf("Figma plugin export failed: %s", strings.Join(msgs, "; "))
			}
			return ExportPNGResponse{}, errors.New("Figma plugin export failed")
		}
		fileName := request.FileName
		if fileName == "" {
			fileName = result.Result.NodeName
		}
		files, err := writePNG(request.Platform, request.OutDir, fileName, result.Result.Exports)
		if err != nil {
			return ExportPNGResponse{}, err
		}
		return ExportPNGResponse{
			OK:        true,
			Operation: "export.png",
			Files:     files,
			Errors:    result.Errors,
		}, nil
	case <-time.After(time.Duration(DefaultTaskTimeoutMS) * time.Millisecond):
		s.removePending(taskID)
		return ExportPNGResponse{}, errors.New("timed out waiting for Figma plugin")
	}
}

func (s *daemonServer) exportSVG(request ExportSVGRequest) (ExportSVGResponse, error) {
	if err := s.waitForPlugin(30 * time.Second); err != nil {
		return ExportSVGResponse{}, err
	}

	taskID := randomID()
	resultCh := make(chan PluginExportResult, 1)

	s.mu.Lock()
	s.pending[taskID] = resultCh
	conn := s.plugin
	s.mu.Unlock()

	if conn == nil {
		s.removePending(taskID)
		return ExportSVGResponse{}, errors.New("Figma plugin is not connected")
	}

	task := PluginTask{
		ID:      taskID,
		Version: 1,
		Action:  "figma.exportNodeSvg",
		Payload: ExportNodeSVGSettings{
			NodeID:         request.NodeID,
			OutlineText:    request.OutlineText,
			IncludeIds:     request.IncludeIds,
			SimplifyStroke: request.SimplifyStroke,
			ContentsOnly:   true,
		},
	}
	s.writeMu.Lock()
	err := conn.WriteJSON(task)
	s.writeMu.Unlock()
	if err != nil {
		s.removePending(taskID)
		return ExportSVGResponse{}, err
	}

	select {
	case result := <-resultCh:
		if !result.OK {
			if result.Error != "" {
				return ExportSVGResponse{}, errors.New(result.Error)
			}
			if len(result.Errors) > 0 {
				msgs := make([]string, len(result.Errors))
				for i, e := range result.Errors {
					msgs[i] = e.Message
				}
				return ExportSVGResponse{}, fmt.Errorf("Figma plugin export failed: %s", strings.Join(msgs, "; "))
			}
			return ExportSVGResponse{}, errors.New("Figma plugin export failed")
		}
		fileName := request.FileName
		if fileName == "" {
			fileName = result.Result.NodeName
		}
		files, err := writeSVG(request.OutDir, fileName, result.Result.Exports)
		if err != nil {
			return ExportSVGResponse{}, err
		}
		return ExportSVGResponse{
			OK:        true,
			Operation: "export.svg",
			Files:     files,
			Errors:    result.Errors,
		}, nil
	case <-time.After(time.Duration(DefaultTaskTimeoutMS) * time.Millisecond):
		s.removePending(taskID)
		return ExportSVGResponse{}, errors.New("timed out waiting for Figma plugin")
	}
}

func (s *daemonServer) waitForPlugin(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		connected := s.plugin != nil
		s.mu.Unlock()
		if connected {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return errors.New("Figma plugin is not connected. Open Plugins -> Development -> Figma Asset")
}

func (s *daemonServer) completeTask(result PluginExportResult) {
	s.mu.Lock()
	resultCh := s.pending[result.ID]
	delete(s.pending, result.ID)
	s.mu.Unlock()
	if resultCh != nil {
		resultCh <- result
	}
}

func (s *daemonServer) removePending(taskID string) {
	s.mu.Lock()
	delete(s.pending, taskID)
	s.mu.Unlock()
}

func (s *daemonServer) stop() {
	s.shutOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.http.Shutdown(ctx)
		close(s.shutdown)
	})
}

func validateExportPNG(request ExportPNGRequest) error {
	if request.NodeID == "" {
		return errors.New("nodeId is required")
	}
	if request.OutDir == "" {
		return errors.New("outDir is required")
	}
	if request.Platform == "" {
		return errors.New("platform is required")
	}
	if !isValidPlatform(request.Platform) {
		return fmt.Errorf("unsupported platform: %s (use: flutter, android, ios, or web)", request.Platform)
	}
	for _, scale := range request.Scales {
		if scale <= 0 {
			return fmt.Errorf("invalid scale: %v", scale)
		}
	}
	return nil
}

func validateExportSVG(request ExportSVGRequest) error {
	if request.NodeID == "" {
		return errors.New("nodeId is required")
	}
	if request.OutDir == "" {
		return errors.New("outDir is required")
	}
	if request.Platform == "" {
		return errors.New("platform is required")
	}
	if !isValidPlatform(request.Platform) {
		return fmt.Errorf("unsupported platform: %s (use: flutter, android, ios, or web)", request.Platform)
	}
	return nil
}

func randomID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(bytes[:])
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
