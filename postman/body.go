package postman

type BodyMode string

const (
	BodyModeForm       BodyMode = "formdata"
	BodyModeURLEncoded BodyMode = "urlencoded"
	BodyModeRaw        BodyMode = "raw"
)

type BodyOptions struct {
	Raw *RawOptions `json:"raw,omitempty"`
}

type RawOptions struct {
	Language string `json:"language"`
}

type Body struct {
	Mode       BodyMode     `json:"mode"`
	Raw        string       `json:"raw,omitempty"`
	Urlencoded []KeyValue   `json:"urlencoded,omitempty"`
	FormData   []KeyValue   `json:"formdata,omitempty"`
	File       *File        `json:"file,omitempty"`
	Options    *BodyOptions `json:"options,omitempty"`
}

func (b *Body) SetLanguage(s string) {
	b.Options = &BodyOptions{
		Raw: &RawOptions{Language: "json"},
	}
}

type KeyValue struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Disabled    bool   `json:"disabled"`
}

type File struct {
	Src     string `json:"src,omitempty"`
	Content string `json:"content,omitempty"`
}
