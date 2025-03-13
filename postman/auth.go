package postman

type AuthType string

const (
	AuthTypeNone     AuthType = "none"
	AuthTypeBasic    AuthType = "basic"
	AuthTypeBearer   AuthType = "bearer"
	AuthTypeDigest   AuthType = "digest"
	AuthTypeEdgeGrid AuthType = "edgegrid"
	AuthTypeHawk     AuthType = "hawk"
	AuthTypeOAuth1   AuthType = "oauth1"
	AuthTypeOAuth2   AuthType = "oauth2"
	AuthTypeNTLM     AuthType = "ntlm"
)

type Auth struct {
	Type   AuthType    `json:"type"`
	OAuth2 *OAuth2Auth `json:"oauth2,omitempty"`
	APIKey []KeyValue  `json:"apikey,omitempty"`
}

type OAuth2Auth struct {
	GrantType   string `json:"grant_type"`
	AccessToken string `json:"accessToken"`
}
