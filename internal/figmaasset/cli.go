package figmaasset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Run(args []string, version string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "start":
		return runStart()
	case "status":
		return runStatus()
	case "restart":
		return runRestart()
	case "export":
		return runExport(args[1:])
	case "version":
		fmt.Println(version)
		return nil
	case "upgrade":
		return runUpgrade(args[1:], version)
	case "plugin-path":
		return runPluginPath()
	case "stop":
		return runStop()
	case "daemon":
		return RunDaemon(DefaultPort)
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printUsage() {
	fmt.Println(`figma-asset

Usage:
  figma-asset start
  figma-asset status
  figma-asset restart
  figma-asset export png  --platform <flutter|android|ios|web> --node <node-id> --project-dir <dir> [--name <name>] [--scales <1,2,3>]
  figma-asset export png  --platform <flutter|android|ios|web> --node <node-id> --out-dir <dir>    [--name <name>] [--scales <1,2,3>]
  figma-asset export svg  --platform <flutter|android|ios|web> --node <node-id> --project-dir <dir> [--name <name>] [svg-options]
  figma-asset export svg  --platform <flutter|android|ios|web> --node <node-id> --out-dir <dir>    [--name <name>] [svg-options]
  figma-asset version
  figma-asset plugin-path
  figma-asset upgrade [--check]
  figma-asset stop

Commands:
  start    Start the local daemon and keep it running until stop.
  status   Show daemon and Figma plugin connection status.
  restart  Restart the local daemon, or start it if it is not running.
  export   Export Figma node as image assets (png or svg).
  version      Print the current version.
  plugin-path  Print the Figma plugin manifest path for import.
  upgrade  Download and install the latest release. Use --check to only check.
  stop     Stop the local daemon.

Output directory:
  --project-dir  Project root; subdirectory is auto-appended per platform convention.
  --out-dir      Specific directory; files are written directly there.
  Only one of --project-dir / --out-dir may be used.

Run "figma-asset export png --help" or "figma-asset export svg --help" for options.`)
}

// --- export ---

func printExportUsage() {
	fmt.Println(`figma-asset export

Usage:
  figma-asset export png  --platform <flutter|android|ios|web> --node <node-id> --project-dir <dir> [--name <name>] [--scales <1,2,3>]
  figma-asset export png  --platform <flutter|android|ios|web> --node <node-id> --out-dir <dir>    [--name <name>] [--scales <1,2,3>]
  figma-asset export svg  --platform <flutter|android|ios|web> --node <node-id> --project-dir <dir> [--name <name>] [svg-options]
  figma-asset export svg  --platform <flutter|android|ios|web> --node <node-id> --out-dir <dir>    [--name <name>] [svg-options]

Commands:
  png  Export one Figma node as PNG assets with platform-specific directory layout.
  svg  Export one Figma node as a single SVG file.

Output directory:
  --project-dir  Project root; subdirectory is auto-appended per platform convention.
  --out-dir      Specific directory; files are written directly there.
  Only one of --project-dir / --out-dir may be used.

Run "figma-asset export png --help" or "figma-asset export svg --help" for options.`)
}

func printExportPNGUsage() {
	fmt.Println(`figma-asset export png

Usage:
  figma-asset export png --platform <flutter|android|ios|web> --node <node-id> --project-dir <dir> [--name <name>] [--scales <1,2,3>]
  figma-asset export png --platform <flutter|android|ios|web> --node <node-id> --out-dir <dir>    [--name <name>] [--scales <1,2,3>]

Required:
  --platform     Target platform: flutter, android, ios, or web.
  --node         Figma node id, for example 2005:709. URL node-id=2005-709 maps to 2005:709.
  --project-dir  Project root directory; subdirectory is auto-appended per platform convention.
  --out-dir      Specific output directory; files are written directly there.
  Only one of --project-dir / --out-dir may be used.

Optional:
  --name      Output file name without extension. Defaults to the Figma node name.
  --scales    Comma-separated export scales. Defaults to platform recommendation:
                flutter: 1,2,3
                android: 1,1.5,2,3,4
                ios:     1,2,3
                web:     2

Platform output layout (with --project-dir):
  flutter  <project>/assets/images/name.png, 2.0x/name.png, 3.0x/name.png
  android  <project>/app/src/main/res/drawable-mdpi/name.png, drawable-hdpi/name.png, ...
  ios      <project>/Assets.xcassets/name.imageset/name.png, name@2x.png, name@3x.png, Contents.json
  web      <project>/public/assets/name@2x.png

Example:
  figma-asset export png \
    --platform flutter \
    --node 2005:709 \
    --project-dir ~/my_flutter_app \
    --name im_group_notice_arrow_icon

  figma-asset export png \
    --platform flutter \
    --node 2005:709 \
    --out-dir ./custom/icons \
    --name im_group_notice_arrow_icon`)
}

func printExportSVGUsage() {
	fmt.Println(`figma-asset export svg

Usage:
  figma-asset export svg --platform <flutter|android|ios|web> --node <node-id> --project-dir <dir> [--name <name>] [svg-options]
  figma-asset export svg --platform <flutter|android|ios|web> --node <node-id> --out-dir <dir>    [--name <name>] [svg-options]

Required:
  --platform     Target platform: flutter, android, ios, or web.
  --node         Figma node id, for example 2005:709.
  --project-dir  Project root directory; subdirectory is auto-appended per platform convention.
  --out-dir      Specific output directory; SVG is written directly as <out-dir>/name.svg.
  Only one of --project-dir / --out-dir may be used.

Optional:
  --name            Output file name without extension. Defaults to the Figma node name.
  --outline-text    Render text as vector outlines. Default: true. Use --outline-text=false to keep <text> elements.
  --include-ids     Include layer names as id attributes in the SVG. Default: false.
  --simplify-stroke Simplify inside/outside stroke rendering. Default: true. Use --simplify-stroke=false for precision.

Example:
  figma-asset export svg \
    --platform flutter \
    --node 2005:709 \
    --project-dir ~/my_flutter_app \
    --name im_group_notice_arrow_icon`)
}

func runExport(args []string) error {
	if len(args) == 0 {
		printExportUsage()
		return nil
	}

	switch args[0] {
	case "png":
		return runExportPNG(args[1:])
	case "svg":
		return runExportSVG(args[1:])
	case "help", "-h", "--help":
		printExportUsage()
		return nil
	default:
		return fmt.Errorf("unknown export command: %s", args[0])
	}
}

func runExportPNG(args []string) error {
	fs := flag.NewFlagSet("export png", flag.ContinueOnError)
	fs.Usage = printExportPNGUsage
	platform := fs.String("platform", "", "target platform: flutter, android, ios, or web")
	nodeID := fs.String("node", "", "Figma node id (comma-separated for batch)")
	projectDir := fs.String("project-dir", "", "project root directory (auto-appends platform subdirectory)")
	outDirFlag := fs.String("out-dir", "", "specific output directory (writes directly)")
	fileName := fs.String("name", "", "output file name (defaults to Figma node name; comma-separated for batch)")
	scalesText := fs.String("scales", "", "comma-separated export scales (defaults to platform recommendation)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	absOut, err := resolveOutputDir(*projectDir, *outDirFlag, *platform)
	if err != nil {
		return err
	}

	if err := validateExportFlags(*platform, *nodeID); err != nil {
		return err
	}

	nodeIDs, names, err := parseNodeAndName(*nodeID, *fileName)
	if err != nil {
		return err
	}

	var scales []float64
	if *scalesText != "" {
		scales, err = parseScales(*scalesText)
		if err != nil {
			return err
		}
	}

	if err := ensureDaemon(); err != nil {
		return err
	}

	health, err := getHealth()
	if err == nil && !health.PluginConnected {
		fmt.Println("Waiting for Figma plugin connection. Open Figma -> Plugins -> Development -> Figma Asset.")
	}

	return runExportBatch(nodeIDs, names, func(nodeID, name string) ([]string, error) {
		request := ExportPNGRequest{
			NodeID:   nodeID,
			OutDir:   absOut,
			FileName: name,
			Platform: *platform,
			Scales:   scales,
		}
		response, err := postJSON[ExportPNGResponse]("/v1/export/png", request)
		if err != nil {
			return nil, err
		}
		return response.Files, nil
	})
}

func runExportSVG(args []string) error {
	fs := flag.NewFlagSet("export svg", flag.ContinueOnError)
	fs.Usage = printExportSVGUsage
	platform := fs.String("platform", "", "target platform: flutter, android, ios, or web")
	nodeID := fs.String("node", "", "Figma node id (comma-separated for batch)")
	projectDir := fs.String("project-dir", "", "project root directory (auto-appends platform subdirectory)")
	outDirFlag := fs.String("out-dir", "", "specific output directory (writes directly)")
	fileName := fs.String("name", "", "output file name (defaults to Figma node name; comma-separated for batch)")
	outlineText := fs.Bool("outline-text", true, "render text as vector outlines")
	includeIds := fs.Bool("include-ids", false, "include layer names as id attributes")
	simplifyStroke := fs.Bool("simplify-stroke", true, "simplify stroke rendering")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	absOut, err := resolveOutputDir(*projectDir, *outDirFlag, *platform)
	if err != nil {
		return err
	}

	if err := validateExportFlags(*platform, *nodeID); err != nil {
		return err
	}

	nodeIDs, names, err := parseNodeAndName(*nodeID, *fileName)
	if err != nil {
		return err
	}

	if err := ensureDaemon(); err != nil {
		return err
	}

	health, err := getHealth()
	if err == nil && !health.PluginConnected {
		fmt.Println("Waiting for Figma plugin connection. Open Figma -> Plugins -> Development -> Figma Asset.")
	}

	return runExportBatch(nodeIDs, names, func(nodeID, name string) ([]string, error) {
		request := ExportSVGRequest{
			NodeID:         nodeID,
			OutDir:         absOut,
			FileName:       name,
			Platform:       *platform,
			OutlineText:    *outlineText,
			IncludeIds:     *includeIds,
			SimplifyStroke: *simplifyStroke,
		}
		response, err := postJSON[ExportSVGResponse]("/v1/export/svg", request)
		if err != nil {
			return nil, err
		}
		return response.Files, nil
	})
}

func parseNodeAndName(nodeID, name string) ([]string, []string, error) {
	nodeIDs := strings.Split(nodeID, ",")
	for i, id := range nodeIDs {
		nodeIDs[i] = strings.TrimSpace(id)
	}
	var names []string
	if name != "" {
		names = strings.Split(name, ",")
		for i, n := range names {
			names[i] = strings.TrimSpace(n)
		}
		if len(names) != len(nodeIDs) {
			return nil, nil, fmt.Errorf("--name has %d entries but --node has %d", len(names), len(nodeIDs))
		}
	}
	return nodeIDs, names, nil
}

type batchExportFunc func(nodeID, name string) ([]string, error)

func runExportBatch(nodeIDs, names []string, fn batchExportFunc) error {
	if len(nodeIDs) == 1 {
		name := ""
		if len(names) > 0 {
			name = names[0]
		}
		files, err := fn(nodeIDs[0], name)
		if err != nil {
			return err
		}
		for _, f := range files {
			fmt.Println(f)
		}
		return nil
	}

	total := len(nodeIDs)
	fmt.Printf("Exporting %d nodes...\n", total)

	const concurrency = 5
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	type batchResult struct {
		nodeID string
		files  []string
		err    error
	}
	results := make(chan batchResult, total)

	for i, id := range nodeIDs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			name := ""
			if len(names) > 0 {
				name = names[i]
			}
			files, err := fn(id, name)
			results <- batchResult{nodeID: id, files: files, err: err}
		}(i, id)
	}

	go func() { wg.Wait(); close(results) }()

	done := 0
	var allFiles []string
	var errorCount int
	for r := range results {
		done++
		if r.err != nil {
			fmt.Printf("[%d/%d] %s: failed: %v\n", done, total, r.nodeID, r.err)
			errorCount++
		} else {
			fmt.Printf("[%d/%d] %s: %d files\n", done, total, r.nodeID, len(r.files))
			allFiles = append(allFiles, r.files...)
		}
	}

	fmt.Printf("\nDone. %d nodes, %d files, %d errors.\n", total, len(allFiles), errorCount)
	if errorCount > 0 {
		return fmt.Errorf("%d node(s) failed", errorCount)
	}
	return nil
}

