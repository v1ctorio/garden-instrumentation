package config

// See https://hackclub.slack.com/docs/T0266FRGM/F0AAT2FLDJS
var RecognizedEvents = map[string]string{
	"program_engaged":   "multi_entry_events",
	"wthc_message_sent": "single_entry_events",
	"introduction_sent": "single_entry_events",
}
