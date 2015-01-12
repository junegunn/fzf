package fzf

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/junegunn/go-shellwords"
)

const usage = `usage: fzf [options]

  Search
    -x, --extended        Extended-search mode
    -e, --extended-exact  Extended-search mode (exact match)
    -i                    Case-insensitive match (default: smart-case match)
    +i                    Case-sensitive match
    -n, --nth=N[,..]      Comma-separated list of field index expressions
                          for limiting search scope. Each can be a non-zero
                          integer or a range expression ([BEGIN]..[END])
        --with-nth=N[,..] Transform the item using index expressions for search
    -d, --delimiter=STR   Field delimiter regex for --nth (default: AWK-style)

  Search result
    -s, --sort            Sort the result
    +s, --no-sort         Do not sort the result. Keep the sequence unchanged.

  Interface
    -m, --multi           Enable multi-select with tab/shift-tab
        --no-mouse        Disable mouse
    +c, --no-color        Disable colors
    +2, --no-256          Disable 256-color
        --black           Use black background
        --reverse         Reverse orientation
        --prompt=STR      Input prompt (default: '> ')

  Scripting
    -q, --query=STR       Start the finder with the given query
    -1, --select-1        Automatically select the only match
    -0, --exit-0          Exit immediately when there's no match
    -f, --filter=STR      Filter mode. Do not start interactive finder.
        --print-query     Print query as the first line

  Environment variables
    FZF_DEFAULT_COMMAND   Default command to use when input is tty
    FZF_DEFAULT_OPTS      Defaults options. (e.g. "-x -m")

`

// Mode denotes the current search mode
type Mode int

// Search modes
const (
	ModeFuzzy Mode = iota
	ModeExtended
	ModeExtendedExact
)

// Case denotes case-sensitivity of search
type Case int

// Case-sensitivities
const (
	CaseSmart Case = iota
	CaseIgnore
	CaseRespect
)

// Options stores the values of command-line options
type Options struct {
	Mode       Mode
	Case       Case
	Nth        []Range
	WithNth    []Range
	Delimiter  *regexp.Regexp
	Sort       int
	Multi      bool
	Mouse      bool
	Color      bool
	Color256   bool
	Black      bool
	Reverse    bool
	Prompt     string
	Query      string
	Select1    bool
	Exit0      bool
	Filter     *string
	PrintQuery bool
	Version    bool
}

func defaultOptions() *Options {
	return &Options{
		Mode:       ModeFuzzy,
		Case:       CaseSmart,
		Nth:        make([]Range, 0),
		WithNth:    make([]Range, 0),
		Delimiter:  nil,
		Sort:       1000,
		Multi:      false,
		Mouse:      true,
		Color:      true,
		Color256:   strings.Contains(os.Getenv("TERM"), "256"),
		Black:      false,
		Reverse:    false,
		Prompt:     "> ",
		Query:      "",
		Select1:    false,
		Exit0:      false,
		Filter:     nil,
		PrintQuery: false,
		Version:    false}
}

func help(ok int) {
	os.Stderr.WriteString(usage)
	os.Exit(ok)
}

func errorExit(msg string) {
	os.Stderr.WriteString(msg + "\n")
	help(1)
}

func optString(arg string, prefix string) (bool, string) {
	rx, _ := regexp.Compile(fmt.Sprintf("^(?:%s)(.*)$", prefix))
	matches := rx.FindStringSubmatch(arg)
	if len(matches) > 1 {
		return true, matches[1]
	}
	return false, ""
}

func nextString(args []string, i *int, message string) string {
	if len(args) > *i+1 {
		*i++
	} else {
		errorExit(message)
	}
	return args[*i]
}

func optionalNumeric(args []string, i *int) int {
	if len(args) > *i+1 {
		if strings.IndexAny(args[*i+1], "0123456789") == 0 {
			*i++
		}
	}
	return 1 // Don't care
}