func resolveOutputDir(projectDir, outDir, platform string) (string, error) {
	if projectDir != "" && outDir != "" {
		return "", errors.New("--project-dir and --out-dir cannot be used together")
	}
	if projectDir == "" && outDir == "" {
		return "", errors.New("--project-dir or --out-dir is required")
	}
	if projectDir != "" {
		subDir, ok := platformSubDirs[platform]
		if !ok {
			return "", fmt.Errorf("unsupported platform: %s", platform)
		}
		return filepath.Abs(filepath.Join(projectDir, subDir))
	}
	return filepath.Abs(outDir)
}

func validateExportFlags(platform string, nodeID string) error {
	var missing []string
	if platform == "" {
		missing = append(missing, "--platform is required: flutter, android, ios, or web")
	} else if !isValidPlatform(platform) {
		missing = append(missing, fmt.Sprintf("--platform %q is not valid; choose flutter, android, ios, or web", platform))
	}
	if nodeID == "" {
		missing = append(missing, "--node is required: choose the Figma node to export, e.g. 2005:709")
	}
	if len(missing) == 0 {
		return nil
	}
	return errors.New(strings.Join(missing, "\n") + "\n\nExample:\n  figma-asset export png --platform flutter --node 2005:709 --project-dir ~/my_app\n  figma-asset export svg --platform flutter --node 2005:709 --out-dir ./custom/icons")
}

