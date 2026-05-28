package cli

type clientCommandMode string

const (
	clientCommandLocal       clientCommandMode = "local"
	clientCommandRPC         clientCommandMode = "rpc"
	clientCommandUnsupported clientCommandMode = "unsupported"
)

var clientCommandModes = map[string]clientCommandMode{
	"version": clientCommandLocal,

	"audio.process-voice": clientCommandRPC,
	"audio.tts":           clientCommandRPC,
	"content.cache.get":   clientCommandRPC,
	"content.cache.put":   clientCommandRPC,
	"doctor":              clientCommandRPC,
	"event.list":          clientCommandRPC,
	"event.log":           clientCommandRPC,
	"learner.state":       clientCommandRPC,
	"report.due":          clientCommandRPC,
	"report.hsk":          clientCommandRPC,
	"report.mastery":      clientCommandRPC,
	"review.bury":         clientCommandRPC,
	"review.grade":        clientCommandRPC,
	"review.start":        clientCommandRPC,
	"snapshot":            clientCommandRPC,
	"srs.preview":         clientCommandRPC,

	"audio.convert":               clientCommandUnsupported,
	"cram.audio-backfill":         clientCommandUnsupported,
	"cram.grade":                  clientCommandUnsupported,
	"cram.next":                   clientCommandUnsupported,
	"cram.overview":               clientCommandUnsupported,
	"dataset.import":              clientCommandUnsupported,
	"db.init":                     clientCommandUnsupported,
	"gc":                          clientCommandUnsupported,
	"hellochinese.audio-backfill": clientCommandUnsupported,
	"hellochinese.import":         clientCommandUnsupported,
	"pleco.score-import":          clientCommandUnsupported,
	"travel.import":               clientCommandUnsupported,
}

func clientCommandModeForID(commandID string) (clientCommandMode, bool) {
	if commandID == "web.serve" {
		return clientCommandUnsupported, true
	}
	mode, ok := clientCommandModes[commandID]
	return mode, ok
}

func runClientBacked(args []string, serverURL string) int {
	commandID, ok := commandIDFromArgs(args)
	if !ok {
		printUsage()
		return 1
	}
	mode, ok := clientCommandModeForID(commandID)
	if !ok {
		printUsage()
		return 1
	}
	switch mode {
	case clientCommandRPC:
		return runClientRPC(commandID, args, serverURL)
	case clientCommandUnsupported:
		return emitUnsupportedClientCommand(commandID, serverURL)
	default:
		printUsage()
		return 1
	}
}

func commandIDFromArgs(args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	switch args[0] {
	case "version", "snapshot", "doctor", "gc":
		return args[0], true
	case "learner":
		return subcommandID(args, "learner", "state")
	case "db":
		return subcommandID(args, "db", "init")
	case "dataset":
		return subcommandID(args, "dataset", "import")
	case "hellochinese":
		return subcommandID(args, "hellochinese", "import", "audio-backfill")
	case "travel":
		return subcommandID(args, "travel", "import")
	case "pleco":
		return subcommandID(args, "pleco", "score-import")
	case "review":
		return subcommandID(args, "review", "start", "grade", "bury")
	case "cram":
		return subcommandID(args, "cram", "overview", "audio-backfill", "next", "grade")
	case "srs":
		return subcommandID(args, "srs", "preview")
	case "report":
		return subcommandID(args, "report", "hsk", "mastery", "due")
	case "event":
		return subcommandID(args, "event", "log", "list")
	case "content":
		if len(args) >= 3 && args[1] == "cache" && (args[2] == "put" || args[2] == "get") {
			return "content.cache." + args[2], true
		}
		return "", false
	case "audio":
		return subcommandID(args, "audio", "tts", "process-voice", "convert")
	case "web":
		return subcommandID(args, "web", "serve")
	default:
		return "", false
	}
}

func subcommandID(args []string, parent string, allowed ...string) (string, bool) {
	if len(args) < 2 {
		return "", false
	}
	for _, sub := range allowed {
		if args[1] == sub {
			return parent + "." + sub, true
		}
	}
	return "", false
}

func emitUnsupportedClientCommand(commandID, serverURL string) int {
	return emitTypedError(
		commandID,
		"UNSUPPORTED_CLIENT_COMMAND",
		"command is not available in xuezh client-backed mode",
		map[string]any{"server_url": serverURL},
	)
}
