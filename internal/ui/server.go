package ui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"elon/waver/internal/app"
)

//go:embed static/*
var staticFiles embed.FS

type combineRequest struct {
	Path       string `json:"path"`
	OutputRoot string `json:"outputRoot"`
}

type selectDirectoryRequest struct {
	Title string `json:"title"`
}

type selectDirectoryResponse struct {
	OK    bool   `json:"ok"`
	Path  string `json:"path,omitempty"`
	Error string `json:"error,omitempty"`
}

type apiResponse struct {
	OK     bool               `json:"ok"`
	Result *app.CombineResult `json:"result,omitempty"`
	Error  string             `json:"error,omitempty"`
}

func Run(ctx context.Context, addr string, open bool) error {
	listener, err := listen(addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", indexHandler)
	mux.HandleFunc("GET /static/", staticHandler)
	mux.HandleFunc("POST /api/combine", combineHandler)
	mux.HandleFunc("POST /api/select-directory", selectDirectoryHandler)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	url := browserURL(listener.Addr())
	fmt.Printf("Waver UI: %s\n", url)

	if open {
		go func() {
			time.Sleep(200 * time.Millisecond)
			if err := openURL(url); err != nil {
				fmt.Printf("Could not open browser automatically: %v\n", err)
			}
		}()
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func listen(addr string) (net.Listener, error) {
	if strings.TrimSpace(addr) == "" {
		addr = "127.0.0.1:4040"
	}

	listener, err := net.Listen("tcp", addr)
	if err == nil {
		return listener, nil
	}

	if addr == "127.0.0.1:4040" {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	return nil, err
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, staticFiles, "static/index.html")
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	http.FileServerFS(staticFiles).ServeHTTP(w, r)
}

func combineHandler(w http.ResponseWriter, r *http.Request) {
	var request combineRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "Некорректный запрос."})
		return
	}

	path := strings.TrimSpace(request.Path)
	if path == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "Не указана папка проекта."})
		return
	}

	outputRoot := strings.TrimSpace(request.OutputRoot)
	if outputRoot == "" {
		outputRoot = "out"
	}

	result, err := app.Combine(r.Context(), path, outputRoot)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Result: result})
}

func selectDirectoryHandler(w http.ResponseWriter, r *http.Request) {
	var request selectDirectoryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writePathJSON(w, http.StatusBadRequest, selectDirectoryResponse{OK: false, Error: "Некорректный запрос."})
		return
	}

	path, err := selectDirectory(request.Title)
	if err != nil {
		if errors.Is(err, errDirectorySelectionCanceled) {
			writePathJSON(w, http.StatusOK, selectDirectoryResponse{OK: false, Error: "selection canceled"})
			return
		}
		writePathJSON(w, http.StatusBadRequest, selectDirectoryResponse{OK: false, Error: err.Error()})
		return
	}

	writePathJSON(w, http.StatusOK, selectDirectoryResponse{OK: true, Path: path})
}

func writeJSON(w http.ResponseWriter, status int, response apiResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writePathJSON(w http.ResponseWriter, status int, response selectDirectoryResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func browserURL(addr net.Addr) string {
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return "http://" + addr.String()
	}

	host := tcpAddr.IP.String()
	if tcpAddr.IP.IsLoopback() {
		host = "localhost"
	}

	return fmt.Sprintf("http://%s:%d", host, tcpAddr.Port)
}

func openURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

func selectDirectory(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Выберите папку"
	}

	switch runtime.GOOS {
	case "darwin":
		return selectDirectoryDarwin(title)
	case "linux":
		return selectDirectoryLinux(title)
	case "windows":
		return selectDirectoryWindows(title)
	default:
		return "", fmt.Errorf("выбор папки не поддерживается для %s", runtime.GOOS)
	}
}

func selectDirectoryDarwin(title string) (string, error) {
	script := fmt.Sprintf("POSIX path of (choose folder with prompt %s)", quoteAppleScriptString(title))
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg != "" && (strings.Contains(msg, "User canceled") || strings.Contains(msg, "Canceled") || strings.Contains(msg, "-128")) {
			return "", errDirectorySelectionCanceled
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return strings.TrimSuffix(msg, "\n"), nil
}

func selectDirectoryWindows(title string) (string, error) {
	script := "Add-Type -AssemblyName System.Windows.Forms; $dialog = New-Object System.Windows.Forms.FolderBrowserDialog; $dialog.Description = '" + quotePowershellString(title) + "'; $result = $dialog.ShowDialog(); if ($result -eq 'OK') { $dialog.SelectedPath }"
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg != "" && (strings.Contains(strings.ToLower(msg), "canceled") || strings.Contains(msg, "Close")) {
			return "", errDirectorySelectionCanceled
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return strings.TrimSuffix(msg, "\n"), nil
}

func selectDirectoryLinux(title string) (string, error) {
	out, err := exec.Command("zenity", "--file-selection", "--directory", "--title="+title).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		return selectDirectoryLinuxFallback(title, msg, err)
	}
	return strings.TrimSuffix(msg, "\n"), nil
}

func selectDirectoryLinuxFallback(title, fallbackMessage string, fallbackError error) (string, error) {
	_ = fallbackMessage
	_ = fallbackError
	out, err := exec.Command("kdialog", "--getexistingdirectory", "--title", title).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("не удалось открыть диалог выбора папки: %w", err)
	}
	return strings.TrimSuffix(msg, "\n"), nil
}

func quoteAppleScriptString(input string) string {
	escaped := strings.ReplaceAll(input, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func quotePowershellString(input string) string {
	escaped := strings.ReplaceAll(input, "`", "``")
	escaped = strings.ReplaceAll(escaped, "'", "''")
	return escaped
}

var errDirectorySelectionCanceled = errors.New("directory selection was canceled")