// --- start / status / restart / stop ---

func runStart() error {
	if err := ensureDaemon(); err != nil {
		return err
	}
	health, err := getHealth()
	if err != nil {
		return err
	}
	printHealth(health)
	return nil
}

func runStatus() error {
	health, err := getHealth()
	if err != nil {
		state, owned, stateErr := ownedDaemonState()
		if stateErr != nil {
			return stateErr
		}
		if owned {
			return fmt.Errorf("figma-asset daemon pid %d is running but not responding", state.PID)
		}
		if isDaemonPortOccupied() {
			return fmt.Errorf("port %d is occupied by a process that is not a healthy %s daemon", DefaultPort, ServiceName)
		}
		fmt.Println("figma-asset daemon is not running.")
		return nil
	}
	if health.Name != ServiceName {
		return fmt.Errorf("port %d is used by %q, not %s", DefaultPort, health.Name, ServiceName)
	}
	printHealth(health)
	return nil
}

func runRestart() error {
	if _, err := shutdownDaemonIfRunning(); err != nil {
		return err
	}
	if err := ensureDaemon(); err != nil {
		return err
	}
	health, err := getHealth()
	if err != nil {
		return err
	}
	printHealth(health)
	return nil
}

func runStop() error {
	stopped, err := shutdownDaemonIfRunning()
	if err != nil {
		return err
	}
	if !stopped {
		fmt.Println("figma-asset daemon is not running.")
		return nil
	}
	fmt.Println("figma-asset daemon stopped.")
	return nil
}

