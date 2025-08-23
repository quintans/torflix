package model

type CacheData struct {
	OriginalQuery string `json:"original_query"`
	Provider      string `json:"provider"`
	Name          string `json:"name"`
	Size          string `json:"size"`
	Seeds         string `json:"seeds"`
	Quality       string `json:"quality"`
	Magnet        string `json:"magnet"`
	Hash          string `json:"hash"`
}