func splitNth(str string) []Range {
	if match, _ := regexp.MatchString("^[0-9,-.]+$", str); !match {
		errorExit("invalid format: " + str)
	}

	tokens := strings.Split(str, ",")
	ranges := make([]Range, len(tokens))
	for idx, s := range tokens {
		r, ok := ParseRange(&s)
		if !ok {
			errorExit("invalid format: " + str)
		}
		ranges[idx] = r
	}
	return ranges
}

func delimiterRegexp(str string) *regexp.Regexp {
	rx, e := regexp.Compile(str)
	if e != nil {
		str = regexp.QuoteMeta(str)
	}

	rx, e = regexp.Compile(fmt.Sprintf("(?:.*?%s)|(?:.+?$)", str))
	if e != nil {
		errorExit("invalid regular expression: " + e.Error())
	}
	return rx
}

func parseOptions(opts *Options, allArgs []string) {
	for i := 0; i < len(allArgs); i++ {
		arg := allArgs[i]
		switch arg {
		case "-h", "--help":
			help(0)
		case "-x", "--extended":
			opts.Mode = ModeExtended
		case "-e", "--extended-exact":
			opts.Mode = ModeExtendedExact
		case "+x", "--no-extended", "+e", "--no-extended-exact":
			opts.Mode = ModeFuzzy
		case "-q", "--query":
			opts.Query = nextString(allArgs, &i, "query string required")
		case "-f", "--filter":
			filter := nextString(allArgs, &i, "query string required")
			opts.Filter = &filter
		case "-d", "--delimiter":
			opts.Delimiter = delimiterRegexp(nextString(allArgs, &i, "delimiter required"))
		case "-n", "--nth":
			opts.Nth = splitNth(nextString(allArgs, &i, "nth expression required"))
		case "--with-nth":
			opts.WithNth = splitNth(nextString(allArgs, &i, "nth expression required"))
		case "-s", "--sort":
			opts.Sort = optionalNumeric(allArgs, &i)
		case "+s", "--no-sort":
			opts.Sort = 0
		case "-i":
			opts.Case = CaseIgnore
		case "+i":
			opts.Case = CaseRespect
		case "-m", "--multi":
			opts.Multi = true
		case "+m", "--no-multi":
			opts.Multi = false
		case "--no-mouse":
			opts.Mouse = false
		case "+c", "--no-color":
			opts.Color = false
		case "+2", "--no-256":
			opts.Color256 = false
		case "--black":
			opts.Black = true
		case "--no-black":
			opts.Black = false
		case "--reverse":
			opts.Reverse = true
		case "--no-reverse":
			opts.Reverse = false
		case "-1", "--select-1":
			opts.Select1 = true
		case "+1", "--no-select-1":
			opts.Select1 = false
		case "-0", "--exit-0":
			opts.Exit0 = true
		case "+0", "--no-exit-0":
			opts.Exit0 = false
		case "--print-query":
			opts.PrintQuery = true
		case "--no-print-query":
			opts.PrintQuery = false
		case "--prompt":
			opts.Prompt = nextString(allArgs, &i, "prompt string required")
		case "--version":
			opts.Version = true
		default:
			if match, value := optString(arg, "-q|--query="); match {
				opts.Query = value
			} else if match, value := optString(arg, "-f|--filter="); match {
				opts.Filter = &value
			} else if match, value := optString(arg, "-d|--delimiter="); match {
				opts.Delimiter = delimiterRegexp(value)
			} else if match, value := optString(arg, "--prompt="); match {
				opts.Prompt = value
			} else if match, value := optString(arg, "-n|--nth="); match {
				opts.Nth = splitNth(value)
			} else if match, value := optString(arg, "--with-nth="); match {
				opts.WithNth = splitNth(value)
			} else if match, _ := optString(arg, "-s|--sort="); match {
				opts.Sort = 1 // Don't care
			} else {
				errorExit("unknown option: " + arg)
			}
		}
	}
}

// ParseOptions parses command-line options
func ParseOptions() *Options {
	opts := defaultOptions()

	// Options from Env var
	words, _ := shellwords.Parse(os.Getenv("FZF_DEFAULT_OPTS"))
	parseOptions(opts, words)

	// Options from command-line arguments
	parseOptions(opts, os.Args[1:])
	return opts
}