func runUpgrade(args []string, currentVersion string) error {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	check := fs.Bool("check", false, "check for updates without installing")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	latest, err := FetchLatestVersion()
	if err != nil {
		return err
	}

	if *check {
		fmt.Printf("Current: %s\n", currentVersion)
		fmt.Printf("Latest:  %s\n", latest)
		if IsUpdateAvailable(currentVersion, latest) {
			fmt.Println("Update available. Run \"figma-asset upgrade\" to update.")
		} else {
			fmt.Println("Already up to date.")
		}
		return nil
	}

	return RunUpgrade(currentVersion)
}

func pluginDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "figma-asset-plugin"
	}
	return filepath.Join(home, "figma-asset-plugin")
}

func runPluginPath() error {
	manifestPath := filepath.Join(pluginDir(), "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		fmt.Fprintf(os.Stderr, "manifest.json not found at %s\nRun install first: curl -fsSL https://raw.githubusercontent.com/kuopenx/figma-asset/main/install.sh | sh\n", manifestPath)
		return err
	}
	fmt.Println(manifestPath)
	return nil
}

func shutdownDaemonIfRunning() (bool, error) {
	health, err := getHealth()
	if err == nil {
		if health.Name != ServiceName {
			return false, fmt.Errorf("port %d is used by %q, not %s", DefaultPort, health.Name, ServiceName)
		}

		if _, err := postJSON[map[string]bool]("/shutdown", map[string]bool{}); err != nil {
			return false, err
		}

		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := getHealth(); err != nil {
				return true, nil
			}
			time.Sleep(100 * time.Millisecond)
		}
		return false, fmt.Errorf("timed out waiting for figma-asset daemon to stop")
	}

	state, owned, err := ownedDaemonState()
	if err != nil {
		return false, err
	}
	if owned {
		if err := terminateProcess(state.PID); err != nil && !isProcessNotFound(err) {
			return false, fmt.Errorf("terminate unresponsive figma-asset daemon (pid %d): %w", state.PID, err)
		}
		if waitForOwnedDaemonExit(state, 2*time.Second) {
			removeDaemonState(state.Nonce)
			return true, nil
		}
		if err := forceKillProcess(state.PID); err != nil && !isProcessNotFound(err) {
			return false, fmt.Errorf("force kill unresponsive figma-asset daemon (pid %d): %w", state.PID, err)
		}
		if waitForOwnedDaemonExit(state, time.Second) {
			removeDaemonState(state.Nonce)
			return true, nil
		}
		return false, fmt.Errorf("timed out waiting for unresponsive figma-asset daemon (pid %d) to stop", state.PID)
	}

	if isDaemonPortOccupied() {
		return false, fmt.Errorf("port %d is occupied by a process that is not a healthy %s daemon", DefaultPort, ServiceName)
	}
	return false, nil
}

