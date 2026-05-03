package audio

import "testing"

func TestBuildTTSCommandPassesNegativeRateAsSingleArg(t *testing.T) {
	cmd := buildTTSCommand("你是龙大。", "zh-CN-XiaoxiaoNeural", "-23%", "out.mp3")
	for _, arg := range cmd {
		if arg == "--rate" {
			t.Fatalf("negative edge-tts rates must be passed as --rate=-23%%, got separate args: %#v", cmd)
		}
	}
	found := false
	for _, arg := range cmd {
		if arg == "--rate=-23%" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing rate arg: %#v", cmd)
	}
}
