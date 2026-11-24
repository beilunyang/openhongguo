package cliapp

import (
	"strconv"
	"strings"
)

type flags struct {
	json      bool
	url       bool
	cursor    string
	channel   string
	tab       string
	quality   string
	codec     string
	player    string
	download  string
	key       string
	noDecrypt bool
	autoplay  bool
	resume    bool
	pick      bool
	filters   []string
	episodes  string
	dir       string
	jobs      int
	all       bool
	pages     int
	activity  string
	list      string
	cell      string
	offset    int
	// offsetSet records an explicit --offset, which single-page reads refuse.
	offsetSet bool
	groups    bool
	shorts    bool
	panel     bool
	content   string
	board     string
	date      string
	genre     string
	// episodeSet records an explicit --episode, which --resume excludes.
	episodeSet bool
	preview    int
	episode    int
	pageSize   int
	rest       []string
}

func parseFlags(args []string) (flags, error) {
	f := flags{pageSize: 20, episode: 1, jobs: 3}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			f.rest = append(f.rest, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "--") {
			f.rest = append(f.rest, a)
			continue
		}
		name, val, has := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		switch name {
		case "json":
			if has {
				return flags{}, usage("--json takes no value")
			}
			f.json = true
		case "url":
			if has {
				return flags{}, usage("--url takes no value")
			}
			f.url = true
		case "episode":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--episode needs a value")
				}
				i++
				val = args[i]
			}
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return flags{}, usage("episode must be a positive integer")
			}
			f.episode = n
			f.episodeSet = true
		case "codec":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--codec needs a value")
				}
				i++
				val = args[i]
			}
			f.codec = strings.TrimSpace(val)
			if f.codec == "" {
				return flags{}, usage("--codec needs a value")
			}
		case "quality":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--quality needs a value")
				}
				i++
				val = args[i]
			}
			f.quality = strings.TrimSpace(val)
		case "player":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--player needs a value")
				}
				i++
				val = args[i]
			}
			f.player = strings.TrimSpace(val)
		case "download":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--download needs a path")
				}
				i++
				val = args[i]
			}
			if strings.TrimSpace(val) == "" {
				return flags{}, usage("--download needs a path")
			}
			f.download = val
		case "preview":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--preview needs a duration in seconds")
				}
				i++
				val = args[i]
			}
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return flags{}, usage("preview must be a positive number of seconds")
			}
			f.preview = n
		case "filter":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--filter needs key=value")
				}
				i++
				val = args[i]
			}
			f.filters = append(f.filters, val)
		case "panel":
			if has {
				return flags{}, usage("--panel takes no value")
			}
			f.panel = true
		case "content", "board", "genre", "date":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--%s needs a value", name)
				}
				i++
				val = args[i]
			}
			if strings.TrimSpace(val) == "" {
				return flags{}, usage("--%s needs a value", name)
			}
			switch name {
			case "content":
				f.content = val
			case "board":
				f.board = val
			case "date":
				f.date = val
			default:
				f.genre = val
			}
		case "pick":
			if has {
				return flags{}, usage("--pick takes no value")
			}
			f.pick = true
		case "episodes", "dir":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--%s needs a value", name)
				}
				i++
				val = args[i]
			}
			if strings.TrimSpace(val) == "" {
				return flags{}, usage("--%s needs a value", name)
			}
			if name == "episodes" {
				f.episodes = val
			} else {
				f.dir = val
			}
		case "jobs":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--jobs needs a value")
				}
				i++
				val = args[i]
			}
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 || n > 16 {
				return flags{}, usage("jobs must be between 1 and 16")
			}
			f.jobs = n
		case "resume":
			if has {
				return flags{}, usage("--resume takes no value")
			}
			f.resume = true
		case "autoplay":
			if has {
				return flags{}, usage("--autoplay takes no value")
			}
			f.autoplay = true
		case "no-decrypt":
			if has {
				return flags{}, usage("--no-decrypt takes no value")
			}
			f.noDecrypt = true
		case "key":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--key needs a hex content key")
				}
				i++
				val = args[i]
			}
			f.key = strings.TrimSpace(val)
			if f.key == "" {
				return flags{}, usage("--key needs a hex content key")
			}
		case "cursor":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--cursor needs a value")
				}
				i++
				val = args[i]
			}
			f.cursor = val
		case "channel":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--channel needs a value")
				}
				i++
				val = args[i]
			}
			f.channel = val
		case "tab":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--tab needs a value")
				}
				i++
				val = args[i]
			}
			f.tab = val
		case "shorts":
			if has {
				return flags{}, usage("--shorts takes no value")
			}
			f.shorts = true
		case "groups":
			if has {
				return flags{}, usage("--groups takes no value")
			}
			f.groups = true
		case "cell":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--cell needs an id")
				}
				i++
				val = args[i]
			}
			f.cell = val
		case "offset":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--offset needs a number")
				}
				i++
				val = args[i]
			}
			n, err := strconv.Atoi(val)
			if err != nil || n < 0 {
				return flags{}, usage("offset must be a non-negative integer")
			}
			f.offset, f.offsetSet = n, true
		case "activity":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--activity needs an id")
				}
				i++
				val = args[i]
			}
			f.activity = val
		case "list":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--list needs an id")
				}
				i++
				val = args[i]
			}
			f.list = val
		case "all":
			if has {
				return flags{}, usage("--all takes no value")
			}
			f.all = true
		case "pages":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--pages needs a value")
				}
				i++
				val = args[i]
			}
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return flags{}, usage("pages must be a positive integer")
			}
			f.pages = n
		case "page-size":
			if !has {
				if i+1 >= len(args) {
					return flags{}, usage("--page-size needs a value")
				}
				i++
				val = args[i]
			}
			n, err := strconv.Atoi(val)
			if err != nil || n < 0 {
				return flags{}, usage("page-size must be a non-negative integer")
			}
			f.pageSize = n
		default:
			return flags{}, usage("unknown flag --%s", name)
		}
	}
	return f, nil
}