func ensureDaemon() error {
	health, err := getHealth()
	if err == nil {
		if health.Name != ServiceName {
			return fmt.Errorf("port %d is used by %q, not %s", DefaultPort, health.Name, ServiceName)
		}
		return nil
	}
	state, owned, err := ownedDaemonState()
	if err != nil {
		return err
	}
	if owned {
		return fmt.Errorf("figma-asset daemon pid %d is running but not responding; run \"figma-asset stop\" to recover it", state.PID)
	}
	if isDaemonPortOccupied() {
		return fmt.Errorf("port %d is occupied by a process that is not a healthy %s daemon", DefaultPort, ServiceName)
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	logFile, err := openDaemonLog()
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command(exe, "daemon")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	setSysProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	exitCh := make(chan error, 1)
	go func() {
		exitCh <- cmd.Wait()
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		health, err := getHealth()
		if err == nil && health.Name == ServiceName {
			return nil
		}
		select {
		case err := <-exitCh:
			if err == nil {
				return fmt.Errorf("figma-asset daemon exited before becoming ready; see %s", daemonLogPath())
			}
			return fmt.Errorf("figma-asset daemon exited before becoming ready: %w; see %s", err, daemonLogPath())
		default:
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("figma-asset daemon did not become ready; see %s", daemonLogPath())
}

func openDaemonLog() (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(daemonLogPath()), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(daemonLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}

func daemonLogPath() string {
	return filepath.Join(pluginDir(), "daemon.log")
}

func getHealth() (HealthResponse, error) {
	client := http.Client{Timeout: 800 * time.Millisecond}
	response, err := client.Get(baseURL() + "/health")
	if err != nil {
		return HealthResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return HealthResponse{}, fmt.Errorf("health returned %s", response.Status)
	}
	var health HealthResponse
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil {
		return HealthResponse{}, err
	}
	return health, nil
}

func postJSON[T any](path string, body any) (T, error) {
	var zero T
	payload, err := json.Marshal(body)
	if err != nil {
		return zero, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute+10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL()+path, bytes.NewReader(payload))
	if err != nil {
		return zero, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return zero, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var errorBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(response.Body).Decode(&errorBody)
		if errorBody.Error != "" {
			return zero, errors.New(errorBody.Error)
		}
		return zero, fmt.Errorf("request failed: %s", response.Status)
	}

	var decoded T
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return zero, err
	}
	return decoded, nil
}

func baseURL() string {
	return "http://" + daemonListenAddress(DefaultPort)
}

func parseScales(input string) ([]float64, error) {
	parts := strings.Split(input, ",")
	scales := make([]float64, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		scale, err := strconv.ParseFloat(value, 64)
		if err != nil || scale <= 0 {
			return nil, fmt.Errorf("invalid scale: %s", value)
		}
		scales = append(scales, scale)
	}
	if len(scales) == 0 {
		return nil, errors.New("at least one scale is required")
	}
	return scales, nil
}

func printHealth(health HealthResponse) {
	fmt.Printf("name: %s\n", health.Name)
	fmt.Printf("listen: http://127.0.0.1:%d\n", DefaultPort)
	fmt.Printf("plugin: %s\n", connectionStatus(health.PluginConnected))
	fmt.Printf("pendingTasks: %d\n", health.PendingTasks)
}

func connectionStatus(connected bool) string {
	if connected {
		return "connected"
	}
	return "disconnected"
}
