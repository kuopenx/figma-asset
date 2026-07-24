package figmaasset

import (
	"net"
	"strconv"
)

const (
	ServiceName          = "figma-asset"
	DefaultPort          = 3849
	DefaultPluginPath    = "/plugin"
	DefaultTaskTimeoutMS = 120000
)

func daemonListenAddress(port int) string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
}

// Platform identifiers.
const (
	PlatformFlutter = "flutter"
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
	PlatformWeb     = "web"
)

func isValidPlatform(platform string) bool {
	switch platform {
	case PlatformFlutter, PlatformAndroid, PlatformIOS, PlatformWeb:
		return true
	default:
		return false
	}
}

// platformDefaultScales is the recommended export scales per platform.
var platformDefaultScales = map[string][]float64{
	PlatformFlutter: {1, 2, 3},
	PlatformAndroid: {1, 1.5, 2, 3, 4},
	PlatformIOS:     {1, 2, 3},
	PlatformWeb:     {2},
}

func defaultScales(platform string) ([]float64, bool) {
	s, ok := platformDefaultScales[platform]
	return s, ok
}

// --- Health ---

type HealthResponse struct {
	OK              bool   `json:"ok"`
	Name            string `json:"name"`
	PluginConnected bool   `json:"pluginConnected"`
	PendingTasks    int    `json:"pendingTasks"`
}

// --- PNG export ---

type ExportPNGRequest struct {
	NodeID   string    `json:"nodeId"`
	OutDir   string    `json:"outDir"`
	FileName string    `json:"fileName"` // optional; empty = use node name
	Platform string    `json:"platform"`
	Scales   []float64 `json:"scales"` // optional; empty = platform defaults
}

type ExportPNGResponse struct {
	OK        bool          `json:"ok"`
	Operation string        `json:"operation"`
	Files     []string      `json:"files"`
	Errors    []PluginError `json:"errors,omitempty"`
}

// --- SVG export ---

type ExportSVGRequest struct {
	NodeID         string `json:"nodeId"`
	OutDir         string `json:"outDir"`
	FileName       string `json:"fileName"` // optional; empty = use node name
	Platform       string `json:"platform"`
	OutlineText    bool   `json:"outlineText"`
	IncludeIds     bool   `json:"includeIds"`
	SimplifyStroke bool   `json:"simplifyStroke"`
}

type ExportSVGResponse struct {
	OK        bool          `json:"ok"`
	Operation string        `json:"operation"`
	Files     []string      `json:"files"`
	Errors    []PluginError `json:"errors,omitempty"`
}

// --- Plugin protocol ---

type PluginTask struct {
	ID      string      `json:"id"`
	Version int         `json:"version"`
	Action  string      `json:"action"`
	Payload interface{} `json:"payload"`
}

type ExportNodePNGPayload struct {
	NodeID       string    `json:"nodeId"`
	Scales       []float64 `json:"scales"`
	ContentsOnly bool      `json:"contentsOnly"`
}

type ExportNodeSVGSettings struct {
	NodeID         string `json:"nodeId"`
	OutlineText    bool   `json:"outlineText"`
	IncludeIds     bool   `json:"includeIds"`
	SimplifyStroke bool   `json:"simplifyStroke"`
	ContentsOnly   bool   `json:"contentsOnly"`
}

type PluginExportResult struct {
	ID     string              `json:"id"`
	OK     bool                `json:"ok"`
	Result PluginExportPayload `json:"result"`
	Errors []PluginError       `json:"errors,omitempty"`
	Error  string              `json:"error,omitempty"`
}

type PluginExportPayload struct {
	Exports  []PluginExportBytes `json:"exports"`
	NodeName string              `json:"nodeName,omitempty"`
}

type PluginExportBytes struct {
	Scale  float64 `json:"scale"`
	Format string  `json:"format"`
	Bytes  []int   `json:"bytes"`
}

type PluginError struct {
	NodeID  string  `json:"nodeId,omitempty"`
	Scale   float64 `json:"scale,omitempty"`
	Message string  `json:"message"`
}
