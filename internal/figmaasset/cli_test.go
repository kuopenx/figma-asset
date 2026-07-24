package figmaasset

import (
	"fmt"
	"path/filepath"
	"testing"
)

// --- parseNodeAndName ---

func TestParseNodeAndNameSingle(t *testing.T) {
	ids, names, err := parseNodeAndName("2005:709", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "2005:709" {
		t.Fatalf("ids = %v, want [2005:709]", ids)
	}
	if len(names) != 0 {
		t.Fatalf("names = %v, want empty", names)
	}
}

func TestParseNodeAndNameSingleWithName(t *testing.T) {
	ids, names, err := parseNodeAndName("2005:709", "icon_home")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "2005:709" {
		t.Fatalf("ids = %v, want [2005:709]", ids)
	}
	if len(names) != 1 || names[0] != "icon_home" {
		t.Fatalf("names = %v, want [icon_home]", names)
	}
}

func TestParseNodeAndNameBatch(t *testing.T) {
	ids, names, err := parseNodeAndName("1:1,2:2,3:3", "a,b,c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 3 || ids[0] != "1:1" || ids[1] != "2:2" || ids[2] != "3:3" {
		t.Fatalf("ids = %v, want [1:1 2:2 3:3]", ids)
	}
	if len(names) != 3 || names[0] != "a" || names[1] != "b" || names[2] != "c" {
		t.Fatalf("names = %v, want [a b c]", names)
	}
}

func TestParseNodeAndNameBatchNoNames(t *testing.T) {
	ids, names, err := parseNodeAndName("1:1,2:2,3:3", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("ids = %v, want 3 entries", ids)
	}
	if len(names) != 0 {
		t.Fatalf("names = %v, want empty", names)
	}
}

func TestParseNodeAndNameCountMismatch(t *testing.T) {
	_, _, err := parseNodeAndName("1:1,2:2", "a")
	if err == nil {
		t.Fatal("expected error for count mismatch")
	}
}

func TestParseNodeAndNameTrimsSpaces(t *testing.T) {
	ids, names, err := parseNodeAndName(" 1:1 , 2:2 ", " a , b ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ids[0] != "1:1" || ids[1] != "2:2" {
		t.Fatalf("ids = %v, want trimmed", ids)
	}
	if names[0] != "a" || names[1] != "b" {
		t.Fatalf("names = %v, want trimmed", names)
	}
}

// --- parseScales ---

func TestParseScalesValid(t *testing.T) {
	scales, err := parseScales("1,2,3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scales) != 3 || scales[0] != 1 || scales[1] != 2 || scales[2] != 3 {
		t.Fatalf("scales = %v, want [1 2 3]", scales)
	}
}

func TestParseScalesSingle(t *testing.T) {
	scales, err := parseScales("2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scales) != 1 || scales[0] != 2 {
		t.Fatalf("scales = %v, want [2]", scales)
	}
}

func TestParseScalesWithSpaces(t *testing.T) {
	scales, err := parseScales("1, 2, 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scales) != 3 {
		t.Fatalf("scales = %v, want 3 entries", scales)
	}
}

func TestParseScalesDecimal(t *testing.T) {
	scales, err := parseScales("1,1.5,2,3,4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scales) != 5 || scales[1] != 1.5 {
		t.Fatalf("scales = %v, want [1 1.5 2 3 4]", scales)
	}
}

func TestParseScalesInvalid(t *testing.T) {
	_, err := parseScales("1,abc,3")
	if err == nil {
		t.Fatal("expected error for invalid scale")
	}
}

func TestParseScalesZero(t *testing.T) {
	_, err := parseScales("0")
	if err == nil {
		t.Fatal("expected error for zero scale")
	}
}

func TestParseScalesNegative(t *testing.T) {
	_, err := parseScales("-1")
	if err == nil {
		t.Fatal("expected error for negative scale")
	}
}

// --- validateExportFlags ---

func TestValidateExportFlagsAllPresent(t *testing.T) {
	err := validateExportFlags("flutter", "2005:709")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateExportFlagsMissingPlatform(t *testing.T) {
	err := validateExportFlags("", "2005:709")
	if err == nil {
		t.Fatal("expected error for missing platform")
	}
}

func TestValidateExportFlagsMissingNode(t *testing.T) {
	err := validateExportFlags("flutter", "")
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}

func TestValidateExportFlagsInvalidPlatform(t *testing.T) {
	err := validateExportFlags("react", "2005:709")
	if err == nil {
		t.Fatal("expected error for invalid platform")
	}
}

func TestValidateExportFlagsAllMissing(t *testing.T) {
	err := validateExportFlags("", "")
	if err == nil {
		t.Fatal("expected error for all flags missing")
	}
}

// --- isValidPlatform ---

func TestIsValidPlatform(t *testing.T) {
	valid := []string{"flutter", "android", "ios", "web"}
	for _, p := range valid {
		if !isValidPlatform(p) {
			t.Errorf("isValidPlatform(%q) = false, want true", p)
		}
	}
}

func TestIsValidPlatformInvalid(t *testing.T) {
	invalid := []string{"", "react", "desktop", "windows", "Flutter", "IOS"}
	for _, p := range invalid {
		if isValidPlatform(p) {
			t.Errorf("isValidPlatform(%q) = true, want false", p)
		}
	}
}

// --- defaultScales ---

func TestDefaultScalesFlutter(t *testing.T) {
	s, ok := defaultScales("flutter")
	if !ok {
		t.Fatal("expected ok for flutter")
	}
	if len(s) != 3 || s[0] != 1 || s[1] != 2 || s[2] != 3 {
		t.Fatalf("flutter scales = %v, want [1 2 3]", s)
	}
}

func TestDefaultScalesAndroid(t *testing.T) {
	s, ok := defaultScales("android")
	if !ok {
		t.Fatal("expected ok for android")
	}
	if len(s) != 5 || s[0] != 1 || s[1] != 1.5 || s[2] != 2 || s[3] != 3 || s[4] != 4 {
		t.Fatalf("android scales = %v, want [1 1.5 2 3 4]", s)
	}
}

func TestDefaultScalesIOS(t *testing.T) {
	s, ok := defaultScales("ios")
	if !ok {
		t.Fatal("expected ok for ios")
	}
	if len(s) != 3 || s[0] != 1 || s[1] != 2 || s[2] != 3 {
		t.Fatalf("ios scales = %v, want [1 2 3]", s)
	}
}

func TestDefaultScalesWeb(t *testing.T) {
	s, ok := defaultScales("web")
	if !ok {
		t.Fatal("expected ok for web")
	}
	if len(s) != 1 || s[0] != 2 {
		t.Fatalf("web scales = %v, want [2]", s)
	}
}

func TestDefaultScalesUnknown(t *testing.T) {
	_, ok := defaultScales("unknown")
	if ok {
		t.Fatal("expected not ok for unknown platform")
	}
}

// --- daemonListenAddress ---

func TestDaemonListenAddress(t *testing.T) {
	addr := daemonListenAddress(3849)
	if addr != "127.0.0.1:3849" {
		t.Fatalf("daemonListenAddress(3849) = %q, want 127.0.0.1:3849", addr)
	}
}

func TestDaemonListenAddressZero(t *testing.T) {
	addr := daemonListenAddress(0)
	if addr != "127.0.0.1:0" {
		t.Fatalf("daemonListenAddress(0) = %q, want 127.0.0.1:0", addr)
	}
}

// --- runExportBatch (single-node path) ---

func TestRunExportBatchSingle(t *testing.T) {
	calls := 0
	err := runExportBatch([]string{"1:1"}, []string{"icon"},
		func(nodeID, name string) ([]string, error) {
			calls++
			if nodeID != "1:1" || name != "icon" {
				t.Fatalf("fn called with (%q, %q), want (1:1, icon)", nodeID, name)
			}
			return []string{"./icon.png", "./2.0x/icon.png"}, nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("fn called %d times, want 1", calls)
	}
}

func TestRunExportBatchSingleNoName(t *testing.T) {
	err := runExportBatch([]string{"1:1"}, nil,
		func(nodeID, name string) ([]string, error) {
			if name != "" {
				t.Fatalf("name = %q, want empty", name)
			}
			return []string{"./node.png"}, nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunExportBatchMultiple(t *testing.T) {
	err := runExportBatch([]string{"1:1", "2:2", "3:3"}, []string{"a", "b", "c"},
		func(nodeID, name string) ([]string, error) {
			return []string{"./" + name + ".png"}, nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunExportBatchMultipleWithFailure(t *testing.T) {
	err := runExportBatch([]string{"1:1", "2:2", "3:3"}, nil,
		func(nodeID, name string) ([]string, error) {
			if nodeID == "2:2" {
				return nil, fmt.Errorf("node not found")
			}
			return []string{"./ok.png"}, nil
		})
	if err == nil {
		t.Fatal("expected error when one node fails")
	}
}

func TestRunExportBatchAllFail(t *testing.T) {
	err := runExportBatch([]string{"1:1", "2:2"}, nil,
		func(nodeID, name string) ([]string, error) {
			return nil, fmt.Errorf("export failed")
		})
	if err == nil {
		t.Fatal("expected error when all nodes fail")
	}
}

// --- validateExportPNG / validateExportSVG (daemon-side) ---

func TestValidateExportPNGValid(t *testing.T) {
	err := validateExportPNG(ExportPNGRequest{
		NodeID:   "2005:709",
		OutDir:   "/tmp/assets",
		Platform: "flutter",
		Scales:   []float64{1, 2, 3},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateExportPNGMissingNode(t *testing.T) {
	err := validateExportPNG(ExportPNGRequest{
		OutDir:   "/tmp/assets",
		Platform: "flutter",
	})
	if err == nil {
		t.Fatal("expected error for missing nodeId")
	}
}

func TestValidateExportPNGMissingOutDir(t *testing.T) {
	err := validateExportPNG(ExportPNGRequest{
		NodeID:   "2005:709",
		Platform: "flutter",
	})
	if err == nil {
		t.Fatal("expected error for missing outDir")
	}
}

func TestValidateExportPNGMissingPlatform(t *testing.T) {
	err := validateExportPNG(ExportPNGRequest{
		NodeID: "2005:709",
		OutDir: "/tmp/assets",
	})
	if err == nil {
		t.Fatal("expected error for missing platform")
	}
}

func TestValidateExportPNGInvalidPlatform(t *testing.T) {
	err := validateExportPNG(ExportPNGRequest{
		NodeID:   "2005:709",
		OutDir:   "/tmp/assets",
		Platform: "react",
	})
	if err == nil {
		t.Fatal("expected error for invalid platform")
	}
}

func TestValidateExportPNGNegativeScale(t *testing.T) {
	err := validateExportPNG(ExportPNGRequest{
		NodeID:   "2005:709",
		OutDir:   "/tmp/assets",
		Platform: "flutter",
		Scales:   []float64{1, -2},
	})
	if err == nil {
		t.Fatal("expected error for negative scale")
	}
}

func TestValidateExportSVGValid(t *testing.T) {
	err := validateExportSVG(ExportSVGRequest{
		NodeID:   "2005:709",
		OutDir:   "/tmp/assets",
		Platform: "flutter",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateExportSVGMissingNode(t *testing.T) {
	err := validateExportSVG(ExportSVGRequest{
		OutDir:   "/tmp/assets",
		Platform: "flutter",
	})
	if err == nil {
		t.Fatal("expected error for missing nodeId")
	}
}

func TestValidateExportSVGInvalidPlatform(t *testing.T) {
	err := validateExportSVG(ExportSVGRequest{
		NodeID:   "2005:709",
		OutDir:   "/tmp/assets",
		Platform: "desktop",
	})
	if err == nil {
		t.Fatal("expected error for invalid platform")
	}
}

// --- resolveOutputDir ---

func TestResolveOutputDirProjectDir(t *testing.T) {
	for _, tc := range []struct {
		platform string
		subDir   string
	}{
		{"flutter", "assets/images"},
		{"android", "app/src/main/res"},
		{"ios", "Assets.xcassets"},
		{"web", "public/assets"},
	} {
		t.Run(tc.platform, func(t *testing.T) {
			got, err := resolveOutputDir("/myproject", "", tc.platform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := filepath.Join("/myproject", tc.subDir)
			if got != want {
				t.Fatalf("resolveOutputDir(%q) = %q, want %q", tc.platform, got, want)
			}
		})
	}
}

func TestResolveOutputDirOutDir(t *testing.T) {
	got, err := resolveOutputDir("", "/custom/icons", "flutter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/custom/icons" {
		t.Fatalf("got %q, want /custom/icons", got)
	}
}

func TestResolveOutputDirBothError(t *testing.T) {
	_, err := resolveOutputDir("/myproject", "/custom", "flutter")
	if err == nil {
		t.Fatal("expected error when both project-dir and out-dir are set")
	}
}

func TestResolveOutputDirNeitherError(t *testing.T) {
	_, err := resolveOutputDir("", "", "flutter")
	if err == nil {
		t.Fatal("expected error when neither project-dir nor out-dir is set")
	}
}

func TestResolveOutputDirInvalidPlatform(t *testing.T) {
	_, err := resolveOutputDir("/myproject", "", "react")
	if err == nil {
		t.Fatal("expected error for invalid platform with project-dir")
	}
}
