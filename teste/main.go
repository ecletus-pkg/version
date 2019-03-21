package main

import (
	"os"

	"github.com/aghape-pkg/version"
)

func main() {
	v := version.Version{}
	v.Assets = map[string]*version.Version{
		"Teste": {},
		"Teste2": {
			Assets: map[string]*version.Version{
				"B":{},
				"A":{},
			},
		},
	}

	data, err := v.MarshalJSONIndent("", "  ")
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(data)
}
