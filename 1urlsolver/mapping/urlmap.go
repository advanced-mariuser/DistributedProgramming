package mapping

import (
	"encoding/json"
	"os"
)

type PathsJSON struct {
	Paths map[string]string `json:"paths"`
}

func LoadMapping(filePath string) (map[string]string, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var data PathsJSON
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}

	if data.Paths == nil {
		return make(map[string]string), nil
	}

	return data.Paths, nil
}

func SaveMapping(filePath string, mappings map[string]string) error {
	data := PathsJSON{
		Paths: mappings,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0666)
}
