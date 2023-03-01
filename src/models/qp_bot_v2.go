package models

type QPBotV2 struct {
	ID              string `db:"id" json:"id"`
	Verified        bool   `db:"is_verified" json:"is_verified"`
	Token           string `db:"token" json:"token"`
	UserID          string `db:"user_id" json:"user_id"`
	CreatedAt       string `db:"created_at" json:"created_at,omitempty"`
	UpdatedAt       string `db:"updated_at" json:"updated_at,omitempty"`
	Devel           bool   `db:"devel" json:"devel"`
	Version         string `db:"version" json:"version,omitempty"`
	HandleGroups    bool   `db:"handlegroups" json:"handlegroups,omitempty"`
	HandleBroadcast bool   `db:"handlebroadcast" json:"handlebroadcast,omitempty"`
}
