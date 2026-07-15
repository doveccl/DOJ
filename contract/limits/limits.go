package limits

const (
	BodyShortText      = "256K"
	BodyMarkdown       = "2M"
	BodySource         = "1M"
	BodyLanguage       = "128K"
	BodySettings       = "2M"
	BodyImage          = "6M"
	BodyProblemPackage = "520M"
	BodyJudgerResult   = "2M"
)

const (
	MaxMarkdownBytes          = 2 << 20
	MaxSourceBytes            = 1 << 20
	MaxShortTextBytes         = 256 << 10
	MaxLanguageCommandBytes   = 128 << 10
	DefaultLanguageCompileMS  = 10_000
	MaxLanguageCompileMS      = 30_000
	MaxJudgerMessageBytes     = 256 << 10
	MaxCaseMessageRunes       = 1024
	MaxProblemTimeMS          = 60_000
	MaxProblemMemoryMB        = 4096
	MaxOutstandingSubmissions = 5
	MaxJudgerCases            = 10_000
	MaxJudgerLeaseBytes       = 32 << 20
	MaxJudgerOutputKB         = 64 << 10
	MaxJudgerPids             = 256
	DefaultJudgerFileKB       = 256 << 10
	MaxJudgerFileKB           = 1 << 20
	MaxPasswordBytes          = 72
)
